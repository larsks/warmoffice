package main

import (
	"github.com/martinohmann/rfoutlet/pkg/gpio"
	rfoutlet "github.com/martinohmann/rfoutlet/pkg/gpio"
	"github.com/rs/zerolog/log"
)

type (
	RemoteSwitch struct {
		Name        string
		Transmitter *rfoutlet.Transmitter
		Protocol    rfoutlet.Protocol
		PulseLength uint
		OnCode      uint64
		OffCode     uint64
		On          bool
	}

	RemoteSwitchOption func(*RemoteSwitch)
)

func NewRemoteSwitch(name string, tx *rfoutlet.Transmitter, oncode uint64, offcode uint64, options ...RemoteSwitchOption) *RemoteSwitch {
	rs := RemoteSwitch{
		Name:        name,
		Transmitter: tx,
		Protocol:    gpio.DefaultProtocols[0],
		PulseLength: 200,
		OnCode:      oncode,
		OffCode:     offcode,
		On:          false,
	}

	for _, option := range options {
		option(&rs)
	}

	return &rs
}

func WithProtocol(protocol uint) RemoteSwitchOption {
	return func(rs *RemoteSwitch) {
		rs.Protocol = gpio.DefaultProtocols[protocol]
	}
}

func WithPulseLength(length uint) RemoteSwitchOption {
	return func(rs *RemoteSwitch) {
		rs.PulseLength = length
	}
}

func (rs *RemoteSwitch) TurnOn() {
	log.Info().
		Str("switch", rs.Name).
		Int("code", int(rs.OnCode)).
		Msg("turning on")
	res := rs.Transmitter.Transmit(
		rs.OnCode,
		rs.Protocol,
		rs.PulseLength)
	<-res
	rs.On = true
}

func (rs *RemoteSwitch) TurnOff() {
	log.Info().
		Str("switch", rs.Name).
		Int("code", int(rs.OffCode)).
		Msg("turning off")
	res := rs.Transmitter.Transmit(
		rs.OffCode,
		rs.Protocol,
		rs.PulseLength)
	<-res
	rs.On = false
}

func (rs *RemoteSwitch) Close() {
	rs.TurnOff()
	rs.Transmitter.Close()
}
