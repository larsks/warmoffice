GOARCH=arm
GOARM=7
GOOS=linux

SRCS = $(shell find * -type f -name '*.go' -not -path "./vendor/*")

%.svg: %.dot
	dot -T svg -o $@ $<

all: warmoffice states.svg

warmoffice: $(SRCS)
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) go build -o $@

clean:
	rm -f warmoffice

install: warmoffice
	scp warmoffice pi@raspberrypi.local:
