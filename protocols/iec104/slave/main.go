package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wendy512/go-iecp5/asdu"
	"github.com/wendy512/go-iecp5/cs104"
	"gopkg.in/yaml.v3"
)

type DataPointConfig struct {
    Type   string      `yaml:"type"`
    Values interface{} `yaml:"values"`
}

type IdentityConfig struct {
    VendorName string `yaml:"vendor_name"`
    ModelName  string `yaml:"model_name"`
}

type SlaveConfig struct {
    IP               string          `yaml:"ip"`
    Port             interface{}     `yaml:"port"`
    StationAddr      interface{}     `yaml:"station_addr"`
    SinglePoints     DataPointConfig `yaml:"single_points"`
    DoublePoints     DataPointConfig `yaml:"double_points"`
    MeasuredValues   DataPointConfig `yaml:"measured_values"`
    IntegratedTotals DataPointConfig `yaml:"integrated_totals"`
    Identity         IdentityConfig  `yaml:"identity"`
}

// ServerHandler implements ServerHandlerInterface
type ServerHandler struct {
    slave       *IEC104Slave
    stationAddr asdu.CommonAddr
}

type IEC104Slave struct {
    config      SlaveConfig
    syncFile    string
    logger      *logrus.Logger
    server      cs104.ServerSpecial
    handler     *ServerHandler
    dataPoints  map[asdu.InfoObjAddr]interface{} // IOA -> value mapping
}

func NewIEC104Slave(configFile, syncFile string) (*IEC104Slave, error) {
    logger := logrus.New()
    logger.SetLevel(logrus.InfoLevel)

    data, err := os.ReadFile(configFile)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }

    var config SlaveConfig
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, fmt.Errorf("failed to parse config file: %w", err)
    }

    slave := &IEC104Slave{
        config:     config,
        syncFile:   syncFile,
        logger:     logger,
        dataPoints: make(map[asdu.InfoObjAddr]interface{}),
    }

    // Create handler
    slave.handler = &ServerHandler{
        slave:       slave,
        stationAddr: asdu.CommonAddr(slave.getStationAddr()),
    }

    return slave, nil
}

func (s *IEC104Slave) parseValues(dpConfig DataPointConfig) []interface{} {
    switch v := dpConfig.Values.(type) {
    case []interface{}:
        return v
    case string:
        if v == "" {
            return []interface{}{}
        }
        parts := strings.Split(v, ",")
        values := make([]interface{}, 0, len(parts))
        for _, part := range parts {
            part = strings.TrimSpace(part)
            if part != "" {
                if val, err := strconv.ParseBool(part); err == nil {
                    values = append(values, val)
                } else if val, err := strconv.ParseFloat(part, 64); err == nil {
                    values = append(values, val)
                } else {
                    values = append(values, part)
                }
            }
        }
        return values
    default:
        return []interface{}{}
    }
}

func (s *IEC104Slave) touchSyncFile() error {
    if err := os.MkdirAll(filepath.Dir(s.syncFile), 0755); err != nil {
        return err
    }
    
    file, err := os.Create(s.syncFile)
    if err != nil {
        return err
    }
    defer file.Close()
    
    s.logger.Infof("Created sync file: %s", s.syncFile)
    return nil
}

func (s *IEC104Slave) getPort() int {
    switch p := s.config.Port.(type) {
    case int:
        return p
    case string:
        if port, err := strconv.Atoi(p); err == nil {
            return port
        }
    case float64:
        return int(p)
    }
    return 2404
}

func (s *IEC104Slave) getStationAddr() int {
    switch p := s.config.StationAddr.(type) {
    case int:
        return p
    case string:
        if addr, err := strconv.Atoi(p); err == nil {
            return addr
        }
    case float64:
        return int(p)
    }
    return 1
}

func (s *IEC104Slave) start() error {
    port := s.getPort()
    stationAddr := s.getStationAddr()
    
    // Initialize data points
    s.setupDataPoints()
    
    // Create server option
    option := cs104.NewOption()
    if err := option.AddRemoteServer(fmt.Sprintf("%s:%d", s.config.IP, port)); err != nil {
        return fmt.Errorf("failed to add remote server: %w", err)
    }
    
    // Create server
    s.server = cs104.NewServerSpecial(s.handler, option)
    
    // Set connection handlers
    s.server.SetOnConnectHandler(func(c asdu.Connect) {
        s.logger.Info("Client connected")
    })
    
    s.server.SetConnectionLostHandler(func(c asdu.Connect) {
        s.logger.Info("Client disconnected")
    })
    
    s.logger.Infof("Starting IEC 60870-5-104 Server on %s:%d (Station: %d)", 
        s.config.IP, port, stationAddr)
    
    // Create sync file
    if err := s.touchSyncFile(); err != nil {
        return fmt.Errorf("failed to create sync file: %w", err)
    }
    
    // Start server
    if err := s.server.Start(); err != nil {
        return fmt.Errorf("failed to start server: %w", err)
    }
    
    time.Sleep(100 * time.Millisecond)
    
    // Wait for interrupt signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    s.logger.Info("Server started successfully, waiting for signals...")
    <-sigChan
    
    s.logger.Info("Received shutdown signal")
    return s.shutdown()
}

