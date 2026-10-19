package main

import (
	"errors"
	"context"
	"io"
	"net"
	"testing"
	"time"
	"bytes"
	"strings"
	"encoding/json"
	"math"
)

func TestSimulatedTemperatureServer(t *testing.T) {
	s := NewSimulatedTemperatureServer(nil)
	var readings []Reading
	for range 100 {
		readings = append(readings, s.service.GetReading())
	}
	if len(readings) != 100 {
		t.Errorf("Only got %d readings", len(readings))
	}
	var unchanged int
	for i := range readings{
		if i == 0{
			continue
		}
		current := readings[i].Value()
		last := readings[i - 1].Value()
		if math.Abs(current - last) > 0.5 {
			t.Errorf("Reading not within half a degree of the last reading.  Current: %g, Last: %g", current, last)
		}
		if current == last {
			unchanged ++
		}
	}
	if unchanged > 5 {
		t.Errorf("There were %d cases where the current reading matched the previous", unchanged)
	}
}

func TestSimulateTempReading(t *testing.T) {
	var results []float64
	for range 100 {
		results = append(results, SimulateTempReading())
	}
	var in0to1 int
	for _, r := range results {
		if r > 0 && r < 1 {
			in0to1++
		}
		if r > 30 || r < -10 {
			t.Errorf("Result out of range: %f", r)
		}
	}
	if in0to1 >= 25 {
		t.Errorf("%d/100 results were in the 0 - 1 range", in0to1)
	}
}

func TestHandleConn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, client := net.Pipe()
	defer srv.Close()
	defer client.Close()

	s := NewSimulatedTemperatureServer(nil)
	go s.handleConn(ctx, srv, time.Millisecond*50)

	var buf bytes.Buffer

	client.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
	_, err := io.Copy(&buf, client)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout(){
			t.Logf("timed out waiting for data: %v", err)
		} else {
			t.Errorf("error: %v", err)
		}
	}

	results := strings.Split(buf.String(), "\n")
	if !(len(results) > 1) {
		t.Errorf("Not enough results: %q", len(results))
	}
	var reading TemperatureReading
	err = json.Unmarshal([]byte(results[0]), &reading)
	if err != nil {
		t.Errorf("First result doesn't unmarshal to a reading: . Result: %q", results[0])
	}

	t.Logf("results: %q", results)

}
