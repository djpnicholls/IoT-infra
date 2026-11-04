package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

type MockProbe struct{
	value float64
	metricName string
	metricDesc string
}
func (p *MockProbe) ReadProbe() float64 {
	return p.value
}
func (p *MockProbe) GetMetricName() string {
	return p.metricName
}
func (p *MockProbe) GetMetricDesc() string{
	return p.metricDesc
}

func TestMetricsServeHTTP(t *testing.T) {
	t.Run("Single probe", func(t *testing.T){
	const expected = `
		# HELP room_temperature_celsius Temperature of the room in degrees celsius
		# TYPE room_temperature_celsius gauge
	
		room_temperature_celsius 10.5
	`
	expectedReader := strings.NewReader(expected)

	probes := []Probe{&MockProbe{
		value: 10.5,
		metricName: "room_temperature_celsius",
		metricDesc: "Temperature of the room in degrees celsius",
	}}

	handler := NewMetricsHandler(probes)
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	if err := testutil.ScrapeAndCompare(ts.URL, expectedReader); err != nil {
		t.Errorf("unexpected scraping result: \n%v", err)
	}
})

t.Run("multiple probes", func(t *testing.T){
	const expected = `
		# HELP room_temperature_celsius Temperature of the room in degrees celsius
		# TYPE room_temperature_celsius gauge
	
		room_temperature_celsius 10.5
		
		# HELP room_volume_decibels Sound volume of the room in decibels
		# TYPE room_volume_decibels gauge
		room_volume_decibels 2.5
	`
	expectedReader := strings.NewReader(expected)

	probe1 := &MockProbe{
		value: 10.5,
		metricName: "room_temperature_celsius",
		metricDesc: "Temperature of the room in degrees celsius",
	}
	probe2 := &MockProbe{
		value: 2.5,
		metricName: "room_volume_decibels",
		metricDesc: "Sound volume of the room in decibels",
	}
	probes := []Probe{probe1, probe2}

	handler := NewMetricsHandler(probes)
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	if err := testutil.ScrapeAndCompare(ts.URL, expectedReader); err != nil {
		t.Errorf("unexpected scraping result: \n%v", err)
	}

})
}

func TestMetricsServerIntegration(t *testing.T) {
	tempProbe := []Probe{&SimulatedTemperatureService{}}

	handler := NewMetricsHandler(tempProbe)
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Errorf("Error scraping metrics: %v", err)
	}
	defer resp.Body.Close()

	parser := expfmt.NewTextParser(model.UTF8Validation)
	metricFamilies, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		t.Errorf("Error parsing metric format: %v", err)
	}
	family, exists := metricFamilies["room_temperature_celsius"]
	if !exists {
		t.Errorf("Missing required metric from scrape values")
	}
	for _, m := range family.Metric {
		value := m.Gauge.GetValue()
		if value < -10 || value > 30 {
			t.Errorf("Metric out of range: %f", value)
		}

	}

}