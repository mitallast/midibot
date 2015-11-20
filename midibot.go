package main

import (
	"bytes"
	"github.com/mitallast/midibot/midi"
	"io/ioutil"
)

func main() {
	data, err := ioutil.ReadFile("smoke_on_the_water.mid")
	if err != nil {
		panic(err)
	}

	b := bytes.NewBuffer(data)
	midi := midi.NewMidi(b)
	midi.ReadMThd()

	for midi.HasNextMTrk() {
		if err := midi.ReadNextMTrk(); err != nil {
			panic(err)
		}
		for midi.HasNextEvent() {
			if err := midi.ReadNextEvent(); err != nil {
				panic(err)
			}
		}
	}
}
