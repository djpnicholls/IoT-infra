package main

import(
	"time"
	"math/rand/v2"
	"net"
)

type SimulatedTemperatureService struct {
	RecentReadings []Reading
}

func NewSimulatedTemperatureServer(ln net.Listener) *MeasurementServer {
	return &MeasurementServer{
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
		reading := TemperatureReading{
			Time: time.Now().Unix(),
			Temperature: temp,
		}
		t.RecentReadings = append(t.RecentReadings, reading)
		return reading
	}
	if len(t.RecentReadings) > 5 {
		t.RecentReadings = t.RecentReadings[1:]
	}
	temp = t.RecentReadings[len(t.RecentReadings)-1].Value() + (rand.Float64() - 0.5)
	reading := TemperatureReading{
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
