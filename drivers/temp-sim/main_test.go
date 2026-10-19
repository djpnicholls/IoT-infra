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
)

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
