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
		service: &SimulatedTemperatureService{},
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

func (t *SimulatedTemperatureService) ReadProbe() float64{
	return t.GetReading().Value()
}
func (t *SimulatedTemperatureService) GetMetricName() string {
	return "room_temperature_celsius"
}
func (t *SimulatedTemperatureService) GetMetricDesc() string {
	return "Temperature of the room in degrees celsius"
}