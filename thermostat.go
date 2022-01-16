package main

import (
	"sync"
	"time"
	"warmoffice/sensors"

	"github.com/rs/zerolog/log"
)

type (
	Thermostat struct {
		SetTemp      int
		MaxDelta     int
		Heat         bool
		RemoteSwitch *RemoteSwitch
		TempSensor   *sensors.DS1820

		quit chan struct{}
		mu   sync.Mutex
	}

	ThermostatOption func(*Thermostat)
)

func NewThermostat(target int, rs *RemoteSwitch, temp *sensors.DS1820, options ...ThermostatOption) *Thermostat {
	therm := Thermostat{
		SetTemp:      target,
		MaxDelta:     2000,
		Heat:         false,
		RemoteSwitch: rs,
		TempSensor:   temp,
	}

	for _, option := range options {
		option(&therm)
	}

	return &therm
}

func WithMaxDelta(maxdelta int) ThermostatOption {
	return func(therm *Thermostat) {
		therm.MaxDelta = maxdelta
	}
}

func (therm *Thermostat) Loop() {
	therm.TempSensor.Start()
	defer func() {
		therm.TempSensor.Stop()
		therm.RemoteSwitch.TurnOff()
	}()

	for loop := true; loop; {
		temp := therm.TempSensor.Read()

		log.Debug().
			Int("have", temp).
			Int("want", therm.SetTemp).
			Bool("active", therm.Heat).
			Bool("switch", therm.RemoteSwitch.On).
			Msg("thermostat active")

		if therm.Heat {
			if (therm.SetTemp - temp) > therm.MaxDelta {
				if !therm.RemoteSwitch.On {
					therm.RemoteSwitch.TurnOn()
				}
			} else {
				if therm.RemoteSwitch.On {
					therm.RemoteSwitch.TurnOff()
				}
			}
		} else {
			if therm.RemoteSwitch.On {
				therm.RemoteSwitch.TurnOff()
			}
		}

		select {
		case <-therm.quit:
			loop = false
		case <-time.After(1 * time.Minute):
		}
	}
}

func (therm *Thermostat) Start() {
	log.Debug().Msg("starting thermostat")
	therm.quit = make(chan struct{})
	go therm.Loop()
}

func (therm *Thermostat) Stop() {
	log.Debug().Msg("stopping thermostat")
	close(therm.quit)
}

func (therm *Thermostat) HeatOn() {
	log.Info().Msg("turn heat on")
	therm.mu.Lock()
	therm.Heat = true
	therm.mu.Unlock()
}

func (therm *Thermostat) HeatOff() {
	log.Info().Msg("turn heat off")
	therm.mu.Lock()
	therm.Heat = false
	therm.mu.Unlock()
}

func (therm *Thermostat) Close() {
	therm.RemoteSwitch.Close()
}