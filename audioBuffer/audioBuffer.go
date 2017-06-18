package audioBuffer

import (
	"log"
	"fmt"
	"math"
)

/*
 * audioBuffer:
 */

type Buffer interface{}

type Block struct {
	buf         Buffer
	sampleRate  int
	Timestamp   float64
	Id          int
	empty       bool
}

func NewBlock(buffer Buffer, sampleRate int) *Block {
	ab := Block{buffer, sampleRate, 0.0, 0, false}
	return &ab
}

func (b *Block) Size() (size int) {
	switch d := b.buf.(type) {
	case []int16:
		size = len(d)
	case []float32:
		size = len(d)
	default:
		log.Fatalf("audioBuffer.Size(): Unrecognised buffer data type %v", d)
	}

	return
}

func (b *Block) DataFormat() (string, error) {
	var t string

	switch b.buf.(type) {
	case []int16:
		t = "int16"
	case []float32:
		t = "float32"
	default:
		return "", fmt.Errorf("NewWav: unrecognised buffer format: %v", b.buf)
	}

	return t, nil
}

func (block *Block) Data() Buffer {
	return block.buf
}

func (block *Block) SetBuffer(buf Buffer) {
	block.buf = buf
}

// convert data block to float64 (if needed) and return
func (block *Block) Float64Data() ([]float64, error) {
	var f64 []float64

	switch d := block.buf.(type) {
	case []int16:
		f64 = make([]float64, len(d))
		for i, v := range d {
			f64[i] = (float64(v) - math.MinInt16) / (math.MaxInt16 - math.MinInt16)
		}
	case []float32:
		f64 = make([]float64, len(d))
		for i, v := range d {
			f64[i] = float64(v)
		}
	default:
		return nil, fmt.Errorf("Block.Float64Data: unrecognised buffer data type: %v", d)
	}
	return f64, nil
}

// convert data block to int16 (if needed) and return
func (block *Block) Int16Data() ([]int16, error) {
	var i16 []int16

	switch d := block.buf.(type) {
	case []int16:
		i16 = d
	default:
		return nil, fmt.Errorf("Block.Int16Data: unrecognised buffer data type: %v", d)
	}
	return i16, nil
}

func (buf *Block) UpdateReadCount(count int) {
	if buf.empty {
		buf.Id++
		buf.Timestamp= float64(buf.Id * count) / float64(buf.sampleRate)
	} else {
		buf.empty = true
	}
}

