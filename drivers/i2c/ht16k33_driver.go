package i2c

import (
	"errors"
	"strconv"
	"strings"

	"gobot.io/x/gobot"
)

const ht16k33Address = 0x70

// registers
const (
	ht16k33RegSetup      byte = 0x20
	ht16k33RegBlink           = 0x80
	ht16k33RegDisplay         = 0x80
	ht16k33RegBrightness      = 0xE0
)

// settings
const (
	ht16k33OscillatorOn byte = 0x21
	ht16k33DisplayOn         = 0x01
	ht16k33ColonOn           = 0x02
	ht16k33Off               = 0x00
)

// digits calculated from converting binary to uint16, where binary positions
// map to the following led panel segments:
//
//	    0
//	    _
//	5 |   | 1
//	  |   |
//	    - <----- 6
//	4 |   | 2
//	  | _ |
//	    3
//
// e.g. 3 needs segments 0, 1, 2, 3 and 6, which is 0000000001001111 with 0
// starting from the right going left.  0000000001001111 converts to 0x004F in
// base 16.
var digit = []uint16{
	0x0C3F, // 0
	0x0006, // 1
	0x00DB, // 2
	0x004F, // 3
	0x00E6, // 4
	0x00ED, // 5
	0x00FD, // 6
	0x0007, // 7
	0x00FF, // 8
	0x00EF, // 9
}

var displayBuffer = make([]byte, 8)

// Errors
var (
	ErrNumberTooBig       = errors.New("number must be less than 10,000")
	ErrDigitTooBig        = errors.New("digit must be less than 10")
	ErrBinaryTooBig       = errors.New("value too big, maximum 65535")
	ErrPositionOutOfRange = errors.New("position must be 0 - 3")
)

// HT16K33Driver is a Driver for the Adafruit LED Backpack
// https://learn.adafruit.com/adafruit-led-backpack
type HT16K33Driver struct {
	name       string
	connector  Connector
	connection Connection
	Config
}

// NewHT16K33Driver creates a new driver with specified i2c interface
// Params:
//		conn Connector - the Adaptor to use with this Driver
//
// Optional params:
//		i2c.WithBus(int):	bus to use with this driver
//		i2c.WithAddress(int):	address to use with this driver
//
func NewHT16K33Driver(a Connector, options ...func(Config)) *HT16K33Driver {
	d := &HT16K33Driver{
		name:      gobot.DefaultName("HT16K33"),
		connector: a,
		Config:    NewConfig(),
	}

	for _, option := range options {
		option(d)
	}

	return d
}

// Name returns the name for this Driver
func (h *HT16K33Driver) Name() string { return h.name }

// SetName sets the name for this Driver
func (h *HT16K33Driver) SetName(n string) { h.name = n }

// Connection returns the connection for this Driver
func (h *HT16K33Driver) Connection() gobot.Connection { return h.connector.(gobot.Connection) }

// Start initializes the ht16k33
func (h *HT16K33Driver) Start() (err error) {

	bus := h.GetBusOrDefault(h.connector.GetDefaultBus())
	address := h.GetAddressOrDefault(ht16k33Address)

	h.connection, err = h.connector.GetConnection(address, bus)
	if err != nil {
		return err
	}

	// Start oscillator
	if err := h.connection.WriteByte(ht16k33RegSetup | ht16k33OscillatorOn); err != nil {
		return err
	}

	// Read first byte to initialise
	if _, err := h.connection.ReadByte(); err != nil {
		return err
	}

	// Turn on the display
	if err := h.SetDisplay(true); err != nil {
		return err
	}

	// Set maximum brightness
	if err := h.SetBrightness(15); err != nil {
		return err
	}

	return nil
}

// Halt returns true if devices is halted successfully
func (h *HT16K33Driver) Halt() (err error) { return }

// SetDisplay turns the display on if on is true, otherwise turns it off
func (h *HT16K33Driver) SetDisplay(on bool) error {

	var v byte = ht16k33Off
	if on {
		v = ht16k33DisplayOn
	}
	return h.connection.WriteByte(ht16k33RegDisplay | v)
}

// SetBrightness sets the display brightness between 0 (off) and 15 (maximum)
func (h *HT16K33Driver) SetBrightness(b uint8) error {
	if b > 15 {
		b = 15
	}
	return h.connection.WriteByte(ht16k33RegBrightness | b)
}

// Colon sets the colon value
func (h *HT16K33Driver) Colon(on bool) error {

	var v uint16 = ht16k33Off
	if on {
		v = ht16k33ColonOn
	}
	return h.connection.WriteWordData(0x04, v)
}

// Clear the display
func (h *HT16K33Driver) Clear() error {

	for p := uint8(0); p < 5; p++ {
		if err := h.connection.WriteWordData(p*2, 0x0000); err != nil {
			return err
		}
	}
	return nil
}

// WriteBinary takes segment on/off specified as a binary string and displays it
// at the specified position (0-3).
func (h *HT16K33Driver) WriteBinary(pos uint8, b string) error {

	if pos > 3 {
		return ErrPositionOutOfRange
	}

	// Skip colon at position 2
	if pos == 2 || pos == 3 {
		pos++
	}

	w, err := strconv.ParseUint(b, 2, 16)
	if err != nil {
		return err
	}
	if w > 65535 {
		return ErrBinaryTooBig
	}

	return h.connection.WriteWordData(pos*2, uint16(w))
}

// WriteDigit displays the digit (0-9) at the specified position (0-3).
func (h *HT16K33Driver) WriteDigit(pos uint8, d int) error {

	if pos > 3 {
		return ErrPositionOutOfRange
	}

	if d > 9 {
		return ErrDigitTooBig
	}

	// Skip colon at position 2
	if pos == 2 || pos == 3 {
		pos++
	}

	// Registers are 2 bytes wide
	return h.connection.WriteWordData(pos*2, digit[d])
}

// WriteNumber displays a 4-digit number on the panel.  Leading zeros are not
// shown.
func (h *HT16K33Driver) WriteNumber(n int) error {

	digits, err := splitNumberIntoDigits(n)
	if err != nil {
		return err
	}

	// Clear the panel.
	if err := h.Clear(); err != nil {
		return err
	}

	foundDigit := false
	for pos, digit := range digits {

		if digit > 0 {
			foundDigit = true
		}

		// Skip leading zeros
		if !foundDigit {
			continue
		}

		if err := h.WriteDigit(uint8(pos), digit); err != nil {
			return err
		}
	}

	return nil
}

// splitNumberIntoDigits takes an int < 10000 and splits it into left-padded
// digits.
func splitNumberIntoDigits(n int) ([]int, error) {

	out := make([]int, 4)

	if n > 9999 {
		return nil, ErrNumberTooBig
	}

	chars := strings.Split(strconv.Itoa(n), "")
	for i, char := range chars {
		digit, err := strconv.Atoi(char)
		if err != nil {
			return nil, err
		}

		// Right-justify
		out[4-len(chars)+i] = digit
	}

	return out, nil
}
