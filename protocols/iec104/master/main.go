package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wendy512/go-iecp5/asdu"
	"github.com/wendy512/go-iecp5/cs104"
)

type Command struct {
    Timestamp    float64
    IP           string
    Port         int
    StationAddr  int
    InfoAddr     int
    CommandType  string // C_IC_NA_1, C_SC_NA_1, C_CS_NA_1, etc.
    Value        interface{}
    Recurrent    bool
    Interval     float64
}

// ClientHandler implements ClientHandlerInterface
type ClientHandler struct {
    master *IEC104Master
}

type IEC104Master struct {
    csvFile   string
    commands  []Command
    client    *cs104.Client  // Cambio aquí: *cs104.Client en lugar de cs104.ClientSpecial
    handler   *ClientHandler
    responses []interface{}
    logger    *logrus.Logger
}

func NewIEC104Master(csvFile string) *IEC104Master {
    logger := logrus.New()
    logger.SetLevel(logrus.InfoLevel)
    
    master := &IEC104Master{
        csvFile:   csvFile,
        commands:  make([]Command, 0),
        responses: make([]interface{}, 0),
        logger:    logger,
    }

    // Create handler
    master.handler = &ClientHandler{
        master: master,
    }
    
    return master
}

func (m *IEC104Master) setup() error {
    file, err := os.Open(m.csvFile)
    if err != nil {
        return fmt.Errorf("failed to open CSV file: %w", err)
    }
    defer file.Close()

    reader := csv.NewReader(file)
    
    // Read header
    header, err := reader.Read()
    if err != nil {
        return fmt.Errorf("failed to read CSV header: %w", err)
    }

    // Read all records
    for {
        record, err := reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            return fmt.Errorf("failed to read CSV record: %w", err)
        }

        // Create map from header and record
        row := make(map[string]string)
        for i, value := range record {
            if i < len(header) {
                row[header[i]] = value
            }
        }

        command, err := m.parseCommand(row)
        if err != nil {
            m.logger.WithError(err).Warn("Failed to parse command, skipping")
            continue
        }

        m.commands = append(m.commands, command)
    }

    // Sort commands by timestamp
    sort.Slice(m.commands, func(i, j int) bool {
        return m.commands[i].Timestamp < m.commands[j].Timestamp
    })

    m.logger.Infof("Loaded %d commands from %s", len(m.commands), m.csvFile)
    return nil
}

func (m *IEC104Master) parseCommand(row map[string]string) (Command, error) {
    cmd := Command{}
    var err error

    // Parse timestamp
    if cmd.Timestamp, err = strconv.ParseFloat(row["timestamp"], 64); err != nil {
        return cmd, fmt.Errorf("invalid timestamp: %w", err)
    }

    // Parse basic fields
    cmd.IP = row["ip"]
    
    if cmd.Port, err = strconv.Atoi(row["port"]); err != nil {
        return cmd, fmt.Errorf("invalid port: %w", err)
    }
    
    if cmd.StationAddr, err = strconv.Atoi(row["station_addr"]); err != nil {
        return cmd, fmt.Errorf("invalid station_addr: %w", err)
    }
    
    if cmd.InfoAddr, err = strconv.Atoi(row["info_addr"]); err != nil {
        return cmd, fmt.Errorf("invalid info_addr: %w", err)
    }

    cmd.CommandType = row["command_type"]
    
    // Parse recurrent
    cmd.Recurrent = strings.ToLower(row["recurrent"]) == "true"

    // Parse interval if recurrent
    if cmd.Recurrent && row["interval"] != "" {
        if cmd.Interval, err = strconv.ParseFloat(row["interval"], 64); err != nil {
            return cmd, fmt.Errorf("invalid interval: %w", err)
        }
    }

    // Parse value based on command type
    switch cmd.CommandType {
    case "C_SC_NA_1": // Single command
        if val, err := strconv.ParseBool(row["value"]); err == nil {
            cmd.Value = val
        }
    case "C_RC_NA_1": // Regulating step command
        if val, err := strconv.ParseInt(row["value"], 10, 8); err == nil {
            cmd.Value = int8(val)
        }
    default:
        cmd.Value = row["value"]
    }

    return cmd, nil
}

