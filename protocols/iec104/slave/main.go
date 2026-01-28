package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/juanjorosendo/go-iecp5/asdu"
	"github.com/juanjorosendo/go-iecp5/cs104"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type FileConfig struct {
	Protocol string        `yaml:"protocol"`
	Node     IEC104NodeCfg `yaml:"node"`
}

type IEC104NodeCfg struct {
	IP                 string             `yaml:"ip"`
	Port               int                `yaml:"port"`
	TickRateMS         int                `yaml:"tick_rate_ms"`
	SelectTimeoutMS    int                `yaml:"select_timeout_ms"`
	MaxConnections     int                `yaml:"max_connections"`
	ProtocolParameters ProtocolParameters `yaml:"protocol_parameters"`
	Stations           []StationConfig    `yaml:"stations"`
	AuthorizedMasters  []int              `yaml:"authorized_masters"`
}

type ProtocolParameters struct {
	T1 int `yaml:"t1"`
	T2 int `yaml:"t2"`
	T3 int `yaml:"t3"`
	K  int `yaml:"k"`
	W  int `yaml:"w"`
}

type StationConfig struct {
	CommonAddress int                      `yaml:"common_address"`
	Points        map[string][]PointConfig `yaml:"points"`
}

type PointConfig struct {
	IOA        int         `yaml:"ioa"`
	Value      interface{} `yaml:"value"`
	ReportMS   int         `yaml:"report_ms"`
	RelatedIOA int         `yaml:"related_ioa"`
}

type PointState struct {
	Type       asdu.TypeID
	Value      interface{}
	ReportMS   int
	RelatedIOA int
	LastReport time.Time
}

type IEC104Server struct {
	config      IEC104NodeCfg
	logger      *logrus.Logger
	server      *cs104.Server
	points      map[int]map[int]*PointState
	connections map[asdu.Connect]struct{}
	connMu      sync.Mutex
	running     bool
}

func NewIEC104Server(configPath string) (*IEC104Server, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var wrapped FileConfig
	if err := yaml.Unmarshal(data, &wrapped); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	return &IEC104Server{
		config:      wrapped.Node,
		logger:      logger,
		points:      make(map[int]map[int]*PointState),
		connections: make(map[asdu.Connect]struct{}),
	}, nil
}

func (s *IEC104Server) setup() {
	s.server = cs104.NewServer(s)
	s.server.LogMode(false)
	s.server.SetOnConnectionHandler(func(c asdu.Connect) {
		s.connMu.Lock()
		if s.config.MaxConnections > 0 && len(s.connections) >= s.config.MaxConnections {
			s.connMu.Unlock()
			remoteAddr := "<unknown>"
			if conn := c.UnderlyingConn(); conn != nil {
				remoteAddr = conn.RemoteAddr().String()
				_ = conn.Close()
			}
			s.logger.Warnf("Rejected IEC104 connection from %s: max connections reached", remoteAddr)
			return
		}
		s.connections[c] = struct{}{}
		s.connMu.Unlock()
		s.logger.Infof("IEC104 connection established from %v", c)
	})
	s.server.SetConnectionLostHandler(func(c asdu.Connect) {
		s.connMu.Lock()
		delete(s.connections, c)
		s.connMu.Unlock()
		s.logger.Infof("IEC104 connection lost from %v", c)
	})

	cfg := cs104.DefaultConfig()
	if s.config.ProtocolParameters.T1 > 0 {
		cfg.SendUnAckTimeout1 = time.Duration(s.config.ProtocolParameters.T1) * time.Second
	}
	if s.config.ProtocolParameters.T2 > 0 {
		cfg.RecvUnAckTimeout2 = time.Duration(s.config.ProtocolParameters.T2) * time.Second
	}
	if s.config.ProtocolParameters.T3 > 0 {
		cfg.IdleTimeout3 = time.Duration(s.config.ProtocolParameters.T3) * time.Second
	}
	if s.config.ProtocolParameters.K > 0 {
		cfg.SendUnAckLimitK = uint16(s.config.ProtocolParameters.K)
	}
	if s.config.ProtocolParameters.W > 0 {
		cfg.RecvUnAckLimitW = uint16(s.config.ProtocolParameters.W)
	}
	s.server.SetConfig(cfg)

	s.buildPointMap()
}

