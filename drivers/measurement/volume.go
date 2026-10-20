package main

type VolumeReading struct {
	Time   int64   `json:"time"`
	Volume float64 `json:"volume"`
}

func (v VolumeReading) Timestamp() int64 {
	return v.Time
}
func (v VolumeReading) Value() float64 {
	return v.Volume
}
