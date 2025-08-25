package modbus_test

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSlaveConfig = `ip: 127.0.0.1
port: 5502
slave_id: 1
discrete_inputs:
  type: sequential
  values: [1, 0, 1, 0]
coils:
  type: sequential
  values: [1, 1, 0, 1]
input_registers:
  type: sequential
  values: [100, 200, 300, 400]
holding_registers:
  type: sequential
  values: [1000, 2000, 3000, 4000]
identity:
  vendor_name: "Test Vendor"
  product_code: "TC"
  vendor_url: "http://test.com"
  product_name: "Test Product"
  model_name: "Test Model"
  major_minor_revision: "1.0"
  user_application_name: "Test App"`

	testMasterConfig = `timestamp,ip,port,function_code,slave_id,recurrent,interval,start_address,count,values
1,127.0.0.1,5502,3,1,false,0,0,4,
2,127.0.0.1,5502,1,1,false,0,0,4,
3,127.0.0.1,5502,6,1,false,0,0,,1500
4,127.0.0.1,5502,3,1,false,0,0,4,`
)

type ModbusTestSuite struct {
	t               *testing.T
	tempDir         string
	slaveConfigPath string
	masterConfigPath string
	syncFilePath    string
	slaveProcess    *exec.Cmd
	slaveReady      chan bool
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

func NewModbusTestSuite(t *testing.T) *ModbusTestSuite {
	tempDir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	
	return &ModbusTestSuite{
		t:                t,
		tempDir:          tempDir,
		slaveConfigPath:  filepath.Join(tempDir, "slave.yaml"),
		masterConfigPath: filepath.Join(tempDir, "master.csv"),
		syncFilePath:     filepath.Join(tempDir, "app_running.lock"),
		slaveReady:       make(chan bool, 1),
		ctx:              ctx,
		cancel:           cancel,
	}
}

func (ts *ModbusTestSuite) SetUp() {
	// Create test configuration files
	require.NoError(ts.t, os.WriteFile(ts.slaveConfigPath, []byte(testSlaveConfig), 0644))
	require.NoError(ts.t, os.WriteFile(ts.masterConfigPath, []byte(testMasterConfig), 0644))
	
	// Build the slave and master binaries if they don't exist
	ts.buildBinaries()
}

func (ts *ModbusTestSuite) TearDown() {
	ts.t.Log("Starting teardown...")
	
	// Cancel context first
	ts.cancel()
	
	// Force kill slave process if still running
	if ts.slaveProcess != nil && ts.slaveProcess.Process != nil {
		ts.t.Log("Killing slave process...")
		ts.slaveProcess.Process.Signal(syscall.SIGTERM)
		
		// Wait a bit for graceful shutdown
		done := make(chan error, 1)
		go func() {
			done <- ts.slaveProcess.Wait()
		}()
		
		select {
		case <-done:
			ts.t.Log("Slave process terminated gracefully")
		case <-time.After(2 * time.Second):
			ts.t.Log("Force killing slave process...")
			ts.slaveProcess.Process.Kill()
			ts.slaveProcess.Wait()
		}
	}
	
	// Wait for all goroutines to finish (with timeout)
	waitDone := make(chan struct{})
	go func() {
		ts.wg.Wait()
		close(waitDone)
	}()
	
	select {
	case <-waitDone:
		ts.t.Log("All goroutines finished")
	case <-time.After(3 * time.Second):
		ts.t.Log("Timeout waiting for goroutines to finish")
	}
	
	// Clean up sync file
	os.Remove(ts.syncFilePath)
	ts.t.Log("Teardown completed")
}

func (ts *ModbusTestSuite) buildBinaries() {
	// Build slave binary
	slaveDir := filepath.Join("slave")
	slaveBinary := filepath.Join(ts.tempDir, "slave")
	
	cmd := exec.Command("go", "build", "-o", slaveBinary, filepath.Join(slaveDir, "main.go"))
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()
	if err != nil {
		ts.t.Fatalf("Failed to build slave binary: %v\nOutput: %s", err, output)
	}
	
	// Build master binary
	masterDir := filepath.Join("master")
	masterBinary := filepath.Join(ts.tempDir, "master")
	
	cmd = exec.Command("go", "build", "-o", masterBinary, filepath.Join(masterDir, "main.go"))
	cmd.Dir = "."
	output, err = cmd.CombinedOutput()
	if err != nil {
		ts.t.Fatalf("Failed to build master binary: %v\nOutput: %s", err, output)
	}
}

func (ts *ModbusTestSuite) startSlave() error {
	slaveBinary := filepath.Join(ts.tempDir, "slave")
	ts.slaveProcess = exec.CommandContext(ts.ctx, slaveBinary)
	ts.slaveProcess.Dir = ts.tempDir
	
	// Set environment variables to override default file paths
	ts.slaveProcess.Env = append(os.Environ(),
		fmt.Sprintf("SLAVE_CONFIG=%s", ts.slaveConfigPath),
		fmt.Sprintf("SYNC_FILE=%s", ts.syncFilePath),
	)
	
	stdout, err := ts.slaveProcess.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	
	stderr, err := ts.slaveProcess.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}
	
	if err := ts.slaveProcess.Start(); err != nil {
		return fmt.Errorf("failed to start slave process: %v", err)
	}
	
	// Monitor output in background
	ts.wg.Add(2)
	go ts.monitorOutput("slave-stdout", stdout)
	go ts.monitorOutput("slave-stderr", stderr)
	
	// Monitor process completion
	ts.wg.Add(1)
	go func() {
		defer ts.wg.Done()
		ts.slaveProcess.Wait()
		ts.t.Log("Slave process finished")
	}()
	
	// Wait for slave to be ready (sync file created)
	go ts.waitForSlaveReady()
	
	return nil
}

