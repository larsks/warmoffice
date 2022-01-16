package sensors

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const w1_device_path string = "/sys/bus/w1/devices/"

type (
	DS1820 struct {
		Id   string
		Temp int

		dev_path string
		quitflag chan struct{}
		mu       sync.Mutex
	}
)

func NewDS1820(id string) *DS1820 {
	dev_path := filepath.Join(w1_device_path, id)

	ds := DS1820{
		Id:       id,
		dev_path: dev_path,
	}

	fileinfo, err := os.Stat(dev_path)
	if err != nil {
		panic(err)
	}

	if !fileinfo.IsDir() {
		panic(fmt.Errorf("device %s does not exist", id))
	}

	return &ds
}

func (ds *DS1820) Loop() {
	var lastread time.Time

	temp_path := filepath.Join(ds.dev_path, "temperature")

	log.Debug().Str("id", ds.Id).Msg("entering read loop")

	for loop := true; loop; {
		select {
		case <-ds.quitflag:
			loop = false
		default:
			if time.Since(lastread) > (1 * time.Minute) {
				val, err := ioutil.ReadFile(temp_path)
				if err != nil {
					panic(err)
				}

				temp, err := strconv.Atoi(strings.TrimSpace(string(val)))
				if err != nil {
					log.Error().Str("id", ds.Id).Msg("failed to read temperature")
					continue
				}
				log.Debug().Str("id", ds.Id).Int("temp", temp).Msg("read temperature")
				ds.mu.Lock()
				ds.Temp = temp
				ds.mu.Unlock()
				lastread = time.Now()
			}

			time.Sleep(1 * time.Second)
		}
	}

	log.Debug().Str("id", ds.Id).Msg("exit read loop")
}

func (ds *DS1820) Start() {
	log.Debug().Str("id", ds.Id).Msg("starting")
	ds.quitflag = make(chan struct{})
	go ds.Loop()
}

func (ds *DS1820) Stop() {
	log.Debug().Str("id", ds.Id).Msg("stopping")
	close(ds.quitflag)
}

func (ds *DS1820) Read() int {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return ds.Temp
}
