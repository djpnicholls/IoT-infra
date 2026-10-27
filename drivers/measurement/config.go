package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type UnknownSourceError struct {
	Source string
}
func (e UnknownSourceError) Error() string {
	return fmt.Sprintf("Unrecognised measurement source: %q", e.Source)
}

type UnknownListenerTypeError struct {
	LType string
}
func (e UnknownListenerTypeError) Error() string {
	return fmt.Sprintf("Unrecognised listener type: %q", e.LType)
}

type MissingListenerAddressError struct {}
func (e MissingListenerAddressError) Error() string {
	return "Missing listener address. Set LISTEN_PID or MEASUREMENT_LISTENER_ADDRESS"
}

type MeasurementSource int
const (
	SourceVolume MeasurementSource = iota
	SourceTemp   MeasurementSource = iota
)
var MeasurementSourceName = map[MeasurementSource]string {
	SourceVolume: "volume",
	SourceTemp: "temp",
}
func (ms MeasurementSource) String() string {
	return MeasurementSourceName[ms]
}

type ListenerType int 
const (
	ListenerSystemd ListenerType = iota
	ListenerUnix    ListenerType = iota
	ListenerTcp     ListenerType = iota
)
var ListenerTypeName = map[ListenerType]string {
	ListenerSystemd: "systemd",
	ListenerUnix: "unix",
	ListenerTcp: "tcp",
}
func (lt ListenerType) String() string {
	return ListenerTypeName[lt]

}

type MeasurementConfig struct {
	ListenerType 	  ListenerType
	ListenerAddr 	  string
	MeasurementSource MeasurementSource
}

func NewConfig() (MeasurementConfig, error) {
	c := MeasurementConfig{}
	s, err := getMeasurementSource()
	if err != nil{
		return MeasurementConfig{}, err
	}
	c.MeasurementSource = s
	t, a, err := getListenerConfig()
	if err != nil {
		return MeasurementConfig{}, err
	}
	c.ListenerType = t
	c.ListenerAddr = a
	return c, nil
}

func getMeasurementSource() (MeasurementSource, error){
	measurementSource := os.Getenv("MEASUREMENT_SOURCE")
	switch strings.ToLower(measurementSource) {
	case "volume":
		return SourceVolume, nil
	case "temp":
		return SourceTemp, nil
	default:
		return 0, UnknownSourceError{Source: measurementSource}
	}
}

func getListenerConfig() (ListenerType, string, error) {
	var t ListenerType
	var a string

	if os.Getenv("LISTEN_PID") == strconv.Itoa(os.Getpid()){
		t = ListenerSystemd
		a = strings.Split(os.Getenv("LISTEN_FDNAMES"), ":")[0]
	} else {
		listenerType := os.Getenv("MEASUREMENT_LISTENER_TYPE")
		switch strings.ToLower(listenerType) {
		case "unix":
			t = ListenerUnix
		case "tcp":
			t = ListenerTcp
		default:
			return 0, "", UnknownListenerTypeError{LType: listenerType}
		}
		
		a = os.Getenv("MEASUREMENT_LISTENER_ADDRESS")
		if a == "" {
			return 0, "", MissingListenerAddressError{}
		}
	}
	return t, a, nil
}