func (s *IEC104Server) buildPointMap() {
	for _, station := range s.config.Stations {
		if _, ok := s.points[station.CommonAddress]; !ok {
			s.points[station.CommonAddress] = make(map[int]*PointState)
		}
		for group, list := range station.Points {
			for _, point := range list {
				state := &PointState{
					Type:       mapPointType(group),
					Value:      point.Value,
					ReportMS:   point.ReportMS,
					RelatedIOA: point.RelatedIOA,
				}
				s.points[station.CommonAddress][point.IOA] = state
			}
		}
	}
}

func mapPointType(group string) asdu.TypeID {
	switch group {
	case "single_points":
		return asdu.M_SP_NA_1
	case "single_points_time":
		return asdu.M_SP_TB_1
	case "double_points":
		return asdu.M_DP_NA_1
	case "measured_normalized":
		return asdu.M_ME_NA_1
	case "measured_scaled":
		return asdu.M_ME_NB_1
	case "measured_short":
		return asdu.M_ME_NC_1
	case "step_positions":
		return asdu.M_ST_NA_1
	case "measured_short_time":
		return asdu.M_ME_TF_1
	case "single_commands":
		return asdu.C_SC_NA_1
	case "single_commands_time":
		return asdu.C_SC_TA_1
	case "setpoint_short":
		return asdu.C_SE_NC_1
	case "setpoint_short_time":
		return asdu.C_SE_TC_1
	case "setpoint_normalized":
		return asdu.C_SE_NA_1
	case "setpoint_scaled":
		return asdu.C_SE_NB_1
	default:
		return asdu.M_SP_NA_1
	}
}

func (s *IEC104Server) run() error {
	addr := fmt.Sprintf("%s:%d", s.config.IP, s.config.Port)
	s.running = true
	go s.startReporter()
	s.logger.Infof("IEC104 server listening on %s", addr)
	s.server.ListenAndServer(addr)
	return nil
}

