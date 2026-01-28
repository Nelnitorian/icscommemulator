package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/juanjorosendo/go-iecp5/asdu"
	"github.com/juanjorosendo/go-iecp5/cs104"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type MasterConfig struct {
	Protocol string          `yaml:"protocol"`
	Messages []ScheduleEntry `yaml:"messages"`
}

type ScheduleEntry struct {
	Timestamp     int    `yaml:"timestamp"`
	Recurrent     bool   `yaml:"recurrent"`
	Interval      int    `yaml:"interval"`
	IP            string `yaml:"ip"`
	Port          int    `yaml:"port"`
	TypeID        int    `yaml:"type_id"`
	CommonAddress int    `yaml:"common_address"`
	IOA           int    `yaml:"ioa"`
	COT           int    `yaml:"cot"`
	Value         string `yaml:"value"`
}

type IEC104Master struct {
	config        MasterConfig
	logger        *logrus.Logger
	clients       map[string]*cs104.Client
	queues        map[string][]ScheduleEntry
	queueMu       sync.Mutex
	hasRecurrent  bool
	activeSignals map[string]chan struct{}
}

func NewIEC104Master(configPath string) (*IEC104Master, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg MasterConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	return &IEC104Master{
		config:        cfg,
		logger:        logger,
		clients:       make(map[string]*cs104.Client),
		queues:        make(map[string][]ScheduleEntry),
		activeSignals: make(map[string]chan struct{}),
	}, nil
}

func (m *IEC104Master) setup() error {
	for _, msg := range m.config.Messages {
		key := fmt.Sprintf("%s:%d", msg.IP, msg.Port)
		m.queues[key] = append(m.queues[key], msg)
		if msg.Recurrent {
			m.hasRecurrent = true
		}
	}

	for key, queue := range m.queues {
		sort.Slice(queue, func(i, j int) bool {
			return queue[i].Timestamp < queue[j].Timestamp
		})
		m.queues[key] = queue
	}

	for key := range m.queues {
		client, err := m.createClient(key)
		if err != nil {
			return err
		}
		m.clients[key] = client
	}

	m.logger.Infof("Loaded %d IEC104 messages from schedule", len(m.config.Messages))
	return nil
}

func (m *IEC104Master) createClient(address string) (*cs104.Client, error) {
	option := cs104.NewOption()
	if err := option.AddRemoteServer(address); err != nil {
		return nil, err
	}

	handler := &iec104ClientHandler{logger: m.logger}
	client := cs104.NewClient(handler, option)
	client.LogMode(false)
	active := make(chan struct{})
	client.SetOnConnectHandler(func(c *cs104.Client) {
		c.SendStartDt()
	})
	client.SetServerActiveHandler(func(c *cs104.Client) {
		select {
		case <-active:
		default:
			close(active)
		}
	})

	if err := client.Start(); err != nil {
		return nil, err
	}

	m.activeSignals[address] = active
	return client, nil
}

func (m *IEC104Master) loop() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	wg := sync.WaitGroup{}

	for key, queue := range m.queues {
		client := m.clients[key]
		wg.Add(1)
		go func(addr string, c *cs104.Client, q []ScheduleEntry) {
			defer wg.Done()
			m.processQueue(addr, c, q, sigChan)
		}(key, client, queue)
	}

	wg.Wait()
}

func (m *IEC104Master) processQueue(addr string, client *cs104.Client, queue []ScheduleEntry, sigChan chan os.Signal) {
	if ch, ok := m.activeSignals[addr]; ok {
		select {
		case <-ch:
		case <-time.After(5 * time.Second):
			m.logger.Warnf("IEC104 client activation timeout to %s", addr)
		}
	}
	currentTime := 0
	for {
		select {
		case <-sigChan:
			return
		default:
		}

		if len(queue) == 0 {
			if !m.hasRecurrent {
				time.Sleep(500 * time.Millisecond)
				return
			}
			time.Sleep(200 * time.Millisecond)
			continue
		}

		msg := queue[0]
		queue = queue[1:]

		delay := msg.Timestamp - currentTime
		if delay > 0 {
			time.Sleep(time.Duration(delay) * time.Second)
		}
		currentTime = msg.Timestamp

		if err := m.sendWithRetry(client, msg, 10, 1*time.Second); err != nil {
			m.logger.WithError(err).Warnf("Failed to send IEC104 message to %s", addr)
		} else {
			time.Sleep(500 * time.Millisecond)
		}

		if msg.Recurrent && msg.Interval > 0 {
			msg.Timestamp += msg.Interval
			inserted := false
			for i, existing := range queue {
				if msg.Timestamp < existing.Timestamp {
					queue = append(queue[:i], append([]ScheduleEntry{msg}, queue[i:]...)...)
					inserted = true
					break
				}
			}
			if !inserted {
				queue = append(queue, msg)
			}
		}
	}
}

