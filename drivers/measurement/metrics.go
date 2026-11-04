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
	probe Probe
}

func(m *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reading := m.probe.ReadProbe()
	metric := m.probe.GetMetricName()
	desc := m.probe.GetMetricDesc()
	fmt.Fprintf(w, "# TYPE %s gauge\n", metric)
	fmt.Fprintf(w, "# HELP %s %s\n", metric, desc)
	fmt.Fprintf(w, "%s %f\n", metric, reading)
}

func NewMetricsHandler(p Probe) *MetricsHandler {
	return &MetricsHandler{
		probe: p,
	}
}