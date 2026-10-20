package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

type Reading interface {
	Timestamp() int64
	Value() float64
}

type MeasurementService interface {
	GetReading() Reading
}

type MeasurementServer struct {
	Listener net.Listener
	service  MeasurementService
}

func (s *MeasurementServer) Serve(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		_ = s.Listener.Close()
	}()
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			continue
		}
		go s.handleConn(ctx, conn, time.Second*5)

	}
}

func (s *MeasurementServer) handleConn(ctx context.Context, c net.Conn, interval time.Duration) {
	defer c.Close()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reading := s.service.GetReading()
			enc := json.NewEncoder(c)
			_ = enc.Encode(reading)
		}
	}
}

func GetListener() (net.Listener, error) {
	if os.Getenv("LISTEN_PID") == strconv.Itoa(os.Getpid()) {
		log.Println("Running under systemd")
		socketFile := os.NewFile(3, "systemd-socket")
		return net.FileListener(socketFile)
	} else {
		socketFile := "/tmp/temp_s.sock"
		os.Remove(socketFile)
		log.Printf("Opening socket file %s", socketFile)
		return net.Listen("unix", socketFile)
	}
}

func NewServer(ln net.Listener) *MeasurementServer {
	measurementSource := os.Getenv("MEASUREMENT_SOURCE")
	switch measurementSource {
	case "volume":
		return NewSimulatedVolumeServer(ln)
	case "temp":
		return NewSimulatedTemperatureServer(ln)
	default:
		log.Fatalf("Unrecognised measurement source: %q", measurementSource)
	}
	return nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ln, err := GetListener()
	if err != nil {
		log.Fatalf("Error listening on socket: %s", err)
	}

	log.Print("Now serving...")
	s := NewServer(ln)
	if err = s.Serve(ctx); err != nil {
		log.Fatalf("Failed to Serve: %s", err)
	}

}
