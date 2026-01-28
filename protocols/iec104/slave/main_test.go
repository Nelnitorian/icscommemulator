package main

import (
	"testing"

	"github.com/juanjorosendo/go-iecp5/asdu"
)

func TestMapPointType(t *testing.T) {
	if got := mapPointType("single_points"); got != asdu.M_SP_NA_1 {
		t.Fatalf("unexpected type for single_points: %v", got)
	}
	if got := mapPointType("measured_short"); got != asdu.M_ME_NC_1 {
		t.Fatalf("unexpected type for measured_short: %v", got)
	}
}

func TestMapPointTypeAllGroups(t *testing.T) {
	cases := map[string]asdu.TypeID{
		"single_points":        asdu.M_SP_NA_1,
		"single_points_time":   asdu.M_SP_TB_1,
		"double_points":        asdu.M_DP_NA_1,
		"measured_normalized":  asdu.M_ME_NA_1,
		"measured_scaled":      asdu.M_ME_NB_1,
		"measured_short":       asdu.M_ME_NC_1,
		"step_positions":       asdu.M_ST_NA_1,
		"measured_short_time":  asdu.M_ME_TF_1,
		"single_commands":      asdu.C_SC_NA_1,
		"single_commands_time": asdu.C_SC_TA_1,
		"setpoint_short":       asdu.C_SE_NC_1,
		"setpoint_short_time":  asdu.C_SE_TC_1,
		"setpoint_normalized":  asdu.C_SE_NA_1,
		"setpoint_scaled":      asdu.C_SE_NB_1,
	}

	for group, expected := range cases {
		if got := mapPointType(group); got != expected {
			t.Fatalf("unexpected type for %s: %v", group, got)
		}
	}
}

func TestParseHelpers(t *testing.T) {
	if !parseBool("true") {
		t.Fatal("expected parseBool true")
	}
	if parseBool("0") {
		t.Fatal("expected parseBool false")
	}

	if parseFloat("2.5") != 2.5 {
		t.Fatal("expected parseFloat 2.5")
	}
	if parseFloat("bad") != 0 {
		t.Fatal("expected parseFloat 0 for bad input")
	}
}

func TestBuildPointMap(t *testing.T) {
	server := &IEC104Server{
		config: IEC104NodeCfg{
			Stations: []StationConfig{
				{
					CommonAddress: 1,
					Points: map[string][]PointConfig{
						"single_points": {
							{IOA: 1001, Value: true, ReportMS: 0},
						},
						"measured_short": {
							{IOA: 2001, Value: 12.5, ReportMS: 1000},
						},
					},
				},
			},
		},
		points: make(map[int]map[int]*PointState),
	}

	server.buildPointMap()
	if len(server.points[1]) != 2 {
		t.Fatalf("expected 2 points, got %d", len(server.points[1]))
	}
	if _, ok := server.points[1][1001]; !ok {
		t.Fatal("expected point 1001 to exist")
	}
}
