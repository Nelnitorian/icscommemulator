package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"syscall"
	"time"

	"github.com/goburrow/modbus"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Message struct {
	Timestamp    int
	IP           string
	Port         int
	FunctionCode int
	StartAddress int
	SlaveID      int
	Recurrent    bool
	Interval     int
	Count        int
	Values       []int
}

type MasterConfig struct {
	Protocol string          `yaml:"protocol"`
	Messages []MessageConfig `yaml:"messages"`
}

type MessageConfig struct {
	Timestamp    int           `yaml:"timestamp"`
	Recurrent    bool          `yaml:"recurrent"`
	Interval     int           `yaml:"interval"`
	IP           string        `yaml:"ip"`
	Port         int           `yaml:"port"`
	SlaveID      int           `yaml:"slave_id"`
	FunctionCode int           `yaml:"function_code"`
	StartAddress int           `yaml:"start_address"`
	Count        int           `yaml:"count"`
	Values       []interface{} `yaml:"values"`
}

type ModbusMaster struct {
	configFile   string
	messages     []Message
	handlers     map[string]*modbus.TCPClientHandler
	responses    []interface{}
	logger       *logrus.Logger
	hasRecurrent bool
}

func NewModbusMaster(configFile string) *ModbusMaster {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	return &ModbusMaster{
		configFile:   configFile,
		messages:     make([]Message, 0),
		handlers:     make(map[string]*modbus.TCPClientHandler),
		responses:    make([]interface{}, 0),
		logger:       logger,
		hasRecurrent: false,
	}
}

func (m *ModbusMaster) setup() error {
	data, err := os.ReadFile(m.configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg MasterConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	for _, raw := range cfg.Messages {
		msg, err := m.parseMessage(raw)
		if err != nil {
			m.logger.WithError(err).WithField("message", raw).Warn("Failed to parse message, skipping")
			continue
		}
		m.messages = append(m.messages, msg)

		if msg.Recurrent {
			m.hasRecurrent = true
		}

		clientKey := fmt.Sprintf("%s:%d", msg.IP, msg.Port)
		if _, exists := m.handlers[clientKey]; !exists {
			handler := modbus.NewTCPClientHandler(fmt.Sprintf("%s:%d", msg.IP, msg.Port))
			handler.Timeout = 10 * time.Second
			handler.SlaveId = byte(msg.SlaveID)
			m.handlers[clientKey] = handler
		}
	}

	sort.Slice(m.messages, func(i, j int) bool {
		return m.messages[i].Timestamp < m.messages[j].Timestamp
	})

	m.logger.Infof("Loaded %d messages from %s", len(m.messages), m.configFile)
	if m.hasRecurrent {
		m.logger.Info("Recurrent messages detected - master will run indefinitely (Ctrl+C to stop)")
	}
	return nil
}

func (m *ModbusMaster) parseMessage(raw MessageConfig) (Message, error) {
	msg := Message{
		Timestamp:    raw.Timestamp,
		IP:           raw.IP,
		Port:         raw.Port,
		FunctionCode: raw.FunctionCode,
		StartAddress: raw.StartAddress,
		SlaveID:      raw.SlaveID,
		Recurrent:    raw.Recurrent,
		Interval:     raw.Interval,
		Count:        raw.Count,
		Values:       []int{},
	}

	for i, value := range raw.Values {
		switch v := value.(type) {
		case int:
			msg.Values = append(msg.Values, v)
		case int64:
			msg.Values = append(msg.Values, int(v))
		case float64:
			msg.Values = append(msg.Values, int(v))
		case string:
			if v == "" {
				continue
			}
			parsed, err := strconv.Atoi(v)
			if err != nil {
				return msg, fmt.Errorf("invalid value '%s' at index %d: %w", v, i, err)
			}
			msg.Values = append(msg.Values, parsed)
		default:
			return msg, fmt.Errorf("unsupported value type at index %d", i)
		}
	}

	return msg, nil
}

func (m *ModbusMaster) sendMessage(msg Message) interface{} {
	clientKey := fmt.Sprintf("%s:%d", msg.IP, msg.Port)
	handler := m.handlers[clientKey]

	handler.SlaveId = byte(msg.SlaveID)

	var err error
	for attempt := 0; attempt < 5; attempt++ {
		err = handler.Connect()
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		m.logger.WithError(err).Error("Failed to connect")
		return err
	}
	defer handler.Close()

	client := modbus.NewClient(handler)

	m.logger.Infof("Sending msg to %s:%d - FC:%d - Addr:%d - Slave:%d - Values:%v - Count:%d",
		msg.IP, msg.Port, msg.FunctionCode, msg.StartAddress, msg.SlaveID, msg.Values, msg.Count)

	var result []byte

	switch msg.FunctionCode {
	case 1:
		result, err = client.ReadCoils(uint16(msg.StartAddress), uint16(msg.Count))

	case 2:
		result, err = client.ReadDiscreteInputs(uint16(msg.StartAddress), uint16(msg.Count))

	case 3:
		result, err = client.ReadHoldingRegisters(uint16(msg.StartAddress), uint16(msg.Count))

	case 4:
		result, err = client.ReadInputRegisters(uint16(msg.StartAddress), uint16(msg.Count))

	case 5:
		if len(msg.Values) > 0 {
			value := uint16(0)
			if msg.Values[0] != 0 {
				value = 0xFF00
			}
			result, err = client.WriteSingleCoil(uint16(msg.StartAddress), value)
		} else {
			err = fmt.Errorf("no values provided for write single coil")
		}

	case 6:
		if len(msg.Values) > 0 {
			result, err = client.WriteSingleRegister(uint16(msg.StartAddress), uint16(msg.Values[0]))
		} else {
			err = fmt.Errorf("no values provided for write single register")
		}

	case 15:
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

	case 16:
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
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	currentTime := 0

	for {
		select {
		case <-sigChan:
			m.logger.Info("Received shutdown signal, stopping...")
			return
		default:
		}

		if len(m.messages) == 0 {
			if !m.hasRecurrent {
				m.logger.Info("All messages processed")
				return
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}

		msg := m.messages[0]
		m.messages = m.messages[1:]

		delay := msg.Timestamp - currentTime
		if delay > 0 {
			time.Sleep(time.Duration(delay) * time.Second)
		}
		currentTime = msg.Timestamp

		response := m.sendMessage(msg)
		m.responses = append(m.responses, response)

		if msg.Recurrent && msg.Interval > 0 {
			msg.Timestamp += msg.Interval

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

	configFile := "/app/config/master.yaml"
	if envConfig := os.Getenv("MASTER_CONFIG"); envConfig != "" {
		configFile = envConfig
	}

	master := NewModbusMaster(configFile)
	if err := master.setup(); err != nil {
		log.Fatalf("Error setting up master: %v", err)
	}

	master.loop()
}
