package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

type MockProbe struct {
	value      float64
	metricName string
	metricDesc string
}

func (p *MockProbe) ReadProbe() float64 {
	return p.value
}
func (p *MockProbe) GetMetricName() string {
	return p.metricName
}
func (p *MockProbe) GetMetricDesc() string {
	return p.metricDesc
}

type MockProbe2 struct {
	value      float64
	metricName string
	metricDesc string
}

func (p *MockProbe2) ReadProbe() float64 {
	return p.value
}
func (p *MockProbe2) GetMetricName() string {
	return p.metricName
}
func (p *MockProbe2) GetMetricDesc() string {
	return p.metricDesc
}

func TestMetricsServeHTTP(t *testing.T) {
	t.Run("Single probe", func(t *testing.T) {
		const expected = `
		# HELP room_temperature_celsius Temperature of the room in degrees celsius
		# TYPE room_temperature_celsius gauge
	
		room_temperature_celsius 10.5
	`
		expectedReader := strings.NewReader(expected)

		probes := []Probe{&MockProbe{
			value:      10.5,
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

	t.Run("multiple probes", func(t *testing.T) {
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
			value:      10.5,
			metricName: "room_temperature_celsius",
			metricDesc: "Temperature of the room in degrees celsius",
		}
		probe2 := &MockProbe{
			value:      2.5,
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
	t.Run("multiple probe types", func(t *testing.T) {
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
			value:      10.5,
			metricName: "room_temperature_celsius",
			metricDesc: "Temperature of the room in degrees celsius",
		}
		probe2 := &MockProbe2{
			value:      2.5,
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
	t.Run("temperature probe", func(t *testing.T) {
		tempProbe := []Probe{&SimulatedTemperatureProbe{}}

		handler := NewMetricsHandler(tempProbe)
		_, err := scrapeMetricValue(t, handler, "room_temperature_celsius", -10, 30)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
	t.Run("volume probe", func(t *testing.T) {

		volProbe := []Probe{&SimulatedVolumeProbe{}}

		handler := NewMetricsHandler(volProbe)
		_, err := scrapeMetricValue(t, handler, "room_volume_decibels", -10, 40)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
	t.Run("multiple probes", func(t *testing.T) {
		probes := []Probe{
			&SimulatedVolumeProbe{},
			&SimulatedTemperatureProbe{},
		}

		handler := NewMetricsHandler(probes)
		_, err := scrapeMetricValue(t, handler, "room_temperature_celsius", -10, 30)
		if err != nil {
			t.Errorf("Unexpected error scraping temperature value: %v", err)
		}
		_, err = scrapeMetricValue(t, handler, "room_volume_decibels", -10, 40)
		if err != nil {
			t.Errorf("Unexpected error scraping volume value: %v", err)
		}
	})
	t.Run("multiple scrapes return different values", func(t *testing.T) {
		tempProbe := []Probe{&SimulatedTemperatureProbe{}}

		handler := NewMetricsHandler(tempProbe)
		value1, err := scrapeMetricValue(t, handler, "room_temperature_celsius", -10, 30)
		if err != nil {
			t.Errorf("Unexpected error scraping first time: %v", err)
		}
		if len(value1) != 1 {
			t.Fatalf("Unexpected number of values in first scrape: %d", len(value1))
		}
		value2, err := scrapeMetricValue(t, handler, "room_temperature_celsius", -10, 30)
		if err != nil {
			t.Errorf("Unexpected error scraping second time: %v", err)
		}
		if len(value2) != 1 {
			t.Fatalf("Unexpected number of values in second scrape: %d", len(value2))
		}
		if value1[0] == value2[0] {
			t.Errorf("Values didn't change. \nFirst: %f \nSecond: %f", value1[0], value2[0])
		}
	})
}

func scrapeMetricValue(t *testing.T, handler http.Handler, metricName string, lowerBound, upperBound float64) ([]float64, error) {
	t.Helper()
	var values []float64

	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL)
	if err != nil {
		err = fmt.Errorf("Error scraping metrics: %v", err)
		return values, err
	}
	defer resp.Body.Close()

	return parseMetricValues(t, resp.Body, metricName, lowerBound, upperBound)
}

func parseMetricValues(t *testing.T, body io.ReadCloser, metricName string, lowerBound, upperBound float64) ([]float64, error) {
	t.Helper()
	var values []float64

	parser := expfmt.NewTextParser(model.UTF8Validation)
	metricFamilies, err := parser.TextToMetricFamilies(body)
	if err != nil {
		err = fmt.Errorf("Error parsing metric format: %v", err)
		return values, err
	}
	family, exists := metricFamilies[metricName]
	if !exists {
		t.Fatalf("Missing required metric from scrape values")
	}
	for _, m := range family.Metric {
		value := m.Gauge.GetValue()
		if value < lowerBound || value > upperBound {
			err = fmt.Errorf("Metric out of range: %f", value)
			return []float64{}, err
		}
		values = append(values, value)

	}
	return values, nil
}
