package i2c

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"gobot.io/x/gobot"
)

var _ gobot.Driver = (*HT16K33Driver)(nil)

func initTestHT16K33Driver() *HT16K33Driver {
	driver, _ := initTestHT16K33DriverWithStubbedAdaptor()
	return driver
}

func initTestHT16K33DriverWithStubbedAdaptor() (*HT16K33Driver, *i2cTestAdaptor) {
	adaptor := newI2cTestAdaptor()
	return NewHT16K33Driver(adaptor), adaptor
}

// i2cWriteByteToWord decodes bytes written in i2cTestAdapter.WriteWordData()
// into a uint16.
// TODO: it looks like i2cTestAdapter.WriteWordData() may have the high and low
// order mixed up, so swapping them here as a workaround.
func i2cWriteByteToWord(b []byte) (uint16, error) {
	if len(b) != 2 {
		return 0, errors.New("expected 2 bytes")
	}

	// high and low bits swapped?
	return uint16(b[1])<<8 | uint16(b[0]), nil
}

func TestNewHT16K33Driver(t *testing.T) {
	type args struct {
		a       Connector
		options []func(Config)
	}
	tests := []struct {
		name string
		args args
		want *HT16K33Driver
	}{
		{
			name: "normal case",
			args: args{
				a: newI2cTestAdaptor(),
			},
			want: &HT16K33Driver{},
		},
		{
			name: "no connector - should still get driver",
			args: args{
				a: nil,
			},
			want: &HT16K33Driver{},
		},
		{
			name: "with options",
			args: args{
				a: newI2cTestAdaptor(),
				options: []func(Config){
					WithAddress(0x71),
					WithBus(5),
				},
			},
			want: &HT16K33Driver{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got interface{} = NewHT16K33Driver(tt.args.a, tt.args.options...)
			_, ok := got.(*HT16K33Driver)
			if !ok {
				t.Errorf("NewHT16K33Driver() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHT16K33Driver_Name(t *testing.T) {
	tests := []struct {
		name string
		drv  *HT16K33Driver
		want string
	}{
		{
			name: "normal case",
			drv:  NewHT16K33Driver(newI2cTestAdaptor()),
			want: "HT16K33",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.drv.Name()
			if !strings.HasPrefix(got, tt.want) {
				t.Errorf("HT16K33Driver.Name() = %v, want prefix %v", got, tt.want)
			}
		})
	}
}

func TestHT16K33Driver_SetName(t *testing.T) {
	type fields struct {
		name string
	}
	type args struct {
		n string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "normal case",
			fields: fields{
				name: "foo",
			},
			args: args{
				n: "bar",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HT16K33Driver{
				name: tt.fields.name,
			}
			h.SetName(tt.args.n)
			if h.name != tt.args.n {
				t.Errorf("HT16K33Driver.SetName(%v) got %v, want %v", tt.args.n, h.name, tt.args.n)
			}
		})
	}
}

func TestHT16K33Driver_Start(t *testing.T) {
	tests := []struct {
		name      string
		writeFunc func([]byte) (int, error)
		wantErr   bool
	}{
		{
			name: "normal start",
		},
		{
			name: "write error",
			writeFunc: func([]byte) (int, error) {
				return 0, errors.New("write error")
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h, a := initTestHT16K33DriverWithStubbedAdaptor()

			if tt.writeFunc != nil {
				a.i2cWriteImpl = tt.writeFunc
			}

			if err := h.Start(); (err != nil) != tt.wantErr {
				t.Errorf("HT16K33Driver.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHT16K33Driver_Halt(t *testing.T) {

	h, _ := initTestHT16K33DriverWithStubbedAdaptor()

	if err := h.Halt(); err != nil {
		t.Errorf("HT16K33Driver.Halt() error = %v, wantErr nil", err)
	}
}

func TestHT16K33Driver_SetDisplay(t *testing.T) {
	type args struct {
		on bool
	}
	tests := []struct {
		name    string
		args    args
		want    uint8
		wantErr bool
	}{
		{
			name: "on",
			args: args{
				on: true,
			},
			want: ht16k33RegDisplay | ht16k33DisplayOn,
		},
		{
			name: "off",
			args: args{
				on: false,
			},
			want: ht16k33RegDisplay | 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h, a := initTestHT16K33DriverWithStubbedAdaptor()
			h.Start()

			a.i2cWriteImpl = func(got []byte) (int, error) {

				if len(got) != 1 {
					t.Errorf("Sequence error, expected 1 byte, got %d", len(got))
				}

				if !reflect.DeepEqual(got[0], tt.want) {
					t.Logf("Sequence error, got %+v, expected %+v", got[0], tt.want)
					return 0, fmt.Errorf("error")
				}
				return 0, nil
			}
			if err := h.SetDisplay(tt.args.on); (err != nil) != tt.wantErr {
				t.Errorf("HT16K33Driver.SetDisplay() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHT16K33Driver_SetBrightness(t *testing.T) {
	type args struct {
		b uint8
	}
	tests := []struct {
		name    string
		args    args
		want    uint8
		wantErr bool
	}{
		{
			name: "off",
			args: args{
				b: 0,
			},
			want: ht16k33RegBrightness | 0,
		},
		{
			name: "max",
			args: args{
				b: 15,
			},
			want: ht16k33RegBrightness | 15,
		},
		{
			name: "out of bounds - set to max",
			args: args{
				b: 16,
			},
			want: ht16k33RegBrightness | 15,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h, a := initTestHT16K33DriverWithStubbedAdaptor()
			h.Start()

			a.i2cWriteImpl = func(got []byte) (int, error) {

				if len(got) != 1 {
					t.Errorf("Sequence error, expected 1 byte, got %d", len(got))
				}

				if !reflect.DeepEqual(got[0], tt.want) {
					t.Logf("Sequence error, got %+v, expected %+v", got[0], tt.want)
					return 0, fmt.Errorf("error")
				}
				return 0, nil
			}
			if err := h.SetBrightness(tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("HT16K33Driver.SetBrightness() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHT16K33Driver_Colon(t *testing.T) {
	type args struct {
		on bool
	}
	tests := []struct {
		name    string
		args    args
		want    uint16
		wantErr bool
	}{
		{
			name: "on",
			args: args{
				on: true,
			},
			want: 0x02,
		},
		{
			name: "off",
			args: args{
				on: false,
			},
			want: 0x00,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h, a := initTestHT16K33DriverWithStubbedAdaptor()
			h.Start()

			a.i2cWriteImpl = func(got []byte) (int, error) {

				word, err := i2cWriteByteToWord(got)
				if err != nil {
					t.Fatalf("Sequence error, got error %v", err)
				}

				if !reflect.DeepEqual(word, tt.want) {
					t.Logf("Sequence error, got %+v, expected %+v", word, tt.want)
					return 0, fmt.Errorf("error")
				}
				return 0, nil
			}
			if err := h.Colon(tt.args.on); (err != nil) != tt.wantErr {
				t.Errorf("HT16K33Driver.Colon() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHT16K33Driver_Clear(t *testing.T) {

	h, _ := initTestHT16K33DriverWithStubbedAdaptor()
	h.Start()

	if err := h.Start(); err != nil {
		t.Errorf("HT16K33Driver.Clear() error = %v, wantErr nil", err)
	}
}

func TestHT16K33Driver_WriteBinary(t *testing.T) {
	type args struct {
		pos uint8
		b   string
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "0000000000000000",
			args: args{
				pos: 0,
				b:   "0000000000000000",
			},
			want: []byte{0x00, 0x00},
		},
		{
			name: "0000000001001111",
			args: args{
				pos: 0,
				b:   "0000000001001111",
			},
			want: []byte{0x4F, 0x00},
		},
		{
			name: "0000000011111111",
			args: args{
				pos: 0,
				b:   "0000000011111111",
			},
			want: []byte{0xFF, 0x00},
		},
		{
			name: "1111111100000000",
			args: args{
				pos: 0,
				b:   "1111111100000000",
			},
			want: []byte{0x00, 0xFF},
		},
		{
			name: "11111111",
			args: args{
				pos: 0,
				b:   "11111111",
			},
			want: []byte{0xFF, 0x00},
		},
		{
			name: "too short",
			args: args{
				pos: 0,
				b:   "1111",
			},
			want:    []byte{},
			wantErr: true,
		},
		{
			name: "not binary",
			args: args{
				pos: 0,
				b:   "FFFFFFFF",
			},
			want:    []byte{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h, a := initTestHT16K33DriverWithStubbedAdaptor()
			h.Start()

			a.i2cWriteImpl = func(got []byte) (int, error) {
				if !reflect.DeepEqual(got, tt.want) {
					t.Logf("Sequence error, got %+v, expected %+v", got, tt.want)
					return 0, fmt.Errorf("error")
				}
				return 0, nil
			}

			if err := h.WriteBinary(tt.args.pos, tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("HT16K33Driver.WriteBinary() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHT16K33Driver_WriteDigit(t *testing.T) {
	type args struct {
		pos uint8
		d   int
	}
	tests := []struct {
		name    string
		args    args
		want    uint16
		wantErr bool
	}{
		{
			name: "0",
			args: args{
				pos: 0,
				d:   0,
			},
			want: 0x0C3F,
		},
		{
			name: "1",
			args: args{
				pos: 0,
				d:   1,
			},
			want: 0x0006,
		},
		{
			name: "2",
			args: args{
				pos: 1,
				d:   2,
			},
			want: 0x00DB,
		},
		{
			name: "3",
			args: args{
				pos: 1,
				d:   3,
			},
			want: 0x004F,
		},
		{
			name: "4",
			args: args{
				pos: 2,
				d:   4,
			},
			want: 0x00E6,
		},
		{
			name: "5",
			args: args{
				pos: 2,
				d:   5,
			},
			want: 0x00ED,
		},
		{
			name: "6",
			args: args{
				pos: 3,
				d:   6,
			},
			want: 0x00FD,
		},
		{
			name: "7",
			args: args{
				pos: 1,
				d:   7,
			},
			want: 0x0007,
		},
		{
			name: "8",
			args: args{
				pos: 1,
				d:   8,
			},
			want: 0x00FF,
		},
		{
			name: "9",
			args: args{
				pos: 1,
				d:   9,
			},
			want: 0x00EF,
		},
		{
			name: "10",
			args: args{
				pos: 1,
				d:   10,
			},
			wantErr: true,
		},
		{
			name: "invalid pos",
			args: args{
				pos: 4,
				d:   1,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h, a := initTestHT16K33DriverWithStubbedAdaptor()
			h.Start()

			a.i2cWriteImpl = func(got []byte) (int, error) {

				word, err := i2cWriteByteToWord(got)
				if err != nil {
					t.Fatalf("Sequence error, got error %v", err)
				}

				if !reflect.DeepEqual(word, tt.want) {
					t.Logf("Sequence error, got %+v, expected %+v", word, tt.want)
					return 0, fmt.Errorf("error")
				}
				return 0, nil
			}
			if err := h.WriteDigit(tt.args.pos, tt.args.d); (err != nil) != tt.wantErr {
				t.Errorf("HT16K33Driver.WriteDigit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHT16K33Driver_WriteNumber(t *testing.T) {
	type args struct {
		n int
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// {
		// 	name: "0",
		// 	args: args{
		// 		n: 0,
		// 	},
		// 	want: "0000",
		// },
		{
			name: "10",
			args: args{
				n: 10,
			},
			want: []byte{0x4F, 0x00},
		},
		// {
		// 	name: "100",
		// 	args: args{
		// 		n: 0,
		// 	},
		// 	want: []byte{0xFF, 0x00},
		// },
		// {
		// 	name: "1000",
		// 	args: args{
		// 		n: 0,
		// 	},
		// 	want: []byte{0x00, 0xFF},
		// },
		// {
		// 	name: "1234",
		// 	args: args{
		// 		n: 0,
		// 	},
		// 	want: []byte{0xFF, 0x00},
		// },
		// {
		// 	name: "9999",
		// 	args: args{
		// 		n: 0,
		// 	},
		// 	want: []byte{0xFF, 0x00},
		// },
		// {
		// 	name: "too big",
		// 	args: args{
		// 		n: 10000,
		// 	},
		// 	want:    []byte{},
		// 	wantErr: true,
		// },
		// {
		// 	name: "negative",
		// 	args: args{
		// 		n: -1,
		// 	},
		// 	want:    []byte{},
		// 	wantErr: true,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h, a := initTestHT16K33DriverWithStubbedAdaptor()
			h.Start()

			a.i2cWriteImpl = func(got []byte) (int, error) {

				word, err := i2cWriteByteToWord(got)
				if err != nil {
					t.Fatalf("Sequence error, got error %v", err)
				}

				if !reflect.DeepEqual(word, tt.want) {
					t.Logf("Sequence error, got %+v, expected %+v", word, tt.want)
					return 0, fmt.Errorf("error")
				}
				return 0, nil
			}

			if err := h.WriteNumber(tt.args.n); (err != nil) != tt.wantErr {
				t.Errorf("HT16K33Driver.WriteNumber() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_splitNumberIntoDigits(t *testing.T) {
	tests := []struct {
		name    string
		n       int
		want    []int
		wantErr bool
	}{
		{
			name: "1",
			n:    1,
			want: []int{0, 0, 0, 1},
		},
		{
			name: "10",
			n:    10,
			want: []int{0, 0, 1, 0},
		},
		{
			name: "100",
			n:    100,
			want: []int{0, 1, 0, 0},
		},
		{
			name: "1000",
			n:    1000,
			want: []int{1, 0, 0, 0},
		},
		{
			name: "1234",
			n:    1234,
			want: []int{1, 2, 3, 4},
		},
		{
			name:    "10000",
			n:       10000,
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := splitNumberIntoDigits(tt.n)
			if (err != nil) != tt.wantErr {
				t.Errorf("splitNumberIntoDigits() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitNumberIntoDigits() = %v, want %v", got, tt.want)
			}
		})
	}
}
