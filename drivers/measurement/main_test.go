package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type TestReading struct {
	Time    int64   `json:"time"`
	TestVal float64 `json:"value"`
}

func (r TestReading) Value() float64   { return r.TestVal }
func (r TestReading) Timestamp() int64 { return r.Time }

type TestMeasurementService struct {
	Calls    int
	Readings []Reading
}

func (s *TestMeasurementService) GetReading() Reading {
	result := s.Readings[s.Calls]
	s.Calls++
	return result
}
func (s *TestMeasurementService) SetReadings(readings []TestReading) {
	for _, reading := range readings {
		s.Readings = append(s.Readings, reading)
	}
}

func NewTestMeasurementServer(testReadings []TestReading) *MeasurementServer {
	testService := &TestMeasurementService{}
	testService.SetReadings(testReadings)
	return &MeasurementServer{
		Listener: nil,
		service:  testService,
	}
}

func TestHandleConn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, client := net.Pipe()
	defer srv.Close()
	defer client.Close()

	s := NewTestMeasurementServer([]TestReading{
		{Time: 1,
			TestVal: 0.1},
		{Time: 2,
			TestVal: 0.2},
		{Time: 3,
			TestVal: 0.3},
		{Time: 4,
			TestVal: 0.4},
		{Time: 5,
			TestVal: 0.5},
		{Time: 6,
			TestVal: 0.6},
	})

	done := make(chan struct{})
	go func() { defer close(done); s.handleConn(ctx, srv, time.Millisecond*50) }()

	t.Run("serves measurement data", func(t *testing.T) {
		var buf bytes.Buffer
		client.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		_, err := io.Copy(&buf, client)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				t.Logf("timed out waiting for data: %v", err)
			} else {
				t.Errorf("error: %v", err)
			}
		}

		results := strings.Split(buf.String(), "\n")
		if !(len(results) > 1) {
			t.Errorf("Not enough results: %d", len(results))
		}

		var reading TestReading
		err = json.Unmarshal([]byte(results[0]), &reading)
		if err != nil {
			t.Errorf("First result doesn't unmarshal to a reading: . Result: %q", results[0])
		}
		t.Logf("results: %q", results)
	})

	t.Run("Context cancels", func(t *testing.T) {
		client.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		cancel()

		var newBuf bytes.Buffer
		_, err := io.Copy(&newBuf, client)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				t.Logf("timed out waiting for data: %v", err)
			} else {
				t.Errorf("error: %v", err)
			}
		}
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("handleConn did not return after cancel")
		}
	})
}

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

			temp, err := parseMetricValues(t, io.NopCloser(bytes.NewReader(bodyBytes)), "room_temperature_celsius", -10, 30)
			if err != nil {
				t.Errorf("Error scraping temp metric: %v\nResponse: \n%s", err, string(bodyBytes))
			}
			for _, v := range temp {
				tempCh <- v
			}
			vol, err := parseMetricValues(t, io.NopCloser(bytes.NewReader(bodyBytes)), "room_volume_decibels", -10, 40)
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