func (m *IEC104Master) connect(ip string, port int) error {
    // Create client option
    option := cs104.NewOption()
    if err := option.AddRemoteServer(fmt.Sprintf("%s:%d", ip, port)); err != nil {
        return fmt.Errorf("failed to add remote server: %w", err)
    }

    // Create client
    m.client = cs104.NewClient(m.handler, option)

    // Set connection handlers
    connectionEstablished := make(chan bool, 1)
    serverActivated := make(chan bool, 1)

    m.client.SetOnConnectHandler(func(c *cs104.Client) {
        m.logger.Info("Connected to IEC 60870-5-104 server")
        select {
        case connectionEstablished <- true:
        default:
        }
    })

    // ¡NUEVO! Handler para cuando el servidor se active
    m.client.SetServerActiveHandler(func(c *cs104.Client) {
        m.logger.Info("Server is now active")
        select {
        case serverActivated <- true:
        default:
        }
    })

    m.client.SetConnectionLostHandler(func(c *cs104.Client) {
        m.logger.Warn("Connection lost to IEC 60870-5-104 server")
    })

    // Start client
    if err := m.client.Start(); err != nil {
        return fmt.Errorf("failed to connect: %w", err)
    }

    // Wait for connection to be established
    select {
    case <-connectionEstablished:
        m.logger.Info("Connection established successfully")
    case <-time.After(10 * time.Second):
        return fmt.Errorf("connection timeout")
    }

    // ¡NUEVO! Enviar STARTDT para activar el servidor
    m.logger.Info("Sending STARTDT to activate server")
    m.client.SendStartDt()

    // ¡NUEVO! Esperar a que el servidor se active
    select {
    case <-serverActivated:
        m.logger.Info("Server activation confirmed")
    case <-time.After(5 * time.Second):
        m.logger.Warn("Server activation timeout, continuing anyway")
    }

    // Give a bit more time for the connection to stabilize
    time.Sleep(200 * time.Millisecond)
    return nil
}


// ClientHandler methods - implementing cs104.ClientHandlerInterface
func (h *ClientHandler) InterrogationHandler(c asdu.Connect, packet *asdu.ASDU) error {
    // Extraer QOI del packet si es necesario
    _, qoi := packet.GetInterrogationCmd()
    h.master.logger.Infof("Received interrogation response with QOI: %v", qoi)
    h.master.responses = append(h.master.responses, packet)
    return nil
}

func (h *ClientHandler) CounterInterrogationHandler(c asdu.Connect, packet *asdu.ASDU) error {
    // Extraer QCC del packet si es necesario
    _, qcc := packet.GetCounterInterrogationCmd()
    h.master.logger.Infof("Received counter interrogation response with QCC: %v", qcc)
    h.master.responses = append(h.master.responses, packet)
    return nil
}

func (h *ClientHandler) ReadHandler(c asdu.Connect, packet *asdu.ASDU) error {
    // Extraer IOA del packet si es necesario
    ioa := packet.GetReadCmd()
    h.master.logger.Infof("Received read response for IOA: %d", ioa)
    h.master.responses = append(h.master.responses, packet)
    return nil
}

func (h *ClientHandler) TestCommandHandler(c asdu.Connect, packet *asdu.ASDU) error {
    // Extraer información del test command del packet si es necesario
    ioa, testValue := packet.GetTestCommand()
    h.master.logger.Infof("Received test command response - IOA: %d, Value: %v", ioa, testValue)
    h.master.responses = append(h.master.responses, packet)
    return nil
}

func (h *ClientHandler) ClockSyncHandler(c asdu.Connect, packet *asdu.ASDU) error {
    // Extraer el tiempo del packet si es necesario
    _, clockSyncTime := packet.GetClockSynchronizationCmd()
    h.master.logger.Infof("Received clock sync response: %v", clockSyncTime)
    h.master.responses = append(h.master.responses, packet)
    return nil
}

func (h *ClientHandler) ResetProcessHandler(c asdu.Connect, packet *asdu.ASDU) error {
    // Extraer QRP del packet si es necesario
    _, qrp := packet.GetResetProcessCmd()
    h.master.logger.Infof("Received reset process response with QRP: %v", qrp)
    h.master.responses = append(h.master.responses, packet)
    return nil
}