func (m *IEC104Master) sendWithRetry(client *cs104.Client, msg ScheduleEntry, retries int, delay time.Duration) error {
	var lastErr error
	for i := 0; i <= retries; i++ {
		if err := m.sendMessage(client, msg); err != nil {
			lastErr = err
			time.Sleep(delay)
			continue
		}
		return nil
	}
	return lastErr
}

func (m *IEC104Master) sendMessage(client *cs104.Client, msg ScheduleEntry) error {
	coa := asdu.CauseOfTransmission{Cause: asdu.Activation}
	if msg.COT > 0 {
		coa.Cause = asdu.Cause(msg.COT)
	}
	ca := asdu.CommonAddr(msg.CommonAddress)
	typeID := asdu.TypeID(msg.TypeID)

	switch typeID {
	case asdu.C_IC_NA_1:
		return client.InterrogationCmd(coa, ca, asdu.QOIStation)
	case asdu.C_CI_NA_1:
		return client.CounterInterrogationCmd(coa, ca, asdu.QualifierCountCall{
			Request: asdu.QCCTotal,
			Freeze:  asdu.QCCFrzRead,
		})
	case asdu.C_RD_NA_1:
		return client.ReadCmd(coa, ca, asdu.InfoObjAddr(msg.IOA))
	case asdu.C_CS_NA_1:
		return client.ClockSynchronizationCmd(coa, ca, time.Now().UTC())
	case asdu.C_TS_NA_1:
		return client.TestCommand(coa, ca)
	case asdu.C_SC_NA_1, asdu.C_SC_TA_1:
		val := parseBool(msg.Value)
		cmd := asdu.SingleCommandInfo{
			Ioa:   asdu.InfoObjAddr(msg.IOA),
			Value: val,
			Qoc:   asdu.QualifierOfCommand{Qual: asdu.QOCShortPulseDuration, InSelect: false},
			Time:  time.Now().UTC(),
		}
		return asdu.SingleCmd(client, typeID, coa, ca, cmd)
	case asdu.C_SE_NA_1, asdu.C_SE_TA_1:
		value := normalizeFromString(msg.Value)
		cmd := asdu.SetpointCommandNormalInfo{
			Ioa:   asdu.InfoObjAddr(msg.IOA),
			Value: value,
			Qos:   asdu.QualifierOfSetpointCmd{Qual: 0, InSelect: false},
			Time:  time.Now().UTC(),
		}
		return asdu.SetpointCmdNormal(client, typeID, coa, ca, cmd)
	case asdu.C_SE_NB_1, asdu.C_SE_TB_1:
		cmd := asdu.SetpointCommandScaledInfo{
			Ioa:   asdu.InfoObjAddr(msg.IOA),
			Value: int16(parseFloat(msg.Value)),
			Qos:   asdu.QualifierOfSetpointCmd{Qual: 0, InSelect: false},
			Time:  time.Now().UTC(),
		}
		return asdu.SetpointCmdScaled(client, typeID, coa, ca, cmd)
	case asdu.C_SE_NC_1, asdu.C_SE_TC_1:
		cmd := asdu.SetpointCommandFloatInfo{
			Ioa:   asdu.InfoObjAddr(msg.IOA),
			Value: parseFloat(msg.Value),
			Qos:   asdu.QualifierOfSetpointCmd{Qual: 0, InSelect: false},
			Time:  time.Now().UTC(),
		}
		return asdu.SetpointCmdFloat(client, typeID, coa, ca, cmd)
	case asdu.C_DC_NA_1, asdu.C_DC_TA_1:
		cmd := asdu.DoubleCommandInfo{
			Ioa:   asdu.InfoObjAddr(msg.IOA),
			Value: parseDoubleCommand(msg.Value),
			Qoc:   asdu.QualifierOfCommand{Qual: asdu.QOCShortPulseDuration, InSelect: false},
			Time:  time.Now().UTC(),
		}
		return asdu.DoubleCmd(client, typeID, coa, ca, cmd)
	case asdu.C_RC_NA_1, asdu.C_RC_TA_1:
		cmd := asdu.StepCommandInfo{
			Ioa:   asdu.InfoObjAddr(msg.IOA),
			Value: parseStepCommand(msg.Value),
			Qoc:   asdu.QualifierOfCommand{Qual: asdu.QOCShortPulseDuration, InSelect: false},
			Time:  time.Now().UTC(),
		}
		return asdu.StepCmd(client, typeID, coa, ca, cmd)
	default:
		return fmt.Errorf("unsupported type_id %d", msg.TypeID)
	}
}

