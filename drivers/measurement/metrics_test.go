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
	const expected = `
		# HELP room_temperature_celsius Temperature of the room in degrees celsius
		# TYPE room_temperature_celsius gauge
	
		room_temperature_celsius 10.5
	`
	expectedReader := strings.NewReader(expected)

	probe := &MockProbe{
		value: 10.5,
		metricName: "room_temperature_celsius",
		metricDesc: "Temperature of the room in degrees celsius",
	}

	handler := NewMetricsHandler(probe)
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	if err := testutil.ScrapeAndCompare(ts.URL, expectedReader); err != nil {
		t.Errorf("unexpected scraping result: \n%v", err)
	}
}

func TestMetricsServerIntegration(t *testing.T) {
	tempProbe := &SimulatedTemperatureService{}

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