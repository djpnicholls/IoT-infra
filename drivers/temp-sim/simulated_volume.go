package main

import (
	"log"
	"math/rand/v2"
	"net"
	"time"
)

type SimulatedVolumeService struct {
	LastReading Reading
}

func NewSimulatedVolumeServer(ln net.Listener) *MeasurementServer {
	return &MeasurementServer{
		Listener: ln,
		service:  &SimulatedVolumeService{},
	}
}

func (v *SimulatedVolumeService) GetReading() Reading {
	var vol float64
	delta := rand.Float64()
	log.Printf("delta: %f", delta)
	if v.LastReading == nil {
		vol = delta
		log.Printf("No last reading.  Setting volume to delta: %f", vol)
	} else {
		last := v.LastReading.Value()
		if last > 30 {
			log.Printf("Last reading greater than 30: %f", last)
			if delta > 0.3 {
				log.Printf("30%% chance of lowering volume: %f", delta)
				delta -= 30
			}
		} else {
			log.Printf("Last reading less than 30: %f", last)
			if delta < 0.1 {
				log.Printf("10%% chance of increasing volume: %f", delta)
				delta += 30
			}
		}
		vol = last + delta
		log.Printf("Altering volume by %f:  %f", delta, vol)
	}
	reading := VolumeReading{
		Time: 		time.Now().Unix(),
		Volume:		vol,
	}
	v.LastReading = reading
	return reading
}
