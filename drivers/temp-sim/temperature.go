package main



type TemperatureReading struct {
	Time int64 `json:"time"`
	Temperature float64 `json:"temp"`
}
func (t TemperatureReading) Timestamp() int64 {
	return t.Time
}
func (t TemperatureReading) Value() float64 {
	return t.Temperature
}