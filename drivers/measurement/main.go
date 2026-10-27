package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"os"
	"os/signal"
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

func GetListener(cfg MeasurementConfig) (net.Listener, error) {
	if cfg.ListenerType == ListenerSystemd {
		log.Println("Running under systemd")
		socketFile := os.NewFile(3, cfg.ListenerAddr)
		return net.FileListener(socketFile)
	} else {
		if cfg.ListenerType == ListenerUnix {
			os.Remove(cfg.ListenerAddr)
		}
		log.Printf("Creating %s listener on %s", cfg.ListenerType, cfg.ListenerAddr)
		return net.Listen(cfg.ListenerType.String(), cfg.ListenerAddr)
	}
}

func NewServer(cfg MeasurementConfig, ln net.Listener) *MeasurementServer {
	switch cfg.MeasurementSource {
	case SourceVolume:
		return NewSimulatedVolumeServer(ln)
	case SourceTemp :
		return NewSimulatedTemperatureServer(ln)
	default:
		log.Fatalf("Missing measurement source: %q", cfg.MeasurementSource)
	}
	return nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	cfg, err := NewConfig()
	if err != nil {
		log.Fatalf("Error marshalling config: %s", err)
	}

	ln, err := GetListener(cfg)
	if err != nil {
		log.Fatalf("Error listening on socket: %s", err)
	}

	log.Print("Now serving...")
	s := NewServer(cfg, ln)
	if err = s.Serve(ctx); err != nil {
		log.Fatalf("Failed to Serve: %s", err)
	}

}
