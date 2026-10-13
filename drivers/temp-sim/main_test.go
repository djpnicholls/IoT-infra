package main

import (
	"errors"
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
	"bytes"
	"strings"
)

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

func TestCreatePayload(t *testing.T) {
	// Unix time: 1000000000
	moment := time.Date(2001, time.September, 9, 1, 46, 40, 0, time.UTC)

	now := time.Now()
	uNow := now.Unix()

	var payloadTests = []struct {
		instant  time.Time
		reading  float64
		expected string
	}{
		{moment, 10.0, "{time: 1000000000, temp: 10.00}\n"},
		{moment, -3.57, "{time: 1000000000, temp: -3.57}\n"},
		{moment, 3.5798, "{time: 1000000000, temp: 3.58}\n"},
		{moment, 20.4, "{time: 1000000000, temp: 20.40}\n"},
		{now, 20.4, fmt.Sprintf("{time: %d, temp: 20.40}\n", uNow)},
	}
	for _, pt := range payloadTests {
		t.Run(pt.expected, func(t *testing.T) {
			got := CreatePayload(pt.instant, pt.reading)
			if pt.expected != got {
				t.Errorf("wanted: %q, got %q", pt.expected, got)
			}

		})
	}
}

func TestHandleConn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, client := net.Pipe()
	defer srv.Close()
	defer client.Close()

	go handleConn(ctx, srv, time.Millisecond*50)

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
	if !strings.Contains(results[0], "{time: ") {
		t.Errorf("First result doesn't contain '{time: . Result: %q", results[0])
	}

	t.Logf("results: %q", results)

}