func (s *IEC104Server) startReporter() {
	interval := 500 * time.Millisecond
	if s.config.TickRateMS > 0 {
		interval = time.Duration(s.config.TickRateMS) * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for s.running {
		<-ticker.C
		now := time.Now().UTC()
		for ca, points := range s.points {
			for ioa, state := range points {
				if state.ReportMS <= 0 {
					continue
				}
				if state.LastReport.IsZero() || now.Sub(state.LastReport) >= time.Duration(state.ReportMS)*time.Millisecond {
					s.broadcastPoint(ca, ioa, state, asdu.Spontaneous)
					state.LastReport = now
				}
			}
		}
	}
}

func (s *IEC104Server) broadcastPoint(ca int, ioa int, state *PointState, cause asdu.Cause) {
	s.connMu.Lock()
	defer s.connMu.Unlock()
	for conn := range s.connections {
		_ = s.sendPoint(conn, ca, ioa, state, cause)
	}
}

func (s *IEC104Server) sendPoint(conn asdu.Connect, ca int, ioa int, state *PointState, cause asdu.Cause) error {
	coa := asdu.CauseOfTransmission{Cause: cause}
	commonAddr := asdu.CommonAddr(ca)
	infoObjAddr := asdu.InfoObjAddr(ioa)

	switch state.Type {
	case asdu.M_SP_NA_1, asdu.M_SP_TB_1:
		info := asdu.SinglePointInfo{
			Ioa:   infoObjAddr,
			Value: parseBool(state.Value),
			Qds:   asdu.QDSGood,
			Time:  time.Now().UTC(),
		}
		if state.Type == asdu.M_SP_TB_1 {
			return asdu.SingleCP56Time2a(conn, coa, commonAddr, info)
		}
		return asdu.Single(conn, false, coa, commonAddr, info)
	case asdu.M_DP_NA_1:
		info := asdu.DoublePointInfo{
			Ioa:   infoObjAddr,
			Value: parseDouble(state.Value),
			Qds:   asdu.QDSGood,
			Time:  time.Now().UTC(),
		}
		return asdu.Double(conn, false, coa, commonAddr, info)
	case asdu.M_ME_NA_1:
		info := asdu.MeasuredValueNormalInfo{
			Ioa:   infoObjAddr,
			Value: normalizeFromValue(state.Value),
			Qds:   asdu.QDSGood,
			Time:  time.Now().UTC(),
		}
		return asdu.MeasuredValueNormal(conn, false, coa, commonAddr, info)
	case asdu.M_ME_NB_1:
		info := asdu.MeasuredValueScaledInfo{
			Ioa:   infoObjAddr,
			Value: int16(parseFloat(state.Value)),
			Qds:   asdu.QDSGood,
			Time:  time.Now().UTC(),
		}
		return asdu.MeasuredValueScaled(conn, false, coa, commonAddr, info)
	case asdu.M_ME_NC_1, asdu.M_ME_TF_1:
		info := asdu.MeasuredValueFloatInfo{
			Ioa:   infoObjAddr,
			Value: parseFloat(state.Value),
			Qds:   asdu.QDSGood,
			Time:  time.Now().UTC(),
		}
		if state.Type == asdu.M_ME_TF_1 {
			return asdu.MeasuredValueFloatCP56Time2a(conn, coa, commonAddr, info)
		}
		return asdu.MeasuredValueFloat(conn, false, coa, commonAddr, info)
	case asdu.M_ST_NA_1:
		info := asdu.StepPositionInfo{
			Ioa:   infoObjAddr,
			Value: asdu.StepPosition{Val: int(parseFloat(state.Value))},
			Qds:   asdu.QDSGood,
			Time:  time.Now().UTC(),
		}
		return asdu.Step(conn, false, coa, commonAddr, info)
	default:
		return fmt.Errorf("unsupported point type %d", state.Type)
	}
}

func (s *IEC104Server) InterrogationHandler(c asdu.Connect, asduPack *asdu.ASDU, qoi asdu.QualifierOfInterrogation) error {
	_ = asduPack.SendReplyMirror(c, asdu.ActivationCon)
	commonAddr := int(asduPack.CommonAddr)
	for ioa, state := range s.points[commonAddr] {
		_ = s.sendPoint(c, commonAddr, ioa, state, asdu.InterrogatedByStation)
	}
	return asduPack.SendReplyMirror(c, asdu.ActivationTerm)
}

func (s *IEC104Server) CounterInterrogationHandler(c asdu.Connect, pack *asdu.ASDU, qcc asdu.QualifierCountCall) error {
	_ = pack.SendReplyMirror(c, asdu.ActivationCon)
	return pack.SendReplyMirror(c, asdu.ActivationTerm)
}

func (s *IEC104Server) ReadHandler(c asdu.Connect, pack *asdu.ASDU, ioa asdu.InfoObjAddr) error {
	commonAddr := int(pack.CommonAddr)
	state, ok := s.points[commonAddr][int(ioa)]
	if !ok {
		return pack.SendReplyMirror(c, asdu.UnknownIOA)
	}
	return s.sendPoint(c, commonAddr, int(ioa), state, asdu.Request)
}

func (s *IEC104Server) ClockSyncHandler(c asdu.Connect, pack *asdu.ASDU, t time.Time) error {
	return pack.SendReplyMirror(c, asdu.ActivationCon)
}

func (s *IEC104Server) ResetProcessHandler(c asdu.Connect, pack *asdu.ASDU, qrp asdu.QualifierOfResetProcessCmd) error {
	return pack.SendReplyMirror(c, asdu.ActivationCon)
}

func (s *IEC104Server) DelayAcquisitionHandler(c asdu.Connect, pack *asdu.ASDU, msec uint16) error {
	return pack.SendReplyMirror(c, asdu.ActivationCon)
}

func (s *IEC104Server) ASDUHandler(c asdu.Connect, pack *asdu.ASDU) error {
	switch pack.Identifier.Type {
	case asdu.C_SC_NA_1, asdu.C_SC_TA_1:
		cmd := pack.Clone().GetSingleCmd()
		if err := s.applyCommandValue(c, pack, int(cmd.Ioa), cmd.Value); err != nil {
			return err
		}
		if err := pack.SendReplyMirror(c, asdu.ActivationCon); err != nil {
			return err
		}
		return pack.SendReplyMirror(c, asdu.ActivationTerm)
	case asdu.C_SE_NA_1, asdu.C_SE_TA_1:
		cmd := pack.Clone().GetSetpointNormalCmd()
		if err := s.applyCommandValue(c, pack, int(cmd.Ioa), cmd.Value.Float64()); err != nil {
			return err
		}
		if err := pack.SendReplyMirror(c, asdu.ActivationCon); err != nil {
			return err
		}
		return pack.SendReplyMirror(c, asdu.ActivationTerm)
	case asdu.C_SE_NB_1, asdu.C_SE_TB_1:
		cmd := pack.Clone().GetSetpointCmdScaled()
		if err := s.applyCommandValue(c, pack, int(cmd.Ioa), int16(cmd.Value)); err != nil {
			return err
		}
		if err := pack.SendReplyMirror(c, asdu.ActivationCon); err != nil {
			return err
		}
		return pack.SendReplyMirror(c, asdu.ActivationTerm)
	case asdu.C_SE_NC_1, asdu.C_SE_TC_1:
		cmd := pack.Clone().GetSetpointFloatCmd()
		if err := s.applyCommandValue(c, pack, int(cmd.Ioa), float32(cmd.Value)); err != nil {
			return err
		}
		if err := pack.SendReplyMirror(c, asdu.ActivationCon); err != nil {
			return err
		}
		return pack.SendReplyMirror(c, asdu.ActivationTerm)
	case asdu.C_DC_NA_1, asdu.C_DC_TA_1:
		cmd := pack.Clone().GetDoubleCmd()
		if err := s.applyCommandValue(c, pack, int(cmd.Ioa), cmd.Value); err != nil {
			return err
		}
		if err := pack.SendReplyMirror(c, asdu.ActivationCon); err != nil {
			return err
		}
		return pack.SendReplyMirror(c, asdu.ActivationTerm)
	case asdu.C_RC_NA_1, asdu.C_RC_TA_1:
		cmd := pack.Clone().GetStepCmd()
		if err := s.applyCommandValue(c, pack, int(cmd.Ioa), cmd.Value); err != nil {
			return err
		}
		if err := pack.SendReplyMirror(c, asdu.ActivationCon); err != nil {
			return err
		}
		return pack.SendReplyMirror(c, asdu.ActivationTerm)
	default:
		return nil
	}
}

func (s *IEC104Server) applyCommandValue(c asdu.Connect, pack *asdu.ASDU, ioa int, value interface{}) error {
	commonAddr := int(pack.CommonAddr)
	state, ok := s.points[commonAddr][ioa]
	if !ok {
		return pack.SendReplyMirror(c, asdu.UnknownIOA)
	}

	state.Value = value
	if state.RelatedIOA != 0 {
		if related, ok := s.points[commonAddr][state.RelatedIOA]; ok {
			related.Value = value
			s.broadcastPoint(commonAddr, state.RelatedIOA, related, asdu.Spontaneous)
		}
	}

	return nil
}

func parseBool(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	case string:
		return v == "1" || v == "true" || v == "TRUE" || v == "on" || v == "ON"
	default:
		return false
	}
}

