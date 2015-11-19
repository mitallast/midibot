package midi

import (
	"bytes"
	"errors"
	"encoding/binary"
	"fmt"
)

var (
	errEOF = errors.New("EOF")
	errNotImplemented = errors.New("Not implemented")
	errInvalidHeader = errors.New("Invalid header")
	errInvalidCommandCode = errors.New("Invalid command code")
	errInvalidMetaEventType = errors.New("Invalid meta event type")
	errInvalidTempoLength = errors.New("Invalid tempo length")
	errInvalidTimeSignatureLength = errors.New("Invalid time signature length")
	errInvalidKeySignatureLen = errors.New("Invalid key signature length")
	errInvalidTrackEndLength = errors.New("Invalid track end length")
	errInvalidPitchWheelByte = errors.New("Invalid pitch wheel byte")
	errInvalidPatch = errors.New("Invalid patch")
	errInvalidAfterTouchPressure = errors.New("Invalid after touch pressure")
	errInvalidSmpteOffset = errors.New("Invalid SMPTE Offset")
	errVarInt32Overflow = errors.New("binary: varint overflows a 64-bit integer")
)

const (
	CommandCodeNoteOff = 0x80
	CommandCodeNoteOn = 0x90
	CommandCodeKeyAfterTouch = 0xA0
	CommandCodeControlChange = 0xB0
	CommandCodePatchChange = 0xC0
	CommandCodeChannelAfterTouch = 0xD0
	CommandCodePitchWheelChange = 0xE0
	CommandCodeSysex = 0xF0
	CommandCodeEox = 0xF7
	CommandCodeTimingClock = 0xF8
	CommandCodeStartSequence = 0xFA
	CommandCodeContinueSequence = 0xFB
	CommandCodeStopSequence = 0xFC
	CommandCodeAutoSensing = 0xFE
	CommandCodeMetaEvent = 0xFF
)