type iec104ClientHandler struct {
	logger *logrus.Logger
}

func (h *iec104ClientHandler) InterrogationHandler(asdu.Connect, *asdu.ASDU) error {
	h.logger.Info("Received interrogation response")
	return nil
}
func (h *iec104ClientHandler) CounterInterrogationHandler(asdu.Connect, *asdu.ASDU) error {
	h.logger.Info("Received counter interrogation response")
	return nil
}
func (h *iec104ClientHandler) ReadHandler(asdu.Connect, *asdu.ASDU) error {
	h.logger.Info("Received read response")
	return nil
}
func (h *iec104ClientHandler) TestCommandHandler(asdu.Connect, *asdu.ASDU) error {
	h.logger.Info("Received test command response")
	return nil
}
func (h *iec104ClientHandler) ClockSyncHandler(asdu.Connect, *asdu.ASDU) error {
	h.logger.Info("Received clock sync response")
	return nil
}
func (h *iec104ClientHandler) ResetProcessHandler(asdu.Connect, *asdu.ASDU) error {
	h.logger.Info("Received reset process response")
	return nil
}
func (h *iec104ClientHandler) DelayAcquisitionHandler(asdu.Connect, *asdu.ASDU) error {
	h.logger.Info("Received delay acquisition response")
	return nil
}
func (h *iec104ClientHandler) ASDUHandler(_ asdu.Connect, pack *asdu.ASDU) error {
	h.logger.Infof("Received ASDU type=%d cause=%d", pack.Identifier.Type, pack.Coa.Cause)
	return nil
}

func parseBool(value string) bool {
	switch value {
	case "1", "true", "TRUE", "on", "ON", "yes", "YES":
		return true
	default:
		return false
	}
}

func parseFloat(value string) float32 {
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return 0
	}
	return float32(parsed)
}

func normalizeFromString(value string) asdu.Normalize {
	f := float64(parseFloat(value))
	if f > 1 {
		f = 1
	}
	if f < -1 {
		f = -1
	}
	return asdu.Normalize(int16(f * 32768))
}

func parseDoubleCommand(value string) asdu.DoubleCommand {
	switch value {
	case "on", "ON", "1", "true", "TRUE":
		return asdu.DCOOn
	case "off", "OFF", "0", "false", "FALSE":
		return asdu.DCOOff
	default:
		return asdu.DCOOff
	}
}

func parseStepCommand(value string) asdu.StepCommand {
	switch value {
	case "up", "UP", "1", "true", "TRUE":
		return asdu.SCOStepUP
	default:
		return asdu.SCOStepDown
	}
}

func main() {
	configFile := "/app/config/master.yaml"
	if envConfig := os.Getenv("MASTER_CONFIG"); envConfig != "" {
		configFile = envConfig
	}

	master, err := NewIEC104Master(configFile)
	if err != nil {
		log.Fatalf("Failed to init IEC104 master: %v", err)
	}

	if err := master.setup(); err != nil {
		log.Fatalf("Failed to setup IEC104 master: %v", err)
	}

	master.loop()
}
