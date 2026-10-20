package main

import (
	"testing"
)

func TestSimulatedVolumeService(t *testing.T) {
	s := NewSimulatedVolumeServer(nil)
	var results []Reading
	var high int
	var low int
	for range(100){
		result := s.service.GetReading()
		if result.Value() > 30 {
			high++
		} else {
			low ++
		}
		results = append(results, result)
	}
	
	if high == 0 || low == 0 {
		t.Errorf("Results aren't spread between high and low: high %d, low %d", high, low)
	}
}