func (h *ClientHandler) DelayAcquisitionHandler(c asdu.Connect, packet *asdu.ASDU) error {
    // Extraer msec del packet si es necesario
    _, msec := packet.GetDelayAcquireCommand()
    h.master.logger.Infof("Received delay acquisition response: %d msec", msec)
    h.master.responses = append(h.master.responses, packet)
    return nil
}

func (h *ClientHandler) ASDUHandler(c asdu.Connect, packet *asdu.ASDU) error {
    switch packet.Type {
    case asdu.M_SP_NA_1: // Single point information
        return h.handleSinglePoint(c, packet)
    case asdu.M_ME_NA_1: // Measured value, normalized
        return h.handleMeasuredValue(c, packet)
    case asdu.M_SP_TB_1: // Single point information with time tag
        return h.handleSinglePoint(c, packet)
    case asdu.M_ME_TD_1: // Measured value, normalized with time tag
        return h.handleMeasuredValue(c, packet)
    default:
        h.master.logger.Warnf("Unhandled ASDU type: %v", packet.Type)
        return nil
    }
}

func (h *ClientHandler) handleSinglePoint(c asdu.Connect, packet *asdu.ASDU) error {
    singlePoints := packet.GetSinglePoint()
    for _, sp := range singlePoints {
        h.master.logger.Infof("Received single point - IOA: %d, Value: %v, Quality: %v", 
            sp.Ioa, sp.Value, sp.Qds)
    }
    h.master.responses = append(h.master.responses, singlePoints)
    return nil
}

func (h *ClientHandler) handleMeasuredValue(c asdu.Connect, packet *asdu.ASDU) error {
    measuredValues := packet.GetMeasuredValueNormal()
    for _, mv := range measuredValues {
        h.master.logger.Infof("Received measured value - IOA: %d, Value: %v, Quality: %v", 
            mv.Ioa, mv.Value, mv.Qds)
    }
    h.master.responses = append(h.master.responses, measuredValues)
    return nil
}

func (m *IEC104Master) sendCommand(cmd Command) error {
    if m.client == nil {
        return fmt.Errorf("not connected to server")
    }

    m.logger.Infof("Sending command to %s:%d - Type:%s - StationAddr:%d - InfoAddr:%d - Value:%v",
        cmd.IP, cmd.Port, cmd.CommandType, cmd.StationAddr, cmd.InfoAddr, cmd.Value)

    stationAddr := asdu.CommonAddr(cmd.StationAddr)
    infoAddr := asdu.InfoObjAddr(cmd.InfoAddr)

    switch cmd.CommandType {
    case "C_IC_NA_1": // Interrogation command
        // Create interrogation command ASDU
        identifier := asdu.Identifier{
            Type:       asdu.C_IC_NA_1,
            Variable:   asdu.VariableStruct{Number: 1},
            Coa:        asdu.CauseOfTransmission{Cause: asdu.Activation},
            CommonAddr: stationAddr,
        }
        
        cmdASDU := asdu.NewASDU(&asdu.Params{
            CauseSize:       2,
            CommonAddrSize:  2,
            InfoObjAddrSize: 3,
            InfoObjTimeZone: time.UTC,
        }, identifier)
        
        // Add information object address (0 for general interrogation)
        if err := cmdASDU.AppendInfoObjAddr(0); err != nil {
            return fmt.Errorf("failed to append IOA: %w", err)
        }
        
        // Add QOI (Qualifier of Interrogation) - 20 for station interrogation
        cmdASDU.AppendBytes(20)
        
        if err := m.client.Send(cmdASDU); err != nil {
            return fmt.Errorf("interrogation command failed: %w", err)
        }

    case "C_CS_NA_1": // Clock synchronization command
        identifier := asdu.Identifier{
            Type:       asdu.C_CS_NA_1,
            Variable:   asdu.VariableStruct{Number: 1},
            Coa:        asdu.CauseOfTransmission{Cause: asdu.Activation},
            CommonAddr: stationAddr,
        }
        
        cmdASDU := asdu.NewASDU(&asdu.Params{
            CauseSize:       2,
            CommonAddrSize:  2,
            InfoObjAddrSize: 3,
            InfoObjTimeZone: time.UTC,
        }, identifier)
        
        // Add information object address (0 for clock sync)
        if err := cmdASDU.AppendInfoObjAddr(0); err != nil {
            return fmt.Errorf("failed to append IOA: %w", err)
        }
        
        // Add current time
        cmdASDU.AppendCP56Time2a(time.Now(), time.UTC)
        
        if err := m.client.Send(cmdASDU); err != nil {
            return fmt.Errorf("clock sync command failed: %w", err)
        }

    case "C_SC_NA_1": // Single command
        if val, ok := cmd.Value.(bool); ok {
            identifier := asdu.Identifier{
                Type:       asdu.C_SC_NA_1,
                Variable:   asdu.VariableStruct{Number: 1},
                Coa:        asdu.CauseOfTransmission{Cause: asdu.Activation},
                CommonAddr: stationAddr,
            }
            
            cmdASDU := asdu.NewASDU(&asdu.Params{
                CauseSize:       2,
                CommonAddrSize:  2,
                InfoObjAddrSize: 3,
                InfoObjTimeZone: time.UTC,
            }, identifier)
            
            // Add information object address
            if err := cmdASDU.AppendInfoObjAddr(infoAddr); err != nil {
                return fmt.Errorf("failed to append IOA: %w", err)
            }
            
            // Add single command information (SCO)
            var sco byte = 0
            if val {
                sco = 1
            }
            cmdASDU.AppendBytes(sco)
            
            if err := m.client.Send(cmdASDU); err != nil {
                return fmt.Errorf("single command failed: %w", err)
            }
        } else {
            return fmt.Errorf("invalid value type for single command: %T", cmd.Value)
        }

    default:
        return fmt.Errorf("unsupported command type: %s", cmd.CommandType)
    }

    return nil
}