const (
	MetaEventTrackSequenceNumber = 0x00
	MetaEventTextEvent = 0x01
	MetaEventCopyright = 0x02
	MetaEventSequenceTrackName = 0x03
	MetaEventTrackInstrumentName = 0x04
	MetaEventLyric = 0x05
	MetaEventMarker = 0x06
	MetaEventCuePoint = 0x07
	MetaEventProgramName = 0x08
	MetaEventDeviceName = 0x09
	MetaEventMidiChannel = 0x20
	MetaEventMidiPort = 0x21
	MetaEventEndTrack = 0x2F
	MetaEventSetTempo = 0x51
	MetaEventSmpteOffset = 0x54
	MetaEventTimeSignature = 0x58
	MetaEventKeySignature = 0x59
	MetaEventSequencerSpecific = 0x7F
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

type Midi struct {
	buffer        *bytes.Buffer
	mthd_length   int32
	mthd_format   int16
	mthd_tracks   int16
	mthd_division int16
	mtrk_length   int32

	commandCode   byte
	channel       uint8
}

func NewMidi(b *bytes.Buffer) *Midi {
	return &Midi{
		buffer: b,
	}
}

func (midi *Midi) ReadMThd() {
	midi.ReadMThdMarker()

	binary.Read(midi.buffer, binary.BigEndian, &midi.mthd_length)
	fmt.Printf("length %d\n", midi.mthd_length)

	binary.Read(midi.buffer, binary.BigEndian, &midi.mthd_format)
	fmt.Printf("format %d\n", midi.mthd_format)

	binary.Read(midi.buffer, binary.BigEndian, &midi.mthd_tracks)
	fmt.Printf("tracks %d\n", midi.mthd_tracks)

	binary.Read(midi.buffer, binary.BigEndian, &midi.mthd_division)
	fmt.Printf("division %d\n", midi.mthd_division)
}

func (midi *Midi) ReadMTrk() {
	midi.readMTrkMarker()

	binary.Read(midi.buffer, binary.BigEndian, &midi.mtrk_length)
	fmt.Printf("mtrk length %d\n", midi.mtrk_length)

	switch midi.mthd_format {
	case 0:
		midi.ReadMTrkFormat0()
	case 1:
		end := midi.buffer.Len() - int(midi.mtrk_length)
		var time uint64
		for midi.buffer.Len() > end {
			time = time + midi.ReadDeltaTime()
			fmt.Printf("%d", time)
			midi.ReadEvent()
			fmt.Print("\n")
		}
	case 2:
		midi.ReadMTrkFormat2()
	default:
		panic(errInvalidHeader)
	}
}

func (midi *Midi) ReadMTrkFormat0() {
	panic(errNotImplemented)
}

func (midi *Midi) ReadMTrkFormat2() {
	panic(errNotImplemented)
}

func (midi *Midi) ReadEvent() {
	// todo check last event type if current event type is & 0x80 == 0
	b := midi.ReadByte()
	var commandCode byte
	var channel uint8 = 1
	if b & 0x80 == 0 {
		// a running command - command & channel are same as previous
		commandCode = midi.commandCode
		channel = midi.channel
		midi.buffer.UnreadByte()
	}else {
		if b & 0xF0 == 0xF0 {
			commandCode = b
		}else {
			commandCode = b & 0xF0
			channel = b & 0x0F + 1
		}
	}

	midi.commandCode = commandCode
	midi.channel = channel

	fmt.Printf(", command code %X", commandCode)
	fmt.Printf(", %d", midi.channel)

	switch commandCode {
	case CommandCodeNoteOn:
		midi.ReadNoteOnEvent()
	case CommandCodeNoteOff,
		CommandCodeKeyAfterTouch:
		midi.ReadNoteOffEvent()
	case CommandCodeControlChange:
		midi.ReadControlChangeEvent()
	case CommandCodePatchChange:
		midi.ReadPatchChangeEvent()
	case CommandCodeChannelAfterTouch:
		midi.ReadChannelAfterTouchEvent()
	case CommandCodePitchWheelChange:
		midi.ReadPitchWheelEvent()
	case CommandCodeSysex:
		midi.ReadSysexEvent()
	case CommandCodeTimingClock,
		CommandCodeStartSequence,
		CommandCodeContinueSequence,
		CommandCodeStopSequence:
	// empty midi event
	case CommandCodeEox: panic(errNotImplemented)
	case CommandCodeAutoSensing: panic(errNotImplemented)
	case CommandCodeMetaEvent:
		midi.ReadMetaEvent()
	default:
		panic(errInvalidCommandCode)
	}
}

func (midi *Midi) ReadSysexEvent() {
	for {
		b := midi.ReadByte()
		if b == 0xF7 {
			break
		}else {
			fmt.Printf(", %X", b)
		}
	}
}

func (midi *Midi) ReadChannelAfterTouchEvent() {
	afterTouchPressure := midi.ReadByte()
	if afterTouchPressure & 0x80 != 0 {
		panic(errInvalidAfterTouchPressure)
	}
	fmt.Printf(", after touch pressure %d", afterTouchPressure)
}

func (midi *Midi) ReadPatchChangeEvent() {
	patch := midi.ReadByte()
	if patch & 0x80 != 0 {
		panic(errInvalidPatch)
	}
	fmt.Printf(", patch %d", patch)
}

func (midi *Midi) ReadPitchWheelEvent() {
	b1 := midi.ReadByte()
	b2 := midi.ReadByte()
	if b1 & 0x80 != 0 {
		panic(errInvalidPitchWheelByte)
	}
	if b2 & 0x80 != 0 {
		panic(errInvalidPitchWheelByte)
	}
	pitch := b1 + (b2 << 7)
	fmt.Printf(", Pitch_bend_c %d", pitch)
}

func (midi *Midi) ReadControlChangeEvent() {
	key := midi.ReadByte()
	pressure := midi.ReadByte()

	fmt.Printf(", Control_c, %d", key)
	fmt.Printf(", %d", pressure)
}

func (midi *Midi) ReadNoteOnEvent() {
	key := midi.ReadByte()
	velocity := midi.ReadByte()

	fmt.Printf(", Note_on_c, %d", key)
	fmt.Printf(", %d", velocity)
}

func (midi *Midi) ReadNoteOffEvent() {
	key := midi.ReadByte()
	velocity := midi.ReadByte()

	fmt.Printf(", Note_off_c, %d", key)
	fmt.Printf(", %d", velocity)
}

func (midi *Midi) ReadPitchBendVoiceMessage() {
	lsb := midi.ReadByte()
	msb := midi.ReadByte()

	fmt.Printf(", lsb: %d\n", lsb)
	fmt.Printf(", msb: %d\n", msb)
}

func (midi *Midi) ReadMetaEvent() {
	metaEvent := midi.ReadByte()
	len := midi.ReadUVarInt()

	fmt.Printf(", meta event %X", metaEvent)
	fmt.Printf(", len %d", len)

	switch metaEvent {
	case MetaEventTrackSequenceNumber:
		midi.ReadTrackSequenceNumber(len)
	case MetaEventTextEvent,
		MetaEventCopyright,
		MetaEventSequenceTrackName,
		MetaEventTrackInstrumentName,
		MetaEventLyric,
		MetaEventMarker,
		MetaEventCuePoint,
		MetaEventProgramName,
		MetaEventDeviceName:
		str := midi.ReadTextEvent(len)
		fmt.Printf(", [%s]", str)
	case MetaEventMidiChannel: panic(errNotImplemented)
	case MetaEventMidiPort: panic(errNotImplemented)
	case MetaEventEndTrack:
		if len != 0 {
			panic(errInvalidTrackEndLength)
		}
		fmt.Printf(", End_track")
	case MetaEventSetTempo:
		midi.ReadTempoEvent(len)
	case MetaEventSmpteOffset:
		midi.ReadSmpteOffsetEvent(len)
	case MetaEventTimeSignature:
		midi.ReadTimeSignatureEvent(len)
	case MetaEventKeySignature:
		midi.ReadKeySignatureEvent(len)
	case MetaEventSequencerSpecific:
		midi.ReadSequencerSpecificEvent(len)
	default:
		panic(errInvalidMetaEventType)
	}
}

func (midi *Midi) ReadSequencerSpecificEvent(len uint64) {
	bytes := midi.ReadBytes(int(len))

	fmt.Printf(", %v", bytes)
}

func (midi *Midi) ReadSmpteOffsetEvent(len uint64) {
	if(len != 5) {
		panic(errInvalidSmpteOffset)
	}
	hours := midi.ReadByte()
	minutes := midi.ReadByte()
	seconds := midi.ReadByte()
	frames := midi.ReadByte()
	subFrames := midi.ReadByte()

	fmt.Printf(", hours %d", hours)
	fmt.Printf(", minutes %d", minutes)
	fmt.Printf(", seconds %d", seconds)
	fmt.Printf(", frames %d", frames)
	fmt.Printf(", subFrames %d", subFrames)
}

func (midi *Midi) ReadKeySignatureEvent(len uint64) {
	if(len != 2) {
		panic(errInvalidKeySignatureLen)
	}
	sharpsFlats := midi.ReadByte() // sf=sharps/flats (-7=7 flats, 0=key of C,7=7 sharps)
	majorMinor := midi.ReadByte() // mi=major/minor (0=major, 1=minor)       }

	fmt.Printf(", sharps flats %d", sharpsFlats)
	fmt.Printf(", major minor %d", majorMinor)
}

func (midi *Midi) ReadTimeSignatureEvent(len uint64) {
	if len != 4 {
		panic(errInvalidTimeSignatureLength)
	}
	numerator := midi.ReadByte()
	denominator := midi.ReadByte() //2=quarter, 3=eigth etc
	ticksInMetronomeClick := midi.ReadByte()
	no32ndNotesInQuarterNote := midi.ReadByte()

	fmt.Printf(", numerator %d", numerator)
	fmt.Printf(", denominator %d", denominator)
	fmt.Printf(", ticks in metronome click %d", ticksInMetronomeClick)
	fmt.Printf(", no 32nd notes in quarter note %d", no32ndNotesInQuarterNote)
}

func (midi *Midi) ReadTextEvent(len uint64) string {
	if len == 0 {
		return ""
	} else {
		b := midi.ReadBytes(int(len))
		return string(b)
	}
}

func (midi *Midi) ReadTempoEvent(len uint64) {
	if len != 3 {
		panic(errInvalidTempoLength)
	}
	microsecondsPerQuarterNote := (midi.ReadByte() << 16) + (midi.ReadByte() << 8) + midi.ReadByte()

	fmt.Printf(", microseconds per quarter note: %d", microsecondsPerQuarterNote)
}

func (midi *Midi) ReadTrackSequenceNumber(len uint64) {
	panic(errNotImplemented)
}

func (midi *Midi) ReadDeltaTime() (uint64) {
	return midi.ReadUVarInt()
}

func (midi *Midi) ReadBytes(bytes int) []byte {
	buffer := make([]byte, bytes, bytes)
	l, err := midi.buffer.Read(buffer)
	if err != nil {
		panic(err)
	}
	if l != len(buffer) {
		panic(errEOF)
	}
	return buffer
}

func (midi *Midi) ReadUVarInt() uint64 {
	var x uint64
	var s uint
	for i := 0;; i++ {
		b, err := midi.buffer.ReadByte()
		if err != nil {
			panic(errVarInt32Overflow)
		}
		if b < 0x80 {
			if i > 5 || i == 5 && b > 1 {
				panic(errVarInt32Overflow)
			}
			return x | uint64(b) << s
		}
		x |= uint64(b & 0x7f) << s
		s += 7
	}
}

func (midi *Midi) ReadUint24() uint32 {
	return uint32(midi.ReadByte()) | uint32(midi.ReadByte()) << 8 | uint32(midi.ReadByte()) << 16
}

func (midi *Midi) ReadByte() byte {
	b, err := midi.buffer.ReadByte()
	check(err)
	return b
}

func (midi *Midi) ReadMThdMarker() {
	buffer := midi.ReadBytes(4)
	fmt.Printf("%s\n", buffer)
	if string(buffer) != "MThd" {
		panic(errInvalidHeader)
	}
}

func (midi *Midi) readMTrkMarker() {
	buffer := midi.ReadBytes(4)
	fmt.Printf("%s\n", buffer)
	if string(buffer) != "MTrk" {
		panic(errInvalidHeader)
	}
}