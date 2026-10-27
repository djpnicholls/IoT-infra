package main

import (
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
	if v.LastReading == nil {
		vol = delta
	} else {
		last := v.LastReading.Value()
		if last > 30 {
			if delta > 0.3 {
				delta -= 30
			}
		} else {
			if delta < 0.1 {
				delta += 30
			}
		}
		vol = last + delta
	}
	reading := VolumeReading{
		Time: 		time.Now().Unix(),
		Volume:		vol,
	}
	v.LastReading = reading
	return reading
}
