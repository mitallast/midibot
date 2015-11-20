package midi

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	errEOF                        = errors.New("EOF")
	errNotImplemented             = errors.New("Not implemented")
	errInvalidHeader              = errors.New("Invalid header")
	errInvalidCommandCode         = errors.New("Invalid command code")
	errInvalidMetaEventType       = errors.New("Invalid meta event type")
	errInvalidTempoLength         = errors.New("Invalid tempo length")
	errInvalidTimeSignatureLength = errors.New("Invalid time signature length")
	errInvalidKeySignatureLen     = errors.New("Invalid key signature length")
	errInvalidTrackEndLength      = errors.New("Invalid track end length")
	errInvalidPitchWheelByte      = errors.New("Invalid pitch wheel byte")
	errInvalidPatch               = errors.New("Invalid patch")
	errInvalidAfterTouchPressure  = errors.New("Invalid after touch pressure")
	errInvalidSmpteOffset         = errors.New("Invalid SMPTE Offset")
	errVarInt32Overflow           = errors.New("binary: varint overflows a 64-bit integer")
)

const (
	CommandCodeNoteOff           = 0x80
	CommandCodeNoteOn            = 0x90
	CommandCodeKeyAfterTouch     = 0xA0
	CommandCodeControlChange     = 0xB0
	CommandCodePatchChange       = 0xC0
	CommandCodeChannelAfterTouch = 0xD0
	CommandCodePitchWheelChange  = 0xE0
	CommandCodeSysex             = 0xF0
	CommandCodeEox               = 0xF7
	CommandCodeTimingClock       = 0xF8
	CommandCodeStartSequence     = 0xFA
	CommandCodeContinueSequence  = 0xFB
	CommandCodeStopSequence      = 0xFC
	CommandCodeAutoSensing       = 0xFE
	CommandCodeMetaEvent         = 0xFF
)

const (
	MetaEventTrackSequenceNumber = 0x00
	MetaEventTextEvent           = 0x01
	MetaEventCopyright           = 0x02
	MetaEventSequenceTrackName   = 0x03
	MetaEventTrackInstrumentName = 0x04
	MetaEventLyric               = 0x05
	MetaEventMarker              = 0x06
	MetaEventCuePoint            = 0x07
	MetaEventProgramName         = 0x08
	MetaEventDeviceName          = 0x09
	MetaEventMidiChannel         = 0x20
	MetaEventMidiPort            = 0x21
	MetaEventEndTrack            = 0x2F
	MetaEventSetTempo            = 0x51
	MetaEventSmpteOffset         = 0x54
	MetaEventTimeSignature       = 0x58
	MetaEventKeySignature        = 0x59
	MetaEventSequencerSpecific   = 0x7F
)

type Midi struct {
	buffer *bytes.Buffer
	mthd   MidiMthd
	mtrk   MidiMtrk

	commandCode byte
	channel     uint8
}

type MidiMthd struct {
	length   int32
	format   int16
	tracks   int16
	division int16
}

type MidiMtrk struct {
	track    int16
	length   int32
	end_pos  int
	time_pos uint64
}

func NewMidi(b *bytes.Buffer) *Midi {
	return &Midi{
		buffer: b,
	}
}

func (midi *Midi) Mthd() MidiMthd {
	return midi.mthd
}

func (midi *Midi) ReadMThd() error {
	if err := midi.ReadMThdMarker(); err != nil {
		return err
	}

	if err := binary.Read(midi.buffer, binary.BigEndian, &midi.mthd.length); err != nil {
		return err
	}
	if err := binary.Read(midi.buffer, binary.BigEndian, &midi.mthd.format); err != nil {
		return err
	}
	if err := binary.Read(midi.buffer, binary.BigEndian, &midi.mthd.tracks); err != nil {
		return err
	}
	if err := binary.Read(midi.buffer, binary.BigEndian, &midi.mthd.division); err != nil {
		return err
	}

	fmt.Printf("format %d\n", midi.mthd.format)
	fmt.Printf("tracks %d\n", midi.mthd.tracks)
	fmt.Printf("length %d\n", midi.mthd.length)
	fmt.Printf("division %d\n", midi.mthd.division)
	return nil
}

func (midi *Midi) HasNextMTrk() bool {
	return midi.mtrk.track < midi.mthd.tracks
}

func (midi *Midi) ReadNextMTrk() error {
	midi.mtrk.track++
	fmt.Printf("MTrk %d\n", midi.mtrk.track)
	return midi.ReadMTrk()
}

func (midi *Midi) ReadMTrk() error {
	if err := midi.readMTrkMarker(); err != nil {
		return err
	}

	switch midi.mthd.format {
	case 0:
		return midi.ReadMTrkFormat0()
	case 1:
		return midi.ReadMTrkFormat1()
	case 2:
		return midi.ReadMTrkFormat2()
	default:
		return errInvalidHeader
	}
}

func (midi *Midi) ReadMTrkFormat0() error {
	return errNotImplemented
}

