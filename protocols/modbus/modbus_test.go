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
	testSlaveConfig = `protocol: modbus
node:
  ip: 127.0.0.1
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

	testMasterConfig = `protocol: modbus
messages:
  - timestamp: 1
    recurrent: false
    interval: 0
    ip: 127.0.0.1
    port: 5502
    slave_id: 1
    function_code: 3
    start_address: 0
    count: 4
  - timestamp: 2
    recurrent: false
    interval: 0
    ip: 127.0.0.1
    port: 5502
    slave_id: 1
    function_code: 1
    start_address: 0
    count: 4
  - timestamp: 3
    recurrent: false
    interval: 0
    ip: 127.0.0.1
    port: 5502
    slave_id: 1
    function_code: 6
    start_address: 0
    values: [1500]
  - timestamp: 4
    recurrent: false
    interval: 0
    ip: 127.0.0.1
    port: 5502
    slave_id: 1
    function_code: 3
    start_address: 0
    count: 4`

	testMasterConfigWithValues = `protocol: modbus
messages:
  - timestamp: 1
    recurrent: false
    interval: 0
    ip: 127.0.0.1
    port: 5502
    slave_id: 1
    function_code: 6
    start_address: 5
    values: [100]
  - timestamp: 2
    recurrent: false
    interval: 0
    ip: 127.0.0.1
    port: 5502
    slave_id: 1
    function_code: 16
    start_address: 10
    values: [200, 300, 400]
  - timestamp: 3
    recurrent: false
    interval: 0
    ip: 127.0.0.1
    port: 5502
    slave_id: 1
    function_code: 3
    start_address: 5
    count: 2`

	testMasterConfigRecurrent = `protocol: modbus
