package pcm

import (
"log"
	"reflect"
)

/*
 * pcm.Buffer:
 * Provide a polymorphic audio buffer class to be used with audio streams.
 * Supports int16 and float32 data formats and allows some type conversions.
 * Note: The type conversion oprations here will not convert between different audio formats
 *  TThe int 16 and float 32 audio formats are scaled differently so must be converted separately
 */

const (
	FMT_FLOAT32 = "f32"
	FMT_INT16   = "s16"
)

const FRAME_SIZE = 4096

// Frame can be multiple data types but will always be a slice of raw audio data
type Frame interface{}

type Buffer struct {
	frame       Frame
	sampleRate  int
	size	    int
	Timestamp   float64
	Id          int
	empty       bool

	fnDataFormat  dataFormat
	fnAsFloat64   asFloat64
	fnAsInt16     asInt16
	fnSetFrame    setFrame
	// asFormat(type)
	// asType(type)
}

type dataFormat func() string
type asFloat64  func() []float64
type asInt16    func() []int16
type setFrame   func(*Buffer, interface {})

func (b *Buffer) DataFormat() string     { return b.fnDataFormat() }
func (b *Buffer) AsFloat64() []float64   { return b.fnAsFloat64() }
func (b *Buffer) AsInt16() []int16       { return b.fnAsInt16() }
func (b *Buffer) SetFrame(d interface{}) { b.fnSetFrame(b, d) }

func (b *Buffer) Size() int {
	return b.size
}

func (b *Buffer) Frame() interface{} {
	return b.frame
}

func (buf *Buffer) UpdateReadCount(count int) {
	if buf.empty {
		buf.Id++
		buf.Timestamp= float64(buf.Id * count) / float64(buf.sampleRate)
	} else {
		buf.empty = true
	}
}


func NewIntBuffer(sampleRate int) *Buffer {

	frame := make([]int16, FRAME_SIZE)

	dataFormat := func() string {
		return FMT_INT16
	}

	asFloat64 := func() []float64 {
		out := make([]float64, FRAME_SIZE)
		for i, x := range frame {
			out[i] = float64(x)
		}

		return out
	}

	asInt16 := func() []int16 {
		return frame
	}

	setFrame := func(b *Buffer, data interface{}) {
		d, ok := data.([]int16)
		if !ok {
			log.Panicf("FloatBuffer: setFrame from %s requires conversion, unsupported", reflect.TypeOf(data))
		}
		if (len(d) != b.size) {
			log.Panicf("Buffer.setFrame: incompatible buffer lengths (%d, %d)", len(d), b.size)
		}

		copy(frame, d)
		b.UpdateReadCount(len(d))
	}

	fb := Buffer{
		frame,
		sampleRate,
		FRAME_SIZE,
		0.0,
		0,
		false,
		dataFormat,
		asFloat64,
		asInt16,
		setFrame,
	}

	return &fb
}

func NewFloatBuffer(sampleRate int) *Buffer {

	frame := make([]float32, FRAME_SIZE)

	dataFormat := func() string {
		return FMT_FLOAT32
	}

	asFloat64 := func() []float64 {
		out := make([]float64, FRAME_SIZE)
		for i, x := range frame {
			out[i] = float64(x)
		}

		return out
	}

	asInt16 := func() []int16 {
		log.Panicln("FloatBuffer: type coversion to int16 not supported")
		return nil
	}

	setFrame := func(b *Buffer, data interface{}) {
		d, ok := data.([]float32)
		if !ok {
			log.Panicf("FloatBuffer: setFrame from %s requires conversion, unsupported", reflect.TypeOf(data))
		}

		if (len(d) != b.size) {
			log.Panicf("Buffer.setFrame: incompatible buffer lengths (%d, %d)", len(d), b.size)
		}
		copy(frame, d)
		b.UpdateReadCount(len(d))
	}

	fb := Buffer{
		frame,
		sampleRate,
		FRAME_SIZE,
		0.0,
		0,
		false,
		dataFormat,
		asFloat64,
		asInt16,
		setFrame,
	}

	return &fb
}