func (s *IEC104Slave) setupDataPoints() {
    // Setup single points
    spValues := s.parseValues(s.config.SinglePoints)
    for i, val := range spValues {
        if boolVal, ok := val.(bool); ok {
            ioa := asdu.InfoObjAddr(i + 1)
            s.dataPoints[ioa] = boolVal
        }
    }
    
    // Setup measured values - start at IOA 1000 to avoid conflicts
    mvValues := s.parseValues(s.config.MeasuredValues)
    for i, val := range mvValues {
        if floatVal, ok := val.(float64); ok {
            ioa := asdu.InfoObjAddr(1000 + i)
            s.dataPoints[ioa] = float32(floatVal)
        }
    }
    
    s.logger.Infof("Configured %d single points and %d measured values", 
        len(spValues), len(mvValues))
}

func (s *IEC104Slave) shutdown() error {
    if s.server != nil {
        s.server.Close()
    }
    
    // Remove sync file
    if err := os.Remove(s.syncFile); err != nil && !os.IsNotExist(err) {
        s.logger.WithError(err).Warn("Failed to remove sync file")
    }
    
    s.logger.Info("Server shutdown completed")
    return nil
}

// ServerHandler methods - implementing cs104.ServerHandlerInterface
func (h *ServerHandler) InterrogationHandler(c asdu.Connect, packet *asdu.ASDU, qoi asdu.QualifierOfInterrogation) error {
    h.slave.logger.Info("Received interrogation command")
    
    // Send all configured data points
    go func() {
        time.Sleep(10 * time.Millisecond) // Small delay
        
        // Send single points
        spValues := h.slave.parseValues(h.slave.config.SinglePoints)
        for i, val := range spValues {
            if boolVal, ok := val.(bool); ok {
                ioa := asdu.InfoObjAddr(i + 1)
                
                // Create response ASDU with proper parameters
                identifier := asdu.Identifier{
                    Type:       asdu.M_SP_NA_1,
                    Variable:   asdu.VariableStruct{Number: 1},
                    Coa:        asdu.CauseOfTransmission{Cause: asdu.Spontaneous},
                    CommonAddr: h.stationAddr,
                }
                
                response := asdu.NewASDU(&asdu.Params{
                    CauseSize:       2,
                    CommonAddrSize:  2,
                    InfoObjAddrSize: 3,
                    InfoObjTimeZone: time.UTC,
                }, identifier)
                
                // Add information object
                if err := response.AppendInfoObjAddr(ioa); err != nil {
                    h.slave.logger.WithError(err).Error("Failed to append IOA")
                    continue
                }
                
                // Add single point value
                var spi byte = 0
                if boolVal {
                    spi = 1
                }
                response.AppendBytes(spi)
                
                if err := c.Send(response); err != nil {
                    h.slave.logger.WithError(err).Error("Failed to send single point")
                }
            }
        }
        
        // Send measured values
        mvValues := h.slave.parseValues(h.slave.config.MeasuredValues)
        for i, val := range mvValues {
            if floatVal, ok := val.(float64); ok {
                ioa := asdu.InfoObjAddr(1000 + i)
                
                // Create response ASDU
                identifier := asdu.Identifier{
                    Type:       asdu.M_ME_NA_1,
                    Variable:   asdu.VariableStruct{Number: 1},
                    Coa:        asdu.CauseOfTransmission{Cause: asdu.Spontaneous},
                    CommonAddr: h.stationAddr,
                }
                
                response := asdu.NewASDU(&asdu.Params{
                    CauseSize:       2,
                    CommonAddrSize:  2,
                    InfoObjAddrSize: 3,
                    InfoObjTimeZone: time.UTC,
                }, identifier)
                
                // Add information object
                if err := response.AppendInfoObjAddr(ioa); err != nil {
                    h.slave.logger.WithError(err).Error("Failed to append IOA")
                    continue
                }
                
                // Add normalized value
                normalized := asdu.Normalize(float32(floatVal))
                response.AppendNormalize(normalized)
                response.AppendBytes(0) // Quality descriptor (good)
                
                if err := c.Send(response); err != nil {
                    h.slave.logger.WithError(err).Error("Failed to send measured value")
                }
            }
        }
        
        // Send interrogation termination
        termIdentifier := asdu.Identifier{
            Type:       asdu.C_IC_NA_1,
            Variable:   packet.Variable,
            Coa:        asdu.CauseOfTransmission{Cause: asdu.ActivationTerm},
            CommonAddr: h.stationAddr,
        }
        
        termination := asdu.NewASDU(&asdu.Params{
            CauseSize:       packet.CauseSize,
            CommonAddrSize:  packet.CommonAddrSize,
            InfoObjAddrSize: packet.InfoObjAddrSize,
            InfoObjTimeZone: packet.InfoObjTimeZone,
        }, termIdentifier)
        
        if err := c.Send(termination); err != nil {
            h.slave.logger.WithError(err).Error("Failed to send termination")
        }
    }()
    
    // Send activation confirmation
    confirmIdentifier := asdu.Identifier{
        Type:       asdu.C_IC_NA_1,
        Variable:   packet.Variable,
        Coa:        asdu.CauseOfTransmission{Cause: asdu.ActivationCon},
        CommonAddr: h.stationAddr,
    }
    
    reply := asdu.NewASDU(&asdu.Params{
        CauseSize:       packet.CauseSize,
        CommonAddrSize:  packet.CommonAddrSize,
        InfoObjAddrSize: packet.InfoObjAddrSize,
        InfoObjTimeZone: packet.InfoObjTimeZone,
    }, confirmIdentifier)
    
    return c.Send(reply)
}

