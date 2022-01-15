package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"warmoffice/sensors"
	"warmoffice/states"

	"github.com/martinohmann/rfoutlet/pkg/gpio"
	rfoutlet "github.com/martinohmann/rfoutlet/pkg/gpio"
	"github.com/mkideal/cli"
	"github.com/warthog618/gpiod"
)

type (
	Options struct {
		cli.Helper

		InitialState string `cli:"s,state" dft:"OFF"`
		Chip         string `cli:"c,chip" dft:"gpiochip0"`
		MotionPin    int    `cli:"m,motion-pin" dft:"22"`
		TxPin        int    `cli:"t,tx-pin" dft:"17"`
		OnCode       uint64 `cli:"on"`
		OffCode      uint64 `cli:"off"`
		TxProtocol   int    `cli:"protocol" dft:"1"`
		PulseLength  uint   `cli:"l,pulse-length" dft:"200"`

		PrewarmTime     string `cli:"prewarm" dft:"60m"`
		MinActivityTime string `cli:"A,min-activity" dft:"10m"`
		MaxIdleTime     string `cli:"I,max-idle" dft:"90m"`
		RecentTime      string `cli:"R,recent" dft:"2m"`
	}

	Application struct {
		Options         *Options
		QuitFlag        bool
		InitialState    states.State
		State           states.State
		PrewarmTime     time.Duration
		MinActivityTime time.Duration
		MaxIdleTime     time.Duration
		RecentTime      time.Duration
		Chip            *gpiod.Chip
		MotionSensor    *sensors.MotionSensor
		Timer           time.Time
		Transmitter     *rfoutlet.Transmitter
	}
)

func NewApplication(args *Options) *Application {
	app := &Application{
		Options: args,
		State:   states.INIT,
	}

	app.InitialState = states.FromString(args.InitialState)

	{
		var err error
		app.PrewarmTime, err = time.ParseDuration(args.PrewarmTime)
		if err != nil {
			panic(err)
		}

		app.MinActivityTime, err = time.ParseDuration(args.MinActivityTime)
		if err != nil {
			panic(err)
		}

		app.MaxIdleTime, err = time.ParseDuration(args.MaxIdleTime)
		if err != nil {
			panic(err)
		}

		app.RecentTime, err = time.ParseDuration(args.RecentTime)
		if err != nil {
			panic(err)
		}
	}
	app.Timer = time.Now()

	chip, err := gpiod.NewChip(args.Chip, gpiod.WithConsumer("warmoffice"))
	if err != nil {
		panic(err)
	}
	app.Chip = chip

	app.MotionSensor = sensors.NewMotionSensor(app.Chip, args.MotionPin,
		sensors.WithRecentActivityThreshold(app.RecentTime))
	app.MotionSensor.InitLastActivity()

	tx, err := rfoutlet.NewTransmitter(app.Chip, args.TxPin,
		rfoutlet.TransmissionCount(3))
	if err != nil {
		panic(err)
	}
	app.Transmitter = tx

	log.Printf("using gpio chip %s", args.Chip)
	log.Printf("motion sensor using pin %d", args.MotionPin)
	log.Printf("tx using pin %d", args.TxPin)
	log.Printf("tx protocol %d, pulse length %d", args.TxProtocol, args.PulseLength)

	return app
}

func (app *Application) TurnSwitchOn() {
	log.Printf("turn switch on (code %d)", app.Options.OnCode)

	res := app.Transmitter.Transmit(app.Options.OnCode,
		gpio.DefaultProtocols[app.Options.TxProtocol-1],
		app.Options.PulseLength)
	<-res
}

func (app *Application) TurnSwitchOff() {
	log.Printf("turn switch off (code %d)", app.Options.OffCode)

	res := app.Transmitter.Transmit(app.Options.OffCode,
		gpio.DefaultProtocols[app.Options.TxProtocol-1],
		app.Options.PulseLength)
	<-res
}

func (app *Application) InitTimer() {
	app.Timer = time.Now()
}

func (app *Application) Close() {
	log.Printf("cleaning up")
	app.MotionSensor.Close()
	app.Transmitter.Close()
}

func (app *Application) NextState(state states.State) {
	app.InitTimer()
	prev_state := app.State
	app.State = state

	log.Printf("%s -> %s", prev_state, state)
}

func (app *Application) Quit() {
	app.QuitFlag = true
}

func (app *Application) WaitForSignals() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		app.QuitFlag = true
	}()
}

func (app *Application) Loop() {
	var prev_state states.State

	app.WaitForSignals()

	for !app.QuitFlag {
		log.Printf("state = %s, delta = %s, lastactive = %s",
			app.State,
			time.Since(app.Timer),
			time.Since(app.MotionSensor.LastActivity))

		start_state := app.State

		switch app.State {
		case states.INIT:
			app.NextState(app.InitialState)

		case states.OFF:
			if prev_state != states.OFF {
				app.TurnSwitchOff()
			}

		case states.PREWARM:
			if prev_state != states.PREWARM {
				log.Printf("PREWARM ends in %s", app.PrewarmTime)
				app.TurnSwitchOn()
			}

			if time.Since(app.Timer) > app.PrewarmTime {
				app.NextState(states.ACTIVE)
			}

		case states.IDLE:
			if prev_state != states.IDLE {
				app.TurnSwitchOff()
			}

			if app.MotionSensor.RecentActivity() {
				app.NextState(states.TRACKING)
			}

		case states.TRACKING:
			if prev_state != states.TRACKING {
				log.Printf("Tracking for %s", app.MinActivityTime)
				log.Printf("Recent activity threshold is %s", app.RecentTime)
				app.TurnSwitchOn()
			}

			if !app.MotionSensor.RecentActivity() {
				app.NextState(states.IDLE)
			}

			if time.Since(app.Timer) > app.MinActivityTime {
				app.NextState(states.ACTIVE)
			}

		case states.ACTIVE:
			if prev_state != states.ACTIVE {
				log.Printf("Max idle time is %s", app.MaxIdleTime)
				app.TurnSwitchOn()
			}

			if time.Since(app.MotionSensor.LastActivity) > app.MaxIdleTime {
				app.NextState(states.IDLE)
			}
		}

		prev_state = start_state
		time.Sleep(1 * time.Second)
	}

	app.TurnSwitchOff()
}

func main() {
	os.Exit(cli.Run(new(Options), func(ctx *cli.Context) error {
		args := ctx.Argv().(*Options)
		app := NewApplication(args)
		defer app.Close()
		app.Loop()

		return nil
	}))
}