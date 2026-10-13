package main

import (
	"context"
	"encoding/json"
	"math/rand/v2"
	"net"
	"os"
	"syscall"
	"time"
	"os/signal"
	"log"
)

type Reading struct {
	Time int64 `json:"time"`
	Temperature float64 `json:"temp"`
}

type TemperatureService interface {
	GetReading() Reading
}

type TemperatureServer struct {
	Listener net.Listener
	service TemperatureService
}

func (t *TemperatureServer) GetReading() Reading{
	return Reading{}
}

type SimulatedTemperatureService struct {
	RecentReadings []Reading
}

func NewSimulatedTemperatureServer(ln net.Listener) *TemperatureServer {
	return &TemperatureServer{
		Listener: ln,
		service: &SimulatedTemperatureService{
			RecentReadings: nil,
			},
		}
}

func(t *SimulatedTemperatureService) GetReading() Reading {
	var temp float64
	if len(t.RecentReadings) == 0 {
		temp = SimulateTempReading()
		reading := Reading{
			Time: time.Now().Unix(),
			Temperature: temp,
		}
		t.RecentReadings = append(t.RecentReadings, reading)
		return reading
	}
	if len(t.RecentReadings) > 5 {
		t.RecentReadings = t.RecentReadings[1:]
	}
	temp = t.RecentReadings[len(t.RecentReadings)-1].Temperature + (rand.Float64() - 0.5)
	reading := Reading{
		Time: time.Now().Unix(),
		Temperature: temp,
	}
	t.RecentReadings = append(t.RecentReadings, reading)
	return reading
}
func SimulateTempReading() float64 {
	rand_reading := rand.Float64()
	// We want the range -10 to 30
	return (rand_reading * 40) - 10
}

func (s *TemperatureServer) Serve(ctx context.Context) error {
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
  
func (s *TemperatureServer) handleConn(ctx context.Context, c net.Conn, interval time.Duration) {
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
	socketFile := "/tmp/temp_s.sock"

	os.Remove(socketFile)
	ln, err := net.Listen("unix", socketFile); 
	if err != nil{
		log.Fatalf("Could not listen on socket file %s: %s", socketFile, err)
	}
	
	s := NewSimulatedTemperatureServer(ln)
	err = s.Serve(ctx)
	if err != nil{
		log.Fatalf("Failed to Serve: %s", err)
	}

}