func (h *ServerHandler) CounterInterrogationHandler(c asdu.Connect, packet *asdu.ASDU, qcc asdu.QualifierCountCall) error {
    h.slave.logger.Info("Received counter interrogation command")
    return nil
}

func (h *ServerHandler) ReadHandler(c asdu.Connect, packet *asdu.ASDU, ioa asdu.InfoObjAddr) error {
    h.slave.logger.Infof("Received read command for IOA: %d", ioa)
    return nil
}

func (h *ServerHandler) ClockSyncHandler(c asdu.Connect, packet *asdu.ASDU, t time.Time) error {
    h.slave.logger.Infof("Received clock synchronization command: %v", t)
    
    // Send activation confirmation
    confirmIdentifier := asdu.Identifier{
        Type:       asdu.C_CS_NA_1,
        Variable:   packet.Variable,
        Coa:        asdu.CauseOfTransmission{Cause: asdu.ActivationCon},
        CommonAddr: h.stationAddr,
    }
    
    reply := asdu.NewASDU(&asdu.Params{
        CauseSize:       packet.CauseSize,
        CommonAddrSize:  packet.CommonAddrSize,
        InfoObjAddrSize: packet.InfoObjAddrSize,
        InfoObjTimeZone: packet.InfoObjTimeZone,
    }, confirmIdentifier)
    
    return c.Send(reply)
}

func (h *ServerHandler) ResetProcessHandler(c asdu.Connect, packet *asdu.ASDU, qrp asdu.QualifierOfResetProcessCmd) error {
    h.slave.logger.Info("Received reset process command")
    return nil
}

func (h *ServerHandler) DelayAcquisitionHandler(c asdu.Connect, packet *asdu.ASDU, msec uint16) error {
    h.slave.logger.Infof("Received delay acquisition command: %d msec", msec)
    return nil
}

func (h *ServerHandler) ASDUHandler(c asdu.Connect, packet *asdu.ASDU) error {
    switch packet.Type {
    case asdu.C_SC_NA_1: // Single command
        return h.handleSingleCommand(c, packet)
    default:
        h.slave.logger.Warnf("Unhandled ASDU type in ASDUHandler: %v", packet.Type)
        return nil
    }
}

func (h *ServerHandler) handleSingleCommand(c asdu.Connect, packet *asdu.ASDU) error {
    h.slave.logger.Info("Received single command")
    
    // Get command info
    cmdInfo := packet.GetSingleCmd()
    h.slave.logger.Infof("Single command for IOA %d: %v", cmdInfo.Ioa, cmdInfo.Value)
    
    // Update internal state
    h.slave.dataPoints[cmdInfo.Ioa] = cmdInfo.Value
    
    // Send activation confirmation
    confirmIdentifier := asdu.Identifier{
        Type:       asdu.C_SC_NA_1,
        Variable:   packet.Variable,
        Coa:        asdu.CauseOfTransmission{Cause: asdu.ActivationCon},
        CommonAddr: h.stationAddr,
    }
    
    reply := asdu.NewASDU(&asdu.Params{
        CauseSize:       packet.CauseSize,
        CommonAddrSize:  packet.CommonAddrSize,
        InfoObjAddrSize: packet.InfoObjAddrSize,
        InfoObjTimeZone: packet.InfoObjTimeZone,
    }, confirmIdentifier)
    
    return c.Send(reply)
}

func main() {
    fmt.Println("Starting IEC 60870-5-104 Slave...")

    configFile := "slave.yaml"
    if envConfig := os.Getenv("SLAVE_CONFIG"); envConfig != "" {
        configFile = envConfig
    }
    
    syncFile := "app_running.lock"
    if envSync := os.Getenv("SYNC_FILE"); envSync != "" {
        syncFile = envSync
    }

    slave, err := NewIEC104Slave(configFile, syncFile)
    if err != nil {
        log.Fatalf("Error creating slave: %v", err)
    }

    if err := slave.start(); err != nil {
        log.Fatalf("Error starting slave: %v", err)
    }
}
