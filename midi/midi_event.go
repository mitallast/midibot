package midi

import "fmt"

type MidiEvent interface {
}

type Event struct {
	delta       uint64
	commandCode byte
	channel     uint8
}

type NoteOnEvent struct {
	Event
	key      byte
	velocity byte
}

func (e *NoteOnEvent) String() string {
	return fmt.Sprintf("%d, %d, Note_on_c, %d, %d", e.delta, e.channel, e.key, e.velocity)
}

type NoteOffEvent struct {
	Event
	key      byte
	velocity byte
}

func (e *NoteOffEvent) String() string {
	return fmt.Sprintf("%d, %d, Note_off_c, %d, %d", e.delta, e.channel, e.key, e.velocity)
}

type ControlChangeEvent struct {
	Event
	key      byte
	pressure byte
}

func (e *ControlChangeEvent) String() string {
	return fmt.Sprintf("%d, %d, Control_c, %d, %d", e.delta, e.channel, e.key, e.pressure)
}

type PatchChangeEvent struct {
	Event
	patch byte
}

func (e *PatchChangeEvent) String() string {
	return fmt.Sprintf("%d, %d, Patch_c, %d", e.delta, e.channel, e.patch)
}

type AfterTouchEvent struct {
	Event
	pressure byte
}

func (e *AfterTouchEvent) String() string {
	return fmt.Sprintf("%d, %d, After_touch_c, %d", e.delta, e.channel, e.pressure)
}

type PitchWheelEvent struct {
	Event
	pitch int
}

func (e *PitchWheelEvent) String() string {
	return fmt.Sprintf("%d, %d, Pitch_wheel_c, %d", e.delta, e.channel, e.pitch)
}

type SysexEvent struct {
	Event
	data []byte
}

func (e *SysexEvent) String() string {
	return fmt.Sprintf("%d, %d, Sysex_c, %X", e.delta, e.channel, e.data)
}

type SequencerSpecificEvent struct {
	Event
	data []byte
}

func (e *SequencerSpecificEvent) String() string {
	return fmt.Sprintf("%d, %d, Sequencer_Specific_c, %X", e.delta, e.channel, e.data)
}

type SmpteOffsetEvent struct {
	Event
	hours     byte
	minutes   byte
	seconds   byte
	frames    byte
	subFrames byte
}

func (e *SmpteOffsetEvent) String() string {
	return fmt.Sprintf("%d, %d, Smpte_offset_c, %d, %d, %d, %d, %d", e.delta, e.channel, e.hours, e.minutes, e.seconds, e.frames, e.subFrames)
}

type KeySignatureEvent struct {
	Event
	sharpsFlats byte
	majorMinor  byte
}

func (e *KeySignatureEvent) String() string {
	return fmt.Sprintf("%d, %d, Key_signature_c, %d, %d", e.delta, e.channel, e.sharpsFlats, e.majorMinor)
}

type TimeSignatureEvent struct {
	Event
	numerator                byte
	denominator              byte
	ticksInMetronomeClick    byte
	no32ndNotesInQuarterNote byte
}

func (e *TimeSignatureEvent) String() string {
	return fmt.Sprintf("%d, %d, Time_signature_c, %d, %d, %d, %d, %d", e.delta, e.channel, e.numerator, e.denominator, e.ticksInMetronomeClick, e.no32ndNotesInQuarterNote)
}

type TextEvent struct {
	Event
	text string
}

func (e *TextEvent) String() string {
	return fmt.Sprintf("%d, %d, Text_c, %s", e.delta, e.channel, e.text)
}

type TempoEvent struct {
	Event
	microsecondsPerQuarterNote int
}

func (e *TempoEvent) String() string {
	return fmt.Sprintf("%d, %d, Text_c, %d", e.delta, e.channel, e.microsecondsPerQuarterNote)
}