func (midi *Midi) ReadMTrkFormat1() error {
	if err := binary.Read(midi.buffer, binary.BigEndian, &midi.mtrk.length); err != nil {
		return err
	}
	fmt.Printf("mtrk length %d\n", midi.mtrk.length)
	midi.mtrk.end_pos = midi.buffer.Len() - int(midi.mtrk.length)
	return nil
}

func (midi *Midi) HasNextEvent() bool {
	return midi.buffer.Len() > midi.mtrk.end_pos
}

func (midi *Midi) ReadNextEvent() error {
	delta, err := midi.ReadUVarInt()
	if err != nil {
		return err
	}
	midi.mtrk.time_pos += delta
	fmt.Printf("%d", midi.mtrk.time_pos)
	if err := midi.ReadEvent(); err != nil {
		return err
	}
	fmt.Print("\n")
	return nil
}

func (midi *Midi) ReadMTrkFormat2() error {
	return errNotImplemented
}

func (midi *Midi) ReadEvent() error {
	b, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}
	var commandCode byte
	var channel uint8 = 1
	if b&0x80 == 0 {
		// a running command - command & channel are same as previous
		commandCode = midi.commandCode
		channel = midi.channel
		if err := midi.buffer.UnreadByte(); err != nil {
			return err
		}
	} else {
		if b&0xF0 == 0xF0 {
			commandCode = b
		} else {
			commandCode = b & 0xF0
			channel = b&0x0F + 1
		}
	}

	midi.commandCode = commandCode
	midi.channel = channel

	fmt.Printf(", %d", midi.channel)

	switch commandCode {
	case CommandCodeNoteOn:
		return midi.ReadNoteOnEvent()
	case CommandCodeNoteOff,
		CommandCodeKeyAfterTouch:
		return midi.ReadNoteOffEvent()
	case CommandCodeControlChange:
		return midi.ReadControlChangeEvent()
	case CommandCodePatchChange:
		return midi.ReadPatchChangeEvent()
	case CommandCodeChannelAfterTouch:
		return midi.ReadChannelAfterTouchEvent()
	case CommandCodePitchWheelChange:
		return midi.ReadPitchWheelEvent()
	case CommandCodeSysex:
		return midi.ReadSysexEvent()
	case CommandCodeTimingClock,
		CommandCodeStartSequence,
		CommandCodeContinueSequence,
		CommandCodeStopSequence:
		// empty midi event
		return nil
	case CommandCodeEox:
		return errNotImplemented
	case CommandCodeAutoSensing:
		return errNotImplemented
	case CommandCodeMetaEvent:
		return midi.ReadMetaEvent()
	default:
		return errInvalidCommandCode
	}
}

func (midi *Midi) ReadSysexEvent() error {
	for {
		b, err := midi.buffer.ReadByte()
		if err != nil {
			return err
		}
		if b == 0xF7 {
			break
		} else {
			fmt.Printf(", %X", b)
		}
	}
	return nil
}

func (midi *Midi) ReadChannelAfterTouchEvent() error {
	afterTouchPressure, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}
	if afterTouchPressure&0x80 != 0 {
		return errInvalidAfterTouchPressure
	}
	fmt.Printf(", after touch pressure %d", afterTouchPressure)
	return nil
}

func (midi *Midi) ReadPatchChangeEvent() error {
	patch, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}
	if patch&0x80 != 0 {
		return errInvalidPatch
	}
	fmt.Printf(", patch %d", patch)
	return nil
}

func (midi *Midi) ReadPitchWheelEvent() error {
	b1, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}
	b2, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}
	if b1&0x80 != 0 {
		return errInvalidPitchWheelByte
	}
	if b2&0x80 != 0 {
		return errInvalidPitchWheelByte
	}
	pitch := b1 + (b2 << 7)
	fmt.Printf(", Pitch_bend_c %d", pitch)
	return nil
}

func (midi *Midi) ReadControlChangeEvent() error {
	key, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}
	pressure, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}

	fmt.Printf(", Control_c, %d", key)
	fmt.Printf(", %d", pressure)
	return nil
}

func (midi *Midi) ReadNoteOnEvent() error {
	key, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}
	velocity, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}

	fmt.Printf(", Note_on_c, %d", key)
	fmt.Printf(", %d", velocity)
	return nil
}

func (midi *Midi) ReadNoteOffEvent() error {
	key, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}
	velocity, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}

	fmt.Printf(", Note_off_c, %d", key)
	fmt.Printf(", %d", velocity)
	return nil
}

func (midi *Midi) ReadMetaEvent() error {
	metaEvent, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}
	len, err := midi.ReadUVarInt()
	if err != nil {
		return err
	}

	fmt.Printf(", meta event %X", metaEvent)
	fmt.Printf(", len %d", len)

	switch metaEvent {
	case MetaEventTrackSequenceNumber:
		return midi.ReadTrackSequenceNumber(len)
	case MetaEventTextEvent,
		MetaEventCopyright,
		MetaEventSequenceTrackName,
		MetaEventTrackInstrumentName,
		MetaEventLyric,
		MetaEventMarker,
		MetaEventCuePoint,
		MetaEventProgramName,
		MetaEventDeviceName:
		str, err := midi.ReadTextEvent(len)
		if err != nil {
			return err
		}
		fmt.Printf(", [%s]", str)
		return nil
	case MetaEventMidiChannel:
		panic(errNotImplemented)
	case MetaEventMidiPort:
		panic(errNotImplemented)
	case MetaEventEndTrack:
		if len != 0 {
			return errInvalidTrackEndLength
		}
		fmt.Printf(", End_track")
		return nil
	case MetaEventSetTempo:
		return midi.ReadTempoEvent(len)
	case MetaEventSmpteOffset:
		return midi.ReadSmpteOffsetEvent(len)
	case MetaEventTimeSignature:
		return midi.ReadTimeSignatureEvent(len)
	case MetaEventKeySignature:
		return midi.ReadKeySignatureEvent(len)
	case MetaEventSequencerSpecific:
		return midi.ReadSequencerSpecificEvent(len)
	default:
		return errInvalidMetaEventType
	}
}

func (midi *Midi) ReadSequencerSpecificEvent(len uint64) error {
	bytes, err := midi.ReadBytes(int(len))
	if err != nil {
		return err
	}

	fmt.Printf(", %v", bytes)
	return nil
}

func (midi *Midi) ReadSmpteOffsetEvent(len uint64) error {
	if len != 5 {
		panic(errInvalidSmpteOffset)
	}
	hours, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}
	minutes, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}
	seconds, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}
	frames, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}
	subFrames, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}
	fmt.Printf(", hours %d", hours)
	fmt.Printf(", minutes %d", minutes)
	fmt.Printf(", seconds %d", seconds)
	fmt.Printf(", frames %d", frames)
	fmt.Printf(", subFrames %d", subFrames)
	return nil
}

func (midi *Midi) ReadKeySignatureEvent(len uint64) error {
	if len != 2 {
		return errInvalidKeySignatureLen
	}
	sharpsFlats, err := midi.buffer.ReadByte() // sf=sharps/flats (-7=7 flats, 0=key of C,7=7 sharps)
	if err != nil {
		return err
	}
	majorMinor, err := midi.buffer.ReadByte() // mi=major/minor (0=major, 1=minor)       }
	if err != nil {
		return err
	}

	fmt.Printf(", sharps flats %d", sharpsFlats)
	fmt.Printf(", major minor %d", majorMinor)
	return nil
}

func (midi *Midi) ReadTimeSignatureEvent(len uint64) error {
	if len != 4 {
		panic(errInvalidTimeSignatureLength)
	}
	numerator, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}
	denominator, err := midi.buffer.ReadByte() //2=quarter, 3=eigth etc
	if err != nil {
		return err
	}
	ticksInMetronomeClick, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}
	no32ndNotesInQuarterNote, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}

	fmt.Printf(", numerator %d", numerator)
	fmt.Printf(", denominator %d", denominator)
	fmt.Printf(", ticks in metronome click %d", ticksInMetronomeClick)
	fmt.Printf(", no 32nd notes in quarter note %d", no32ndNotesInQuarterNote)
	return nil
}

func (midi *Midi) ReadTextEvent(len uint64) (string, error) {
	if len == 0 {
		return "", nil
	} else {
		b, err := midi.ReadBytes(int(len))
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}

func (midi *Midi) ReadTempoEvent(len uint64) error {
	if len != 3 {
		panic(errInvalidTempoLength)
	}
	b1, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}
	b2, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}
	b3, err := midi.buffer.ReadByte()
	if err != nil {
		return err
	}

	microsecondsPerQuarterNote := (b1 << 16) + (b2 << 8) + b3

	fmt.Printf(", microseconds per quarter note: %d", microsecondsPerQuarterNote)
	return nil
}

func (midi *Midi) ReadTrackSequenceNumber(len uint64) error {
	return errNotImplemented
}

func (midi *Midi) ReadBytes(bytes int) ([]byte, error) {
	buffer := make([]byte, bytes, bytes)
	l, err := midi.buffer.Read(buffer)
	if err != nil {
		return nil, err
	}
	if l != len(buffer) {
		return nil, errEOF
	}
	return buffer, nil
}

func (midi *Midi) ReadUVarInt() (uint64, error) {
	var x uint64
	var s uint
	for i := 0; ; i++ {
		b, err := midi.buffer.ReadByte()
		if err != nil {
			return 0, errVarInt32Overflow
		}
		if b < 0x80 {
			if i > 5 || i == 5 && b > 1 {
				return 0, errVarInt32Overflow
			}
			return x | uint64(b)<<s, nil
		}
		x |= uint64(b&0x7f) << s
		s += 7
	}
}

func (midi *Midi) ReadMThdMarker() error {
	buffer, err := midi.ReadBytes(4)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", buffer)
	if string(buffer) != "MThd" {
		return errInvalidHeader
	}
	return nil
}

func (midi *Midi) readMTrkMarker() error {
	buffer, err := midi.ReadBytes(4)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", buffer)
	if string(buffer) != "MTrk" {
		return errInvalidHeader
	}
	return nil
}
