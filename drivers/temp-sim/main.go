package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"os"
	"syscall"
	"time"
	"os/signal"
	"log"
)

func SimulateTempReading() float64 {
	rand_reading := rand.Float64()
	// We want the range -10 to 30
	return (rand_reading * 40) - 10
}

func CreatePayload(instant time.Time, reading float64) string {
	return fmt.Sprintf("{time: %d, temp: %.2f}\n", instant.Unix(), reading)
}

func Serve(ctx context.Context, ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			continue
		}
		go handleConn(ctx, conn, time.Second*5)

	}
}
  
func handleConn(ctx context.Context, c net.Conn, interval time.Duration) {
	defer c.Close()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reading := SimulateTempReading()
			payload := CreatePayload(time.Now(), reading)
			c.Write([]byte(payload))
		}
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	socketFile := "/var/run/temp_s.sock"

	os.Remove(socketFile)
	ln, err := net.Listen("unix", socketFile); 
	if err != nil{
		log.Fatalf("Could not listen on socket file %s: %s", socketFile, err)
	}
	
	err = Serve(ctx, ln)
	if err != nil{
		log.Fatalf("Failed to Serve: %s", err)
	}


}