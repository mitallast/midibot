package midi

import (
	"bytes"
	"encoding/binary"
	"errors"
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
	errVarInt32Overflow           = errors.New("binary: varint overflows a 32-bit integer")
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
	mthd   Mthd
	mtrk   Mtrk
	event  Event
}

type Mthd struct {
	length   int32
	format   int16
	tracks   int16
	division int16
}

type Mtrk struct {
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

func (midi *Midi) Mthd() Mthd {
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

	return nil
}

func (midi *Midi) HasNextMTrk() bool {
	return midi.mtrk.track < midi.mthd.tracks
}

func (midi *Midi) ReadNextMTrk() error {
	midi.mtrk.track++
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
	midi.mtrk.end_pos = midi.buffer.Len() - int(midi.mtrk.length)
	return nil
}

func (midi *Midi) ReadMTrkFormat2() error {
	return errNotImplemented
}

func (midi *Midi) HasNextEvent() bool {
	return midi.buffer.Len() > midi.mtrk.end_pos
}

func (midi *Midi) ReadNextEvent() (MidiEvent, error) {
	var err error
	if midi.event.delta, err = midi.ReadUVarInt(); err != nil {
		return nil, err
	}
	midi.mtrk.time_pos += midi.event.delta
	event, err := midi.ReadEvent()
	return event, err
}

func (midi *Midi) ReadEvent() (MidiEvent, error) {
	b, err := midi.buffer.ReadByte()
	if err != nil {
		return nil, err
	}
	var commandCode byte
	var channel uint8 = 1
	if b&0x80 == 0 {
		// a running command - command & channel are same as previous
		commandCode = midi.event.commandCode
		channel = midi.event.channel
		if err := midi.buffer.UnreadByte(); err != nil {
			return nil, err
		}
	} else {
		if b&0xF0 == 0xF0 {
			commandCode = b
		} else {
			commandCode = b & 0xF0
			channel = b&0x0F + 1
		}
	}

	midi.event.commandCode = commandCode
	midi.event.channel = channel

	switch midi.event.commandCode {
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
		return midi.ReadAfterTouchEvent()
	case CommandCodePitchWheelChange:
		return midi.ReadPitchWheelEvent()
	case CommandCodeSysex:
		return midi.ReadSysexEvent()
	case CommandCodeTimingClock,
		CommandCodeStartSequence,
		CommandCodeContinueSequence,
		CommandCodeStopSequence:
		// empty midi event
		return nil, nil
	case CommandCodeEox:
		return nil, errNotImplemented
	case CommandCodeAutoSensing:
		return nil, errNotImplemented
	case CommandCodeMetaEvent:
		return midi.ReadMetaEvent()
	default:
		return nil, errInvalidCommandCode
	}
}

func (midi *Midi) ReadSysexEvent() (event MidiEvent, err error) {
	data := []byte{}
	for {
		var b byte
		b, err = midi.buffer.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == 0xF7 {
			break
		} else {
			data = append(data, b)
		}
	}
	event = &SysexEvent{
		midi.event,
		data,
	}
	return
}

func (midi *Midi) ReadAfterTouchEvent() (event MidiEvent, err error) {
	var pressure byte
	if pressure, err = midi.buffer.ReadByte(); err != nil {
		return nil, err
	}
	if pressure&0x80 != 0 {
		return nil, errInvalidAfterTouchPressure
	}
	event = &AfterTouchEvent{
		midi.event,
		pressure,
	}
	return
}

func (midi *Midi) ReadPatchChangeEvent() (event MidiEvent, err error) {
	var patch byte
	if patch, err = midi.buffer.ReadByte(); err != nil {
		return
	}
	if patch&0x80 != 0 {
		return nil, errInvalidPatch
	}
	event = &PatchChangeEvent{
		midi.event,
		patch,
	}
	return
}

func (midi *Midi) ReadPitchWheelEvent() (event MidiEvent, err error) {
	var b1, b2 byte
	if b1, err = midi.buffer.ReadByte(); err != nil {
		return nil, err
	}
	if b2, err = midi.buffer.ReadByte(); err != nil {
		return nil, err
	}
	if b1&0x80 != 0 {
		return nil, errInvalidPitchWheelByte
	}
	if b2&0x80 != 0 {
		return nil, errInvalidPitchWheelByte
	}
	pitch := int(b1) + int(b2<<7)
	event = &PitchWheelEvent{
		midi.event,
		pitch,
	}
	return
}

func (midi *Midi) ReadControlChangeEvent() (event MidiEvent, err error) {
	var key, pressure byte
	if key, err = midi.buffer.ReadByte(); err != nil {
		return nil, err
	}
	if pressure, err = midi.buffer.ReadByte(); err != nil {
		return nil, err
	}
	event = &ControlChangeEvent{
		midi.event,
		key,
		pressure,
	}
	return
}

func (midi *Midi) ReadNoteOnEvent() (event MidiEvent, err error) {
	var key, velocity byte
	if key, err = midi.buffer.ReadByte(); err != nil {
		return nil, err
	}
	if velocity, err = midi.buffer.ReadByte(); err != nil {
		return nil, err
	}
	event = &NoteOnEvent{
		midi.event,
		key,
		velocity,
	}
	return
}

func (midi *Midi) ReadNoteOffEvent() (event MidiEvent, err error) {
	var key, velocity byte
	if key, err = midi.buffer.ReadByte(); err != nil {
		return nil, err
	}
	if velocity, err = midi.buffer.ReadByte(); err != nil {
		return nil, err
	}
	event = &NoteOffEvent{
		midi.event,
		key,
		velocity,
	}
	return
}

func (midi *Midi) ReadMetaEvent() (MidiEvent, error) {
	metaEvent, err := midi.buffer.ReadByte()
	if err != nil {
		return nil, err
	}
	len, err := midi.ReadUVarInt()
	if err != nil {
		return nil, err
	}

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
		return midi.ReadTextEvent(len)
	case MetaEventMidiChannel:
		return nil, errNotImplemented
	case MetaEventMidiPort:
		return nil, errNotImplemented
	case MetaEventEndTrack:
		if len != 0 {
			return nil, errInvalidTrackEndLength
		}
		return nil, nil
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
		return nil, errInvalidMetaEventType
	}
}

