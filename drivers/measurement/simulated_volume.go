package main

import (
	"math/rand/v2"
)

type SimulatedVolumeProbe struct {
	LastReading float64
}

func (p *SimulatedVolumeProbe) ReadProbe() float64 {
	var vol float64
	delta := rand.Float64()
	last := p.LastReading
	if last == 0 {
		vol = delta
	} else {
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
	p.LastReading = vol
	return vol
}
func (p *SimulatedVolumeProbe) GetMetricName() string {
	return "room_volume_decibels"
}
func (p *SimulatedVolumeProbe) GetMetricDesc() string {
	return "Sound volume of the room in decibels"
}
