package sensors

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

func (ds *DS1820) reader() {
	var lastread time.Time

	temp_path := filepath.Join(ds.dev_path, "temperature")

	log.Debug().Str("id", ds.Id).Msg("entering read loop")

	for {
		//nolint
		select {
		case <-ds.quitflag:
			break
		default:
			if time.Since(lastread) > (1 * time.Minute) {
				val, err := ioutil.ReadFile(temp_path)
				if err != nil {
					panic(err)
				}

				temp, err := strconv.Atoi(strings.TrimSpace(string(val)))
				if err != nil {
					panic(err)
				}
				log.Debug().Str("id", ds.Id).Int("temp", temp).Msg("read temperature")
				ds.Temp = temp
				lastread = time.Now()
			}

			time.Sleep(1 * time.Second)
		}
	}

	//nolint
	log.Debug().Str("id", ds.Id).Msg("exit read loop")
}

func (ds *DS1820) Start() {
	ds.quitflag = make(chan struct{})
	go ds.reader()
}

func (ds *DS1820) Stop() {
	close(ds.quitflag)
}
