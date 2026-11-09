package main

import (
	"math/rand/v2"
)

func SimulateInitialTempReading() float64 {
	rand_reading := rand.Float64()
	// We want the range -10 to 30
	return (rand_reading * 40) - 10
}

type SimulatedTemperatureProbe struct {
	LastReading float64
}

func (p *SimulatedTemperatureProbe) ReadProbe() float64 {
	var temp float64
	if p.LastReading == 0 {
		temp = SimulateInitialTempReading()
	} else {
		temp = p.LastReading + (rand.Float64() - 0.5)
	}
	p.LastReading = temp
	return temp
}
func (p *SimulatedTemperatureProbe) GetMetricName() string {
	return "room_temperature_celsius"
}
func (p *SimulatedTemperatureProbe) GetMetricDesc() string {
	return "Temperature of the room in degrees celsius"
}
