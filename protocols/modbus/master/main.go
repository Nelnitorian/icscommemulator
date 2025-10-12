package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/goburrow/modbus"
	"github.com/sirupsen/logrus"
)

type Message struct {
	Timestamp    float64
	IP           string
	Port         int
	FunctionCode int
	StartAddress int
	SlaveID      int
	Recurrent    bool
	Interval     float64 // In milliseconds
	Count        int
	Values       []int
	OriginalRow  map[string]string
}

type ModbusMaster struct {
	csvFile   string
	messages  []Message
	handlers  map[string]*modbus.TCPClientHandler
	responses []interface{}
	logger    *logrus.Logger
	hasRecurrent bool
}

func NewModbusMaster(csvFile string) *ModbusMaster {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	return &ModbusMaster{
		csvFile:   csvFile,
		messages:  make([]Message, 0),
		handlers:  make(map[string]*modbus.TCPClientHandler),
		responses: make([]interface{}, 0),
		logger:    logger,
		hasRecurrent: false,
	}
}

func (m *ModbusMaster) setup() error {
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

		message, err := m.parseMessage(row)
		if err != nil {
			m.logger.WithError(err).WithField("row", row).Warn("Failed to parse message, skipping")
			continue
		}

		m.messages = append(m.messages, message)
		
		// Track if we have any recurrent messages
		if message.Recurrent {
			m.hasRecurrent = true
		}

		// Create handler if not exists
		clientKey := fmt.Sprintf("%s:%d", message.IP, message.Port)
		if _, exists := m.handlers[clientKey]; !exists {
			handler := modbus.NewTCPClientHandler(fmt.Sprintf("%s:%d", message.IP, message.Port))
			handler.Timeout = 10 * time.Second
			handler.SlaveId = byte(message.SlaveID)
			m.handlers[clientKey] = handler
		}
	}

	// Sort messages by timestamp
	sort.Slice(m.messages, func(i, j int) bool {
		return m.messages[i].Timestamp < m.messages[j].Timestamp
	})

	m.logger.Infof("Loaded %d messages from %s", len(m.messages), m.csvFile)
	if m.hasRecurrent {
		m.logger.Info("Recurrent messages detected - master will run indefinitely (Ctrl+C to stop)")
	}
	return nil
}

// cleanValueString removes brackets, braces and extra whitespace from value strings
func cleanValueString(s string) string {
	s = strings.TrimSpace(s)
	// Remove surrounding brackets or braces
	s = strings.Trim(s, "[]{}()")
	return strings.TrimSpace(s)
}

func (m *ModbusMaster) parseMessage(row map[string]string) (Message, error) {
	msg := Message{OriginalRow: row}
	var err error

	// Parse timestamp
	if msg.Timestamp, err = strconv.ParseFloat(row["timestamp"], 64); err != nil {
		return msg, fmt.Errorf("invalid timestamp: %w", err)
	}

	// Parse basic fields
	msg.IP = row["ip"]

	if msg.Port, err = strconv.Atoi(row["port"]); err != nil {
		return msg, fmt.Errorf("invalid port: %w", err)
	}

	if msg.FunctionCode, err = strconv.Atoi(row["function_code"]); err != nil {
		return msg, fmt.Errorf("invalid function_code: %w", err)
	}

	if msg.SlaveID, err = strconv.Atoi(row["slave_id"]); err != nil {
		return msg, fmt.Errorf("invalid slave_id: %w", err)
	}

	// Parse recurrent
	msg.Recurrent = strings.ToLower(row["recurrent"]) == "true"

	// Parse interval if recurrent (in milliseconds)
	if msg.Recurrent && row["interval"] != "" {
		if msg.Interval, err = strconv.ParseFloat(row["interval"], 64); err != nil {
			return msg, fmt.Errorf("invalid interval: %w", err)
		}
	}

	// Parse start_address for read/write functions
	if msg.FunctionCode >= 1 && msg.FunctionCode <= 6 || msg.FunctionCode == 15 || msg.FunctionCode == 16 {
		if row["start_address"] != "" {
			if msg.StartAddress, err = strconv.Atoi(row["start_address"]); err != nil {
				return msg, fmt.Errorf("invalid start_address: %w", err)
			}
		}
	}

	// Parse count for read functions
	if msg.FunctionCode >= 1 && msg.FunctionCode <= 4 && row["count"] != "" {
		if msg.Count, err = strconv.Atoi(row["count"]); err != nil {
			return msg, fmt.Errorf("invalid count: %w", err)
		}
	}

	// Parse values for write functions
	if (msg.FunctionCode == 5 || msg.FunctionCode == 6 || msg.FunctionCode == 15 || msg.FunctionCode == 16) && row["values"] != "" {
		// Clean the values string (remove brackets, trim spaces)
		cleanedValues := cleanValueString(row["values"])
		
		if cleanedValues == "" {
			return msg, fmt.Errorf("empty values after cleaning")
		}

		valueStrs := strings.Split(cleanedValues, ",")
		msg.Values = make([]int, 0, len(valueStrs))
		
		for i, valueStr := range valueStrs {
			valueStr = strings.TrimSpace(valueStr)
			if valueStr == "" {
				continue // Skip empty values
			}
			
			val, err := strconv.Atoi(valueStr)
			if err != nil {
				return msg, fmt.Errorf("invalid value '%s' at index %d: %w", valueStr, i, err)
			}
			msg.Values = append(msg.Values, val)
		}

		if len(msg.Values) == 0 {
			return msg, fmt.Errorf("no valid values parsed from '%s'", row["values"])
		}
	}

	return msg, nil
}

