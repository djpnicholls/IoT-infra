package main

import (
	"fmt"
	"net/http"
)

type Probe interface{
	ReadProbe() float64
	GetMetricName() string
	GetMetricDesc() string
}
type MetricsHandler struct {
	probes []Probe
}

func(m *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, p := range m.probes {
	reading := p.ReadProbe()
	metric := p.GetMetricName()
	desc := p.GetMetricDesc()
	fmt.Fprintf(w, "# TYPE %s gauge\n", metric)
	fmt.Fprintf(w, "# HELP %s %s\n", metric, desc)
	fmt.Fprintf(w, "%s %f\n", metric, reading)
	}
}

func NewMetricsHandler(p []Probe) *MetricsHandler {
	return &MetricsHandler{
		probes: p,
	}
}