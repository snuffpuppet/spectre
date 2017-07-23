package pcm

import (
	"os/exec"
	"github.com/mjibson/go-dsp/wav"
	"io"
	"fmt"
	//"log"
	"github.com/snuffpuppet/spectre/ffmpeg"
)

/*
 * audioStream:
 * Provide abstraction over an audio stream source.
 * File streams are provided via ffmpeg decoding and microphone streams are provided through the portaudio library
 * The Stream struct abstracts the differences
 */

type FileStream struct {
	cmd	   *exec.Cmd
	in	   io.ReadCloser
	audio	   *wav.Wav
	blockSize  int
	sampleRate int
	empty      bool
	blockId    int
}

func (f *FileStream) Close() (err error) {
	f.in.Close()
	return f.cmd.Wait()
}

func (f *FileStream) Read() (*Frame, error) {
	block, err := f.audio.ReadSamples(f.blockSize)
	if err != nil {
		return nil, err
	}

	if f.empty {
		f.empty = false
	} else {
		f.blockId++
	}

	frame := NewFrame(block.([]int16), f.blockId, f.sampleRate)

	return &frame, nil
}

func (f *FileStream) Start() (err error) {
	return nil
}


func NewFileStream(filename string, sampleRate, blockSize int) (*FileStream, error) {
	cmd, err := ffmpeg.Cmd(filename, ffmpeg.CONTAINER_WAV, ffmpeg.FMT_INT16, sampleRate)
	if (err != nil) {
		return nil, err
	}

	in, err := ffmpeg.StartStream(cmd)
	if (err != nil) {
		return nil, err
	}

	audio, err := wav.New(in)
	if err != nil {
		return nil, fmt.Errorf("Opening Wav file: %s", err)
	}
	if audio.SampleRate != uint32(sampleRate) {
		return nil, fmt.Errorf("Wav file has different sample rate (%d) to requested rate (%d)", audio.SampleRate, sampleRate)
	}


	stream := FileStream{
		blockSize:  blockSize,
		sampleRate: sampleRate,
		audio:      audio,
		cmd:        cmd,
		in:         in,
		empty:	    true,
		blockId:    0,
	}

	return &stream, nil

}