func (m *ModbusMaster) sendMessage(msg Message) interface{} {
	clientKey := fmt.Sprintf("%s:%d", msg.IP, msg.Port)
	handler := m.handlers[clientKey]

	// Set correct slave ID for this message
	handler.SlaveId = byte(msg.SlaveID)

	// Connect
	err := handler.Connect()
	if err != nil {
		m.logger.WithError(err).Error("Failed to connect")
		return err
	}
	defer handler.Close()

	// Create client
	client := modbus.NewClient(handler)

	m.logger.Infof("Sending msg to %s:%d - FC:%d - Addr:%d - Slave:%d - Values:%v - Count:%d",
		msg.IP, msg.Port, msg.FunctionCode, msg.StartAddress, msg.SlaveID, msg.Values, msg.Count)

	var result []byte

	switch msg.FunctionCode {
	case 1: // Read Coils
		result, err = client.ReadCoils(uint16(msg.StartAddress), uint16(msg.Count))

	case 2: // Read Discrete Inputs
		result, err = client.ReadDiscreteInputs(uint16(msg.StartAddress), uint16(msg.Count))

	case 3: // Read Holding Registers
		result, err = client.ReadHoldingRegisters(uint16(msg.StartAddress), uint16(msg.Count))

	case 4: // Read Input Registers
		result, err = client.ReadInputRegisters(uint16(msg.StartAddress), uint16(msg.Count))

	case 5: // Write Single Coil
		if len(msg.Values) > 0 {
			value := uint16(0)
			if msg.Values[0] != 0 {
				value = 0xFF00
			}
			result, err = client.WriteSingleCoil(uint16(msg.StartAddress), value)
		} else {
			err = fmt.Errorf("no values provided for write single coil")
		}

	case 6: // Write Single Register
		if len(msg.Values) > 0 {
			result, err = client.WriteSingleRegister(uint16(msg.StartAddress), uint16(msg.Values[0]))
		} else {
			err = fmt.Errorf("no values provided for write single register")
		}

	case 15: // Write Multiple Coils
		// Convert values to byte array
		numBytes := (len(msg.Values) + 7) / 8
		values := make([]byte, numBytes)
		for i, v := range msg.Values {
			if v != 0 {
				byteIndex := i / 8
				bitIndex := i % 8
				values[byteIndex] |= 1 << bitIndex
			}
		}
		result, err = client.WriteMultipleCoils(uint16(msg.StartAddress), uint16(len(msg.Values)), values)

	case 16: // Write Multiple Registers
		values := make([]byte, len(msg.Values)*2)
		for i, v := range msg.Values {
			values[i*2] = byte(v >> 8)
			values[i*2+1] = byte(v & 0xFF)
		}
		result, err = client.WriteMultipleRegisters(uint16(msg.StartAddress), uint16(len(msg.Values)), values)

	default:
		err = fmt.Errorf("unsupported function code: %d", msg.FunctionCode)
	}

	if err != nil {
		m.logger.WithError(err).Errorf("Modbus operation failed for function code %d", msg.FunctionCode)
		return err
	}

	m.logger.Infof("Operation successful - FC:%d - Result length: %d bytes", msg.FunctionCode, len(result))
	return result
}

func (m *ModbusMaster) loop() {
	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start time tracking
	currentTime := 0.0

	for {
		// Check for signals
		select {
		case <-sigChan:
			m.logger.Info("Received shutdown signal, stopping...")
			return
		default:
			// Continue processing
		}

		// Check if we have messages to process
		if len(m.messages) == 0 {
			if !m.hasRecurrent {
				// No more messages and no recurrent ones - we're done
				m.logger.Info("All messages processed")
				return
			}
			// If we have recurrent messages but queue is empty, wait a bit
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Get next message
		msg := m.messages[0]
		m.messages = m.messages[1:]

		// Wait for the appropriate time
		delay := msg.Timestamp - currentTime
		if delay > 0 {
			time.Sleep(time.Duration(delay * float64(time.Second)))
		}
		currentTime = msg.Timestamp

		// Send the message
		response := m.sendMessage(msg)
		m.responses = append(m.responses, response)

		// If recurrent, reschedule
		if msg.Recurrent && msg.Interval > 0 {
			// Convert interval from milliseconds to seconds and add to timestamp
			msg.Timestamp += msg.Interval / 1000.0

			// Insert back into messages maintaining order
			inserted := false
			for i, existingMsg := range m.messages {
				if msg.Timestamp < existingMsg.Timestamp {
					m.messages = append(m.messages[:i], append([]Message{msg}, m.messages[i:]...)...)
					inserted = true
					break
				}
			}
			if !inserted {
				m.messages = append(m.messages, msg)
			}
		}
	}
}


func main() {
	fmt.Println("Starting ModbusMaster...")

	// Allow config file path to be overridden via environment variable
	configFile := "/app/config/master.csv"
	if envConfig := os.Getenv("MASTER_CONFIG"); envConfig != "" {
		configFile = envConfig
	}

	master := NewModbusMaster(configFile)
	if err := master.setup(); err != nil {
		log.Fatalf("Error setting up master: %v", err)
	}

	master.loop()
}
