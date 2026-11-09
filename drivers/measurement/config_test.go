package main

import (
	"errors"
	"testing"
)

func TestNewConfig(t *testing.T) {
	var configTests = []struct {
		name        string
		ltype       string
		laddr       string
		expected    MeasurementConfig
		validateErr func(t *testing.T, err error)
	}{
		{
			"Unix socket",
			"unix",
			"/tmp/list.sock",
			MeasurementConfig{
				ListenerType:      ListenerUnix,
				ListenerAddr:      "/tmp/list.sock",
			},
			nil,
		},
		{
			"tcp socket",
			"tcp",
			":8080",
			MeasurementConfig{
				ListenerType:      ListenerTcp,
				ListenerAddr:      ":8080",
			},
			nil,
		},
		{
			"Invalid listener",
			"unknown",
			":8080",
			MeasurementConfig{},
			func(t *testing.T, err error) {
				var target UnknownListenerTypeError
				if !errors.As(err, &target) {
					t.Errorf("expected UnknownListenerTypeError, got: %T (%v)", err, err)
				}
				if target.LType != "unknown" {
					t.Errorf("expected embedded listener type to be 'unknown', got %q", target.LType)
				}
			},
		},
		{
			"Missing listener",
			"tcp",
			"",
			MeasurementConfig{},
			func(t *testing.T, err error) {
				var target MissingListenerAddressError
				if !errors.As(err, &target) {
					t.Errorf("expected MissingListenerAddressError got: %T (%v)", err, err)
				}
			},
		},
	}

	for _, ct := range configTests {
		t.Run(ct.name, func(t *testing.T) {
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
