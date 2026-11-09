package main

import (
	"testing"
)

func TestSimulatedVolumeService(t *testing.T) {
	s := SimulatedVolumeProbe{}
	var results []float64
	var high int
	var low int
	for range 100 {
		result := s.ReadProbe()
		if result > 30 {
			high++
		} else {
			low++
		}
		results = append(results, result)
	}

	if high == 0 || low == 0 {
		t.Errorf("Results aren't spread between high and low: high %d, low %d, \nresults: %v", high, low, results)
	}
}
