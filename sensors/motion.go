package sensors

import (
	"log"
	"time"

	"github.com/warthog618/gpiod"
)

type (
	MotionSensor struct {
		Chip *gpiod.Chip
		Line *gpiod.Line

		RecentActivityThreshold time.Duration
		LastActivity            time.Time
	}

	MotionSensorOption func(*MotionSensor)
)

func WithRecentActivityThreshold(d time.Duration) MotionSensorOption {
	return func(sensor *MotionSensor) {
		sensor.RecentActivityThreshold = d
	}
}

func NewMotionSensor(chip *gpiod.Chip, pin int, options ...MotionSensorOption) *MotionSensor {
	sensor := MotionSensor{
		Chip:                    chip,
		RecentActivityThreshold: 2 * time.Minute,
	}

	line, err := chip.RequestLine(pin, gpiod.AsInput,
		gpiod.WithEventHandler(sensor.UpdateLastActivity), gpiod.WithBothEdges)
	if err != nil {
		panic(err)
	}
	sensor.Line = line

	for _, option := range options {
		option(&sensor)
	}

	return &sensor
}

func (sensor *MotionSensor) InitLastActivity() {
	sensor.LastActivity = time.Now()
}

func (sensor *MotionSensor) UpdateLastActivity(event gpiod.LineEvent) {
	log.Printf("activity detected")
	sensor.LastActivity = time.Now()
}

func (sensor *MotionSensor) RecentActivity() bool {
	delta := time.Since(sensor.LastActivity)
	return delta < sensor.RecentActivityThreshold
}

func (sensor *MotionSensor) Close() {
	sensor.Line.Close()
	sensor.Chip.Close()
}
