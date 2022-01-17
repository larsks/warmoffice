package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"warmoffice/sensors"
	"warmoffice/states"

	rfoutlet "github.com/martinohmann/rfoutlet/pkg/gpio"
	"github.com/mkideal/cli"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/warthog618/gpiod"
)

type (
	Options struct {
		cli.Helper

		InitialState    string  `cli:"s,state" dft:"OFF"`
		Chip            string  `cli:"chip" dft:"gpiochip0"`
		MotionPin       uint    `cli:"motion-pin" dft:"22"`
		TxPin           uint    `cli:"tx-pin" dft:"17"`
		OnCode          uint64  `cli:"on"`
		OffCode         uint64  `cli:"off"`
		TxProtocol      uint    `cli:"protocol" dft:"1"`
		PulseLength     uint    `cli:"l,pulse-length" dft:"200"`
		PrewarmTime     string  `cli:"prewarm" dft:"60m"`
		MinActivityTime string  `cli:"A,min-activity" dft:"10m"`
		MaxIdleTime     string  `cli:"I,max-idle" dft:"90m"`
		RecentTime      string  `cli:"R,recent" dft:"2m"`
		Verbose         uint    `cli:"v,verbose" dft:"1"`
		TempSensorID    string  `cli:"T,temp-sensor-id"`
		TargetTemp      float64 `cli:"t,temp" dft:"22"`
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
		Heater          *Thermostat
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

	tx, err := rfoutlet.NewTransmitter(app.Chip, int(args.TxPin),
		rfoutlet.TransmissionCount(3))
	if err != nil {
		panic(err)
	}

	tempsensor := sensors.NewDS1820(args.TempSensorID)
	rs := NewRemoteSwitch("heater", tx, args.OnCode, args.OffCode,
		WithProtocol(args.TxProtocol-1),
		WithPulseLength(args.PulseLength))

	app.Heater = NewThermostat(int(args.TargetTemp*1000), rs, tempsensor)

	log.Info().Msgf("using gpio chip %s", args.Chip)
	log.Info().Msgf("motion sensor using pin %d", args.MotionPin)
	log.Info().Msgf("tx using pin %d", args.TxPin)

	return app
}

func (app *Application) InitTimer() {
	app.Timer = time.Now()
}

func (app *Application) Close() {
	log.Info().Msgf("cleaning up")
	app.MotionSensor.Close()
	app.Heater.Close()
}

func (app *Application) NextState(state states.State) {
	app.InitTimer()
	prev_state := app.State
	app.State = state

	log.Info().Msgf("%s -> %s", prev_state, state)
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
	app.Heater.Start()

	for !app.QuitFlag {
		log.Debug().
			Str("state", app.State.String()).
			Dur("state_time", time.Since(app.Timer)).
			Dur("last_activity", time.Since(app.MotionSensor.LastActivity)).
			Msg("warmoffice status")

		start_state := app.State

		switch app.State {
		case states.INIT:
			app.NextState(app.InitialState)

		case states.OFF:
			if prev_state != states.OFF {
				app.Heater.HeatOn()
			}

		case states.PREWARM:
			if prev_state != states.PREWARM {
				log.Info().Msgf("PREWARM ends in %s", app.PrewarmTime)
				app.Heater.HeatOn()
			}

			if time.Since(app.Timer) > app.PrewarmTime {
				app.NextState(states.ACTIVE)
			}

		case states.IDLE:
			if prev_state != states.IDLE {
				app.Heater.HeatOff()
			}

			if app.MotionSensor.RecentActivity() {
				app.NextState(states.TRACKING)
			}

		case states.TRACKING:
			if prev_state != states.TRACKING {
				log.Info().Msgf("Tracking for %s", app.MinActivityTime)
				log.Info().Msgf("Recent activity threshold is %s", app.RecentTime)
				app.Heater.HeatOn()
			}

			if !app.MotionSensor.RecentActivity() {
				app.NextState(states.IDLE)
			}

			if time.Since(app.Timer) > app.MinActivityTime {
				app.NextState(states.ACTIVE)
			}

		case states.ACTIVE:
			if prev_state != states.ACTIVE {
				log.Info().Msgf("Max idle time is %s", app.MaxIdleTime)
				app.Heater.HeatOn()
			}

			if time.Since(app.MotionSensor.LastActivity) > app.MaxIdleTime {
				app.NextState(states.IDLE)
			}
		}

		prev_state = start_state
		time.Sleep(1000 * time.Millisecond)
	}

	app.Heater.Stop()
}

func main() {
	os.Exit(cli.Run(new(Options), func(ctx *cli.Context) error {
		args := ctx.Argv().(*Options)

		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

		var loglevel zerolog.Level
		switch args.Verbose {
		case 0:
			loglevel = zerolog.WarnLevel
		case 1:
			loglevel = zerolog.InfoLevel
		default:
			loglevel = zerolog.DebugLevel
		}
		zerolog.SetGlobalLevel(loglevel)

		app := NewApplication(args)
		defer app.Close()
		app.Loop()

		return nil
	}))
}