func parseFloat(value interface{}) float32 {
	switch v := value.(type) {
	case float32:
		return v
	case float64:
		return float32(v)
	case int:
		return float32(v)
	case int64:
		return float32(v)
	case string:
		parsed, err := strconv.ParseFloat(v, 32)
		if err != nil {
			return 0
		}
		return float32(parsed)
	default:
		return 0
	}
}

func normalizeFromValue(value interface{}) asdu.Normalize {
	f := float64(parseFloat(value))
	if f > 1 {
		f = 1
	}
	if f < -1 {
		f = -1
	}
	return asdu.Normalize(int16(f * 32768))
}

func parseDouble(value interface{}) asdu.DoublePoint {
	switch v := value.(type) {
	case string:
		if v == "ON" || v == "on" || v == "1" {
			return asdu.DPIDeterminedOn
		}
		return asdu.DPIDeterminedOff
	case int:
		if v != 0 {
			return asdu.DPIDeterminedOn
		}
		return asdu.DPIDeterminedOff
	default:
		return asdu.DPIDeterminedOff
	}
}

func parseStepCommand(value interface{}) asdu.StepCommand {
	switch v := value.(type) {
	case asdu.StepCommand:
		return v
	case int:
		if v > 0 {
			return asdu.SCOStepUP
		}
		return asdu.SCOStepDown
	case string:
		if v == "up" || v == "UP" || v == "1" {
			return asdu.SCOStepUP
		}
		return asdu.SCOStepDown
	default:
		return asdu.SCOStepDown
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	configFile := "/app/config/slave.yaml"
	if envConfig := os.Getenv("SLAVE_CONFIG"); envConfig != "" {
		configFile = envConfig
	}

	server, err := NewIEC104Server(configFile)
	if err != nil {
		log.Fatalf("Failed to init IEC104 server: %v", err)
	}
	server.setup()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-stop
		server.running = false
		os.Exit(0)
	}()

	if err := server.run(); err != nil {
		log.Fatalf("IEC104 server error: %v", err)
	}
}
