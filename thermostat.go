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
		MaxDelta:     1000,
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
		delta := therm.SetTemp - temp

		clog := log.With().
			Int("have", temp).
			Int("want", therm.SetTemp).
			Int("delta", delta).
			Int("maxdelta", therm.MaxDelta).
			Logger()

		clog.Debug().
			Bool("active", therm.Heat).
			Bool("switch", therm.RemoteSwitch.On).
			Msg("thermostat running")

		if therm.Heat {
			if delta > therm.MaxDelta {
				if !therm.RemoteSwitch.On {
					clog.Info().Msg("heater on")

					therm.RemoteSwitch.TurnOn()
				}
			} else {
				if therm.RemoteSwitch.On {
					clog.Info().Msg("heater off")
					therm.RemoteSwitch.TurnOff()
				}
			}
		} else {
			if therm.RemoteSwitch.On {
				clog.Info().Msg("thermostat disabled")
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
	log.Info().Msg("thermostat active")
	therm.mu.Lock()
	therm.Heat = true
	therm.mu.Unlock()
}

func (therm *Thermostat) HeatOff() {
	log.Info().Msg("thermostat inactive")
	therm.mu.Lock()
	therm.Heat = false
	therm.mu.Unlock()
}

func (therm *Thermostat) Close() {
	therm.RemoteSwitch.Close()
}