func (m *IEC104Master) loop() error {
    currentTime := 0.0
    // Connect to first server
    if len(m.commands) > 0 {
        firstCmd := m.commands[0]
        err := m.connect(firstCmd.IP, firstCmd.Port)
        if err != nil {
            return err
        }
        defer func() {
            // ¡NUEVO! Enviar STOPDT antes de cerrar
            m.logger.Info("Sending STOPDT to deactivate server")
            m.client.SendStopDt()
            m.client.Close()
        }()

        // Additional wait to ensure connection is stable
        time.Sleep(500 * time.Millisecond)
    }

    for len(m.commands) > 0 {
        // Get next command
        cmd := m.commands[0]
        m.commands = m.commands[1:]

        // Wait for the appropriate time
        delay := cmd.Timestamp - currentTime
        if delay > 0 {
            time.Sleep(time.Duration(delay * float64(time.Second)))
        }
        currentTime = cmd.Timestamp

        // Check if client is still connected before sending
        if m.client == nil {
            m.logger.Error("Client is nil, cannot send command")
            continue
        }

        // Send the command
        err := m.sendCommand(cmd)
        if err != nil {
            m.logger.WithError(err).Error("Failed to send command")
        }

        // Small delay between commands to avoid overwhelming the connection
        time.Sleep(100 * time.Millisecond)

        // If recurrent, reschedule
        if cmd.Recurrent && cmd.Interval > 0 {
            cmd.Timestamp += cmd.Interval
            
            // Insert back into commands maintaining order
            inserted := false
            for i, existingCmd := range m.commands {
                if cmd.Timestamp < existingCmd.Timestamp {
                    m.commands = append(m.commands[:i], append([]Command{cmd}, m.commands[i:]...)...)
                    inserted = true
                    break
                }
            }
            if !inserted {
                m.commands = append(m.commands, cmd)
            }
        }
    }

    // Keep connection alive for a bit to receive responses
    time.Sleep(2 * time.Second)
    
    m.logger.Infof("All commands processed. Received %d responses", len(m.responses))
    return nil
}

func main() {
    fmt.Println("Starting IEC 60870-5-104 Master...")

    // Allow config file path to be overridden via environment variable
    configFile := "master.csv"
    if envConfig := os.Getenv("MASTER_CONFIG"); envConfig != "" {
        configFile = envConfig
    }

    master := NewIEC104Master(configFile)
    
    if err := master.setup(); err != nil {
        log.Fatalf("Error setting up master: %v", err)
    }

    if err := master.loop(); err != nil {
        log.Fatalf("Error in main loop: %v", err)
    }
}
