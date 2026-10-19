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
	service MeasurementService
}

func (s *MeasurementServer) Serve(ctx context.Context) error {
	go func(){
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

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var ln net.Listener
	var err error
	
	if os.Getenv("LISTEN_PID") == strconv.Itoa(os.Getpid()){
		log.Println("Running under systemd")
		socketFile := os.NewFile(3, "systemd-socket")
		ln, err = net.FileListener(socketFile)
	} else {
		socketFile := "/tmp/temp_s.sock"
		os.Remove(socketFile)
		log.Printf("Opening socket file %s", socketFile)
		ln, err = net.Listen("unix", socketFile); 
		if err != nil{
			log.Fatalf("Could not listen on socket file %s: %s", socketFile, err)
		}
		if err = os.Chmod(socketFile, 0660); err != nil{
			log.Fatalf("Could not make socket group-writeable: %s", err)
		}
	}
	log.Print("Now serving...")
	s := NewSimulatedTemperatureServer(ln)
	if err = s.Serve(ctx); err != nil{
		log.Fatalf("Failed to Serve: %s", err)
	}

}
