package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"strings"
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
	go s.handleConn(ctx, srv, time.Millisecond*50)

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
		t.Errorf("Not enough results: %q", len(results))
	}
	var reading TestReading
	err = json.Unmarshal([]byte(results[0]), &reading)
	if err != nil {
		t.Errorf("First result doesn't unmarshal to a reading: . Result: %q", results[0])
	}

	t.Logf("results: %q", results)
	

	client.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	cancel()

	var newBuf bytes.Buffer
	_, err = io.Copy(&newBuf, client)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			t.Logf("timed out waiting for data: %v", err)
		} else {
			t.Errorf("error: %v", err)
		}
	}
	if "" != newBuf.String() {
		t.Errorf("Context not closed properly.  Received results: %q", newBuf.String())
	}

}
