package main

import (
	"math/rand/v2"
	"net"
	"time"
)

type SimulatedTemperatureService struct {
	LastReading Reading
}

func NewSimulatedTemperatureServer(ln net.Listener) *MeasurementServer {
	return &MeasurementServer{
		Listener: ln,
		service:  &SimulatedTemperatureService{},
	}
}

func (t *SimulatedTemperatureService) GetReading() Reading {
	var temp float64
	if t.LastReading == nil {
		temp = SimulateInitialTempReading()
	} else {
		temp = t.LastReading.Value() + (rand.Float64() - 0.5)
	}
	reading := TemperatureReading{
		Time:        time.Now().Unix(),
		Temperature: temp,
	}
	t.LastReading = reading
	return reading
}
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