func (ts *ModbusTestSuite) monitorOutput(name string, reader io.ReadCloser) {
	defer ts.wg.Done()
	defer reader.Close()
	
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		ts.t.Logf("%s: %s", name, line)
		
		// Signal when slave is ready
		if strings.Contains(line, "Server started successfully") {
			select {
			case ts.slaveReady <- true:
			default:
				// Channel already has a value
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
		ts.t.Logf("%s scanner error: %v", name, err)
	}
}

func (ts *ModbusTestSuite) waitForSlaveReady() {
	// Alternative way to check readiness via sync file
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ts.ctx.Done():
			return
		case <-ticker.C:
			if _, err := os.Stat(ts.syncFilePath); err == nil {
				select {
				case ts.slaveReady <- true:
				default:
					// Channel already has a value
				}
				return
			}
		}
	}
}

func (ts *ModbusTestSuite) runMaster() ([]byte, error) {
	masterBinary := filepath.Join(ts.tempDir, "master")
	cmd := exec.CommandContext(ts.ctx, masterBinary)
	cmd.Dir = ts.tempDir
	
	// Set environment variables to override default file paths
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("MASTER_CONFIG=%s", ts.masterConfigPath),
	)
	
	return cmd.CombinedOutput()
}

func (ts *ModbusTestSuite) waitForSlaveStartup(timeout time.Duration) bool {
	select {
	case <-ts.slaveReady:
		return true
	case <-time.After(timeout):
		return false
	case <-ts.ctx.Done():
		return false
	}
}

// Test functions

func TestModbusCommunication(t *testing.T) {
	// Disable logrus output during tests
	logrus.SetLevel(logrus.FatalLevel)
	
	suite := NewModbusTestSuite(t)
	suite.SetUp()
	defer suite.TearDown()
	
	// Start the slave server
	require.NoError(t, suite.startSlave())
	
	// Wait for slave to be ready
	assert.True(t, suite.waitForSlaveStartup(10*time.Second), 
		"Slave server failed to start within timeout")
	
	// Give a bit more time for the server to fully initialize
	time.Sleep(500 * time.Millisecond)
	
	// Run the master
	output, err := suite.runMaster()
	t.Logf("Master output: %s", string(output))
	
	// Check that master ran successfully
	assert.NoError(t, err, "Master process should complete without error")
	
	// Verify that master generated some output (indicating communication occurred)
	assert.NotEmpty(t, string(output), "Master should generate output")
	
	// Check for expected patterns in output
	outputStr := string(output)
	assert.Contains(t, outputStr, "Starting ModbusMaster", "Master should start correctly")
	assert.Contains(t, outputStr, "Sending msg to 127.0.0.1:5502", "Master should send messages")
	assert.Contains(t, outputStr, "All messages processed", "Master should complete all messages")
	
	// Verify no error patterns
	assert.NotContains(t, outputStr, "Failed to connect", "Should not have connection failures")
}

func TestModbusSlaveStartup(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)
	
	suite := NewModbusTestSuite(t)
	suite.SetUp()
	defer suite.TearDown()
	
	// Start the slave server
	require.NoError(t, suite.startSlave())
	
	// Test that slave starts successfully
	assert.True(t, suite.waitForSlaveStartup(10*time.Second),
		"Slave server should start within timeout")
	
	// Verify sync file exists
	_, err := os.Stat(suite.syncFilePath)
	assert.NoError(t, err, "Sync file should exist after slave startup")
}

func TestModbusReadOperations(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)
	
	// Create a custom master config with only read operations
	readOnlyConfig := `timestamp,ip,port,function_code,slave_id,recurrent,interval,start_address,count,values
1,127.0.0.1,5502,3,1,false,0,0,2,
2,127.0.0.1,5502,1,1,false,0,0,2,
3,127.0.0.1,5502,2,1,false,0,0,2,
4,127.0.0.1,5502,4,1,false,0,0,2,`

	suite := NewModbusTestSuite(t)
	suite.SetUp()
	defer suite.TearDown()
	
	// Override master config with read-only operations
	require.NoError(t, os.WriteFile(suite.masterConfigPath, []byte(readOnlyConfig), 0644))
	
	// Start the slave server
	require.NoError(t, suite.startSlave())
	assert.True(t, suite.waitForSlaveStartup(10*time.Second))
	
	time.Sleep(500 * time.Millisecond)
	
	// Run the master
	output, err := suite.runMaster()
	t.Logf("Read operations output: %s", string(output))
	
	assert.NoError(t, err, "Read operations should succeed")
	
	outputStr := string(output)
	assert.Contains(t, outputStr, "FC:3", "Should perform holding register reads")
	assert.Contains(t, outputStr, "FC:1", "Should perform coil reads")
	assert.Contains(t, outputStr, "FC:2", "Should perform discrete input reads")
	assert.Contains(t, outputStr, "FC:4", "Should perform input register reads")
}

func TestMain(m *testing.M) {
	// Setup code that runs before all tests
	
	// Run tests
	code := m.Run()
	
	// Cleanup code that runs after all tests
	
	os.Exit(code)
}
