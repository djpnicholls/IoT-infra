package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

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

func ConfigureListener(cfg *MeasurementConfig) error {
	ln, err := GetListener(*cfg)
	if err != nil {
		err = fmt.Errorf("Error getting listener: %w", err)
		return err
	}
	cfg.ln = ln
	return nil
}

func RunMetricsServer(ctx context.Context, cfg *MeasurementConfig) error {

	handler := NewMetricsHandler([]Probe{
		&SimulatedTemperatureProbe{},
		&SimulatedVolumeProbe{},
	})
	server := &http.Server{
		Handler: handler,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("Listening on %s\n", cfg.ln.Addr().String())
		if err := server.Serve(cfg.ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()
	select {
	case <-ctx.Done():
	case err := <-errCh:
		return err
	}
	if err := server.Shutdown(context.Background()); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return <-errCh
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	cfg, err := NewConfig()
	if err != nil {
		log.Fatalf("Error marshalling config: %s", err)
	}

	if err = ConfigureListener(&cfg); err != nil {
		log.Fatalf("Error listening on socket: %s", err)
	}

	log.Print("Now serving...")
	if err = RunMetricsServer(ctx, &cfg); err != nil {
		log.Fatalf("Failed to Serve: %s", err)
	}

}
