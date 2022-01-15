package states

import (
	"fmt"
	"strings"
)

type State int

const (
	INIT     State = iota
	OFF            // heater is off, not watching for activity
	IDLE           // heater is off, watching for activity
	TRACKING       // heater is off, motioned detected, watching for activity
	PREWARM        // heater is on, not watching for activity
	ACTIVE         // heater is on, watching for activity
	LOCKED         // heater is on, not watching for activity
)

func FromString(s string) State {
	switch strings.ToUpper(s) {
	case "INIT":
		return INIT
	case "OFF":
		return OFF
	case "IDLE":
		return IDLE
	case "TRACKING":
		return TRACKING
	case "PREWARM":
		return PREWARM
	case "ACTIVE":
		return ACTIVE
	case "LOCKED":
		return LOCKED
	}

	panic(fmt.Errorf("unknown state: %s", s))
}

func (s State) String() string {
	switch s {
	case INIT:
		return "INIT"
	case OFF:
		return "OFF"
	case PREWARM:
		return "PREWARM"
	case IDLE:
		return "IDLE"
	case TRACKING:
		return "TRACKING"
	case ACTIVE:
		return "ACTIVE"
	case LOCKED:
		return "LOCKED"
	}

	return "(unknown)"
}
