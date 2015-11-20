package main

import (
	"bytes"
	"fmt"
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
	fmt.Println("mtrk 1")
	midi.ReadMTrk()
	for midi.HasNextEvent() {
		if err := midi.ReadNextEvent(); err != nil {
			panic(err)
		}
	}
	fmt.Println("mtrk 2")
	midi.ReadMTrk()
	for midi.HasNextEvent() {
		if err := midi.ReadNextEvent(); err != nil {
			panic(err)
		}
	}
	fmt.Println("mtrk 3")
	midi.ReadMTrk()
	for midi.HasNextEvent() {
		if err := midi.ReadNextEvent(); err != nil {
			panic(err)
		}
	}
	fmt.Println("mtrk 4")
	midi.ReadMTrk()
	for midi.HasNextEvent() {
		if err := midi.ReadNextEvent(); err != nil {
			panic(err)
		}
	}
	fmt.Println("mtrk 5")
	midi.ReadMTrk()
	for midi.HasNextEvent() {
		if err := midi.ReadNextEvent(); err != nil {
			panic(err)
		}
	}
	fmt.Println("mtrk 6")
	midi.ReadMTrk()
	for midi.HasNextEvent() {
		if err := midi.ReadNextEvent(); err != nil {
			panic(err)
		}
	}
}
