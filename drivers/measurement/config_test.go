package main

import (
	"errors"
	"testing"
)

func TestNewConfig(t *testing.T) {
	var configTests = []struct{
		name string
		msource string
		ltype string
		laddr string
		expected MeasurementConfig
		validateErr func(t *testing.T, err error)
	}{
		{
			"Unix volume",
			"volume",
			"unix",
			"/tmp/list.sock",
			MeasurementConfig{
				MeasurementSource: SourceVolume,
				ListenerType: ListenerUnix,
				ListenerAddr: "/tmp/list.sock",
			},
			nil,
		},
		{
			"tcp temp",
			"temp",
			"tcp",
			":8080",
			MeasurementConfig{
				MeasurementSource: SourceTemp,
				ListenerType: ListenerTcp,
				ListenerAddr: ":8080",
			},
			nil,
		},
		{
			"Upper-case source",
			"TEMP",
			"tcp",
			":8080",
			MeasurementConfig{
				MeasurementSource: SourceTemp,
				ListenerType: ListenerTcp,
				ListenerAddr: ":8080",
			},
			nil,
		},
		{
			"Invalid source",
			"unknown",
			"tcp",
			":8080",
			MeasurementConfig{},
			func(t *testing.T, err error) {
				var target UnknownSourceError
				if !errors.As(err, &target) {
					t.Errorf("expected UnknownSourceError, got: %T (%v)", err, err)
				}
				if target.Source != "unknown" {
					t.Errorf("expected embedded Source to be 'unknown', got %q", target.Source)
				}
			},
		},
	}

	for _, ct := range configTests {
	t.Run(ct.name, func(t *testing.T) {
		t.Setenv("MEASUREMENT_SOURCE", ct.msource)
		t.Setenv("MEASUREMENT_LISTENER_TYPE", ct.ltype)
		t.Setenv("MEASUREMENT_LISTENER_ADDRESS", ct.laddr)
		got, err := NewConfig()
		if ct.validateErr != nil {
			ct.validateErr(t, err)
		} else {
			if err != nil {
				t.Errorf("Unexpected error loading config: %v", err)
			}
		}
		if got != ct.expected {
			t.Errorf("Wanted: %q, got %q", ct.expected, got)
		}
	})
}
}