func (midi *Midi) ReadSequencerSpecificEvent(len uint64) (event MidiEvent, err error) {
	var data []byte
	if data, err = midi.ReadBytes(int(len)); err != nil {
		return nil, err
	}
	event = &SequencerSpecificEvent{
		midi.event,
		data,
	}
	return
}

func (midi *Midi) ReadSmpteOffsetEvent(len uint64) (event MidiEvent, err error) {
	if len != 5 {
		return nil, errInvalidSmpteOffset
	}
	var hours, minutes, seconds, frames, subFrames byte
	if hours, err = midi.buffer.ReadByte(); err != nil {
		return nil, err
	}
	if minutes, err = midi.buffer.ReadByte(); err != nil {
		return nil, err
	}
	if seconds, err = midi.buffer.ReadByte(); err != nil {
		return nil, err
	}
	if frames, err = midi.buffer.ReadByte(); err != nil {
		return nil, err
	}
	if subFrames, err = midi.buffer.ReadByte(); err != nil {
		return nil, err
	}
	event = &SmpteOffsetEvent{
		midi.event,
		hours,
		minutes,
		seconds,
		frames,
		subFrames,
	}
	return
}

func (midi *Midi) ReadKeySignatureEvent(len uint64) (event MidiEvent, err error) {
	if len != 2 {
		return nil, errInvalidKeySignatureLen
	}
	var sharpsFlats, majorMinor byte
	// sf=sharps/flats (-7=7 flats, 0=key of C,7=7 sharps)
	if sharpsFlats, err = midi.buffer.ReadByte(); err != nil {
		return nil, err
	}
	// mi=major/minor (0=major, 1=minor)
	if majorMinor, err = midi.buffer.ReadByte(); err != nil {
		return nil, err
	}

	event = &KeySignatureEvent{
		midi.event,
		sharpsFlats,
		majorMinor,
	}
	return
}

func (midi *Midi) ReadTimeSignatureEvent(len uint64) (event MidiEvent, err error) {
	if len != 4 {
		return nil, errInvalidTimeSignatureLength
	}
	var numerator, denominator, ticksInMetronomeClick, no32ndNotesInQuarterNote byte
	if numerator, err = midi.buffer.ReadByte(); err != nil {
		return
	}
	//2=quarter, 3=eigth etc
	if denominator, err = midi.buffer.ReadByte(); err != nil {
		return
	}
	if ticksInMetronomeClick, err = midi.buffer.ReadByte(); err != nil {
		return
	}
	if no32ndNotesInQuarterNote, err = midi.buffer.ReadByte(); err != nil {
		return
	}

	event = &TimeSignatureEvent{
		midi.event,
		numerator,
		denominator,
		ticksInMetronomeClick,
		no32ndNotesInQuarterNote,
	}
	return
}

func (midi *Midi) ReadTextEvent(len uint64) (event MidiEvent, err error) {
	text := ""
	if len > 0 {
		var b []byte
		b, err = midi.ReadBytes(int(len))
		if err != nil {
			return
		}
		text = string(b)
	}

	event = &TextEvent{
		midi.event,
		text,
	}
	return
}

func (midi *Midi) ReadTempoEvent(len uint64) (event MidiEvent, err error) {
	if len != 3 {
		return nil, errInvalidTempoLength
	}
	var b1, b2, b3 byte
	if b1, err = midi.buffer.ReadByte(); err != nil {
		return
	}
	if b2, err = midi.buffer.ReadByte(); err != nil {
		return
	}
	if b3, err = midi.buffer.ReadByte(); err != nil {
		return
	}
	microsecondsPerQuarterNote := int(b1<<16) + int(b2<<8) + int(b3)
	event = &TempoEvent{
		midi.event,
		microsecondsPerQuarterNote,
	}
	return
}

func (midi *Midi) ReadTrackSequenceNumber(len uint64) (event MidiEvent, err error) {
	return nil, errNotImplemented
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
	if string(buffer) != "MTrk" {
		return errInvalidHeader
	}
	return nil
}