messages:
  - timestamp: 0
    recurrent: false
    interval: 0
    ip: 127.0.0.1
    port: 5502
    slave_id: 1
    function_code: 3
    start_address: 0
    count: 2
  - timestamp: 1
    recurrent: false
    interval: 0
    ip: 127.0.0.1
    port: 5502
    slave_id: 1
    function_code: 1
    start_address: 0
    count: 2`
)

type ModbusTestSuite struct {
	t                *testing.T
	tempDir          string
	slaveConfigPath  string
	masterConfigPath string
	syncFilePath     string
	slaveProcess     *exec.Cmd
	slaveReady       chan bool
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
}

func NewModbusTestSuite(t *testing.T) *ModbusTestSuite {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	os.MkdirAll(configDir, 0755)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	return &ModbusTestSuite{
		t:                t,
		tempDir:          tempDir,
		slaveConfigPath:  filepath.Join(configDir, "slave.yaml"),
		masterConfigPath: filepath.Join(configDir, "master.yaml"),
		syncFilePath:     filepath.Join(tempDir, "app_running.lock"),
		slaveReady:       make(chan bool, 1),
		ctx:              ctx,
		cancel:           cancel,
	}
}

func (ts *ModbusTestSuite) SetUp() {
	require.NoError(ts.t, os.WriteFile(ts.slaveConfigPath, []byte(testSlaveConfig), 0644))
	require.NoError(ts.t, os.WriteFile(ts.masterConfigPath, []byte(testMasterConfig), 0644))

	ts.buildBinaries()
}

func (ts *ModbusTestSuite) TearDown() {
	ts.t.Log("Starting teardown...")

	ts.cancel()

	if ts.slaveProcess != nil && ts.slaveProcess.Process != nil {
		ts.t.Log("Killing slave process...")
		ts.slaveProcess.Process.Signal(syscall.SIGTERM)

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

	os.Remove(ts.syncFilePath)
	ts.t.Log("Teardown completed")
}

func (ts *ModbusTestSuite) buildBinaries() {
	slaveDir := filepath.Join("slave")
	slaveBinary := filepath.Join(ts.tempDir, "slave")
	cmd := exec.Command("go", "build", "-o", slaveBinary, filepath.Join(slaveDir, "main.go"))
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()
	if err != nil {
		ts.t.Fatalf("Failed to build slave binary: %v\nOutput: %s", err, output)
	}

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

	ts.wg.Add(2)
	go ts.monitorOutput("slave-stdout", stdout)
	go ts.monitorOutput("slave-stderr", stderr)

	ts.wg.Add(1)
	go func() {
		defer ts.wg.Done()
		ts.slaveProcess.Wait()
		ts.t.Log("Slave process finished")
	}()

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

		if strings.Contains(line, "Server started successfully") {
			select {
			case ts.slaveReady <- true:
			default:
			}
		}
	}

	if err := scanner.Err(); err != nil {
		ts.t.Logf("%s scanner error: %v", name, err)
	}
}

func (ts *ModbusTestSuite) waitForSlaveReady() {
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

func TestModbusCommunication(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	suite := NewModbusTestSuite(t)
	suite.SetUp()
	defer suite.TearDown()

	require.NoError(t, suite.startSlave())

	assert.True(t, suite.waitForSlaveStartup(10*time.Second),
		"Slave server failed to start within timeout")

	time.Sleep(500 * time.Millisecond)

	output, err := suite.runMaster()
	t.Logf("Master output: %s", string(output))

	assert.NoError(t, err, "Master process should complete without error")

	assert.NotEmpty(t, string(output), "Master should generate output")

	outputStr := string(output)
	assert.Contains(t, outputStr, "Starting ModbusMaster", "Master should start correctly")
	assert.Contains(t, outputStr, "Sending msg to 127.0.0.1:5502", "Master should send messages")
	assert.Contains(t, outputStr, "All messages processed", "Master should complete all messages")

	assert.Contains(t, outputStr, "Loaded 4 messages", "Should load all 4 messages from YAML")

	assert.NotContains(t, outputStr, "Failed to connect", "Should not have connection failures")
	assert.NotContains(t, outputStr, "Failed to parse message", "Should not have parse failures")
}

func TestModbusSlaveStartup(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	suite := NewModbusTestSuite(t)
	suite.SetUp()
	defer suite.TearDown()

	require.NoError(t, suite.startSlave())

	assert.True(t, suite.waitForSlaveStartup(10*time.Second),
		"Slave server should start within timeout")

	_, err := os.Stat(suite.syncFilePath)
	assert.NoError(t, err, "Sync file should exist after slave startup")
}

func TestModbusReadOperations(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	readOnlyConfig := `protocol: modbus
messages:
  - timestamp: 1
    recurrent: false
    interval: 0
    ip: 127.0.0.1
    port: 5502
    slave_id: 1
    function_code: 3
    start_address: 0
    count: 2
  - timestamp: 2
    recurrent: false
    interval: 0
    ip: 127.0.0.1
    port: 5502
    slave_id: 1
    function_code: 1
    start_address: 0
    count: 2
  - timestamp: 3
    recurrent: false
    interval: 0
    ip: 127.0.0.1
    port: 5502
    slave_id: 1
    function_code: 2
    start_address: 0
    count: 2
  - timestamp: 4
    recurrent: false
    interval: 0
    ip: 127.0.0.1
    port: 5502
    slave_id: 1
    function_code: 4
    start_address: 0
    count: 2`

	suite := NewModbusTestSuite(t)
	suite.SetUp()
	defer suite.TearDown()

	require.NoError(t, os.WriteFile(suite.masterConfigPath, []byte(readOnlyConfig), 0644))

	require.NoError(t, suite.startSlave())
	assert.True(t, suite.waitForSlaveStartup(10*time.Second))
	time.Sleep(500 * time.Millisecond)

	output, err := suite.runMaster()
	t.Logf("Read operations output: %s", string(output))

	assert.NoError(t, err, "Read operations should succeed")

	outputStr := string(output)
	assert.Contains(t, outputStr, "FC:3", "Should perform holding register reads")
	assert.Contains(t, outputStr, "FC:1", "Should perform coil reads")
	assert.Contains(t, outputStr, "FC:2", "Should perform discrete input reads")
	assert.Contains(t, outputStr, "FC:4", "Should perform input register reads")
	assert.Contains(t, outputStr, "Loaded 4 messages", "Should load all 4 read messages")
}

func TestModbusValuesWithBrackets(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	suite := NewModbusTestSuite(t)
	suite.SetUp()
	defer suite.TearDown()

	require.NoError(t, os.WriteFile(suite.masterConfigPath, []byte(testMasterConfigWithValues), 0644))

	require.NoError(t, suite.startSlave())
	assert.True(t, suite.waitForSlaveStartup(10*time.Second))
	time.Sleep(500 * time.Millisecond)

	output, err := suite.runMaster()
	t.Logf("Bracketed values test output: %s", string(output))

	assert.NoError(t, err, "Master should handle bracketed values correctly")

	outputStr := string(output)

	assert.Contains(t, outputStr, "Loaded 3 messages", "Should load all 3 messages with explicit values")

	assert.NotContains(t, outputStr, "Failed to parse message", "Should not have parse failures")
	assert.NotContains(t, outputStr, "invalid syntax", "Should not have syntax errors")
	assert.NotContains(t, outputStr, "failed to parse", "Should not have YAML parse errors")

	assert.Contains(t, outputStr, "FC:6", "Should perform single register write with 100")
	assert.Contains(t, outputStr, "FC:16", "Should perform multiple register write with 200,300,400")

	assert.Contains(t, outputStr, "All messages processed", "Should process all messages")
}

func TestModbusWriteOperations(t *testing.T) {
	logrus.SetLevel(logrus.FatalLevel)

	writeConfig := `protocol: modbus
messages:
  - timestamp: 1
    recurrent: false
    interval: 0
    ip: 127.0.0.1
    port: 5502
    slave_id: 1
    function_code: 6
    start_address: 0
    values: [1500]
  - timestamp: 2
    recurrent: false
    interval: 0
    ip: 127.0.0.1
    port: 5502
    slave_id: 1
    function_code: 16
    start_address: 5
    values: [100, 200, 300]
  - timestamp: 3
    recurrent: false
    interval: 0
    ip: 127.0.0.1
    port: 5502
    slave_id: 1
    function_code: 3
    start_address: 0
    count: 3`

	suite := NewModbusTestSuite(t)
	suite.SetUp()
	defer suite.TearDown()

	require.NoError(t, os.WriteFile(suite.masterConfigPath, []byte(writeConfig), 0644))

	require.NoError(t, suite.startSlave())
	assert.True(t, suite.waitForSlaveStartup(10*time.Second))
	time.Sleep(500 * time.Millisecond)

	output, err := suite.runMaster()
	t.Logf("Write operations output: %s", string(output))

	assert.NoError(t, err, "Write operations should succeed")

	outputStr := string(output)
	assert.Contains(t, outputStr, "FC:6", "Should perform single register write")
	assert.Contains(t, outputStr, "FC:16", "Should perform multiple register write")
	assert.Contains(t, outputStr, "Loaded 3 messages", "Should load all 3 write messages")
	assert.Contains(t, outputStr, "All messages processed", "Should complete all writes")
}

func TestMain(m *testing.M) {
	code := m.Run()

	os.Exit(code)
}
