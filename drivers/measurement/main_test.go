package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)
func TestGetListener(t *testing.T) {
	tempdir := t.TempDir()
	unixAddr := filepath.Join(tempdir, "test.sock")

	t.Run("unix", func(t *testing.T) {
		cfg := MeasurementConfig{
			ListenerType: ListenerUnix,
			ListenerAddr: unixAddr,
		}
		ln, err := GetListener(cfg)
		if err != nil {
			t.Fatalf("Error setting up listener: %v", err)
		}
		defer ln.Close()
		if _, ok := ln.(*net.UnixListener); !ok {
			t.Errorf("Expected unix listener, got %T", ln)
		}
		if ln.Addr().String() != unixAddr {
			t.Errorf("Unexpected listener address.  Expected: %v, got %v", unixAddr, ln.Addr().String())
		}

	})
	t.Run("tcp", func(t *testing.T) {
		cfg := MeasurementConfig{
			ListenerType: ListenerTcp,
			ListenerAddr: "127.0.0.1:0",
		}
		ln, err := GetListener(cfg)
		if err != nil {
			t.Fatalf("Error setting up listener: %v", err)
		}
		defer ln.Close()
		if _, ok := ln.(*net.TCPListener); !ok {
			t.Errorf("Expected tcp listener, got %T", ln)
		}
		ip, port, _ := strings.Cut(ln.Addr().String(), ":")
		if ip != "127.0.0.1" {
			t.Errorf("Unexpected listener address.  Expected next open port on IP 127.0.0.1, got %v:%v", ip, port)
		}
	})
}

func TestRunMetricsServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := &MeasurementConfig{
		ListenerType: ListenerTcp,
		ListenerAddr: ":8091",
	}
	if err := ConfigureListener(cfg); err != nil {
		t.Fatalf("Error configuring listener: %v", err)
	}

	errCh := make(chan error, 1)
	go func() { errCh <- RunMetricsServer(ctx, cfg) }()
	tempCh := make(chan float64, 30)
	volCh := make(chan float64, 30)

	go func() {
		for range 100 {

			resp, err := http.Get(fmt.Sprintf("http://%s/", cfg.ln.Addr().String()))
			if err != nil {
				t.Fatalf("Unexpected error calling metrics server: %v", err)
			}
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("Failed to read body: %v", err)
			}

			temp, err := parseMetricValues(t, io.NopCloser(bytes.NewReader(bodyBytes)), "room_temperature_celsius", -10, 40)
			if err != nil {
				t.Errorf("Error scraping temp metric: %v\nResponse: \n%s", err, string(bodyBytes))
			}
			for _, v := range temp {
				tempCh <- v
			}
			vol, err := parseMetricValues(t, io.NopCloser(bytes.NewReader(bodyBytes)), "room_volume_decibels", -10, 60)
			if err != nil {
				t.Errorf("Error scraping volume metric: %v\nResponse: \n%s", err, string(bodyBytes))
			}
			for _, v := range vol {
				volCh <- v
			}
		}

		close(tempCh)
		close(volCh)
		cancel()
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	var lastTemp float64
	var unchangedTemp int
	go func() {
		for i := range tempCh {
			if lastTemp == 0 {
				lastTemp = i
				continue
			}
			if i == lastTemp {
				unchangedTemp++
			}
			lastTemp = i
		}
		wg.Done()
	}()

	var lastVol float64
	var unchangedVol int
	go func() {
		for i := range volCh {
			if lastVol == 0 {
				lastVol = i
				continue
			}
			if i == lastVol {
				unchangedVol++
			}
			lastVol = i
		}
		wg.Done()
	}()

	wg.Wait()

	if unchangedTemp > 10 {
		t.Errorf("%d temperature readings unchanged", unchangedTemp)
	}
	if unchangedVol > 10 {
		t.Errorf("%d volume readings unchanged. Last reading: %f", unchangedVol, lastVol)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("unexpected server error: %v", err)
	}
}
