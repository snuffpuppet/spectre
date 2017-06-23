package pcmstream

import (
	"os/exec"
	"github.com/mjibson/go-dsp/wav"
	"io"
	"github.com/gordonklaus/portaudio"
	"github.com/snuffpuppet/spectre/pcmframe"
	"fmt"
	"strconv"
)

/*
 * audioStream
 */
type starter func() error
type reader  func() (*pcmframe.Block, error)
type closer  func() error

type Stream struct {
	Filename   string
	Buffer     *pcmframe.Block
	blockSize  int
	sampleRate int
	start	   starter	// function with closure to start the stream running (if needed)
	read	   reader       // function with closure to read a new data block from this stream
	close      closer       // function with closure to close the stream and cleanup
}

func (f *Stream) Close() (err error) {
	return f.close()
}

func (f *Stream) ReadBlock() (buf *pcmframe.Block, err error) {
	return f.read()
}

func (f *Stream) Start() (err error) {
	return f.start()
}

func ffmpegCmd(filename, containerType, pcmDataType string, sampleRate int) (*exec.Cmd, error) {
	// containerType: "raw"|"wav", pcmFormat: "int16"|"float32"
	// containerType describes if we want a raw output or a wav container
	// pcmDataType describes the internal format of the data we want e.g. float32 / signed int 16 etc
	// codec indicates (to ffmpeg) a raw format and which (raw) codec to use

	codec := ""    // indicates (to ffmpeg) how to encode the pcm data
	format := ""   // indicates (to ffmpeg) how to format the file (wav or raw - with raw format 's16le' etc)
	ffmpegDataType := ""  // internal data type for ffmpeg to use for PCM data

	switch pcmDataType {
	case "int16":
		ffmpegDataType = "s16le"
	case "float32":
		ffmpegDataType = "f32le"
	default:
		return nil, fmt.Errorf("ffmpegCmd: Unrecognised PCM format: %s", pcmDataType)
	}
	codec = "pcm_" + ffmpegDataType

	switch containerType {
	case "raw":
		// if raw then we need to set the format to the internal pcm data type
		format = ffmpegDataType
	case "wav":
		format = "wav"
	default:
		return nil, fmt.Errorf("ffmpegCmd: Unrecognised container type: %s", containerType)
	}

	//duration := "20"
	channels := "1"
	bitRate := "192k"

	args := make([]string, 0, 15)
	//introArgs := []string{"-t", duration}
	inputArgs := []string{"-i", filename}
	codecArgs := []string{"-acodec", codec}
	formatArgs := []string{"-f", format}
	bitRateArgs := []string{"-ab", bitRate}
	sampleRateArgs := []string{"-ar", strconv.Itoa(sampleRate)}
	channelArgs := []string{"-ac", channels}
	pipeArgs := []string{"pipe:1"}

	args = append(args, inputArgs...)
	args = append(args, formatArgs...)
	if containerType != "wav" {  // for wav containers, use default (int16) codec -otherwise trouble
		args = append(args, codecArgs...)
	}
	args = append(args, bitRateArgs...)
	args = append(args, sampleRateArgs...)
	args = append(args, channelArgs...)
	args = append(args, pipeArgs...)

	cmd := exec.Command("ffmpeg", args...)

	//log.Printf("ffmpeg %s", args)

	return cmd, nil
}

func ffmpegStartStream(cmd *exec.Cmd) (io.ReadCloser, error) {
	audio, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return audio, nil
}

func NewBufferedWav(filename string, buffer pcmframe.Buffer, sampleRate int) (*Stream, error) {
	block := pcmframe.NewBlock(buffer, sampleRate)

	pcmFormat := block.DataFormat()

	cmd, err := ffmpegCmd(filename, "wav", pcmFormat, sampleRate)
	if (err != nil) {
		return nil, err
	}

	audio, err := ffmpegStartStream(cmd)
	if (err != nil) {
		return nil, err
	}

	pcm, err := wav.New(audio)
	if err != nil {
		return nil, fmt.Errorf("Opening Wav file: %s", err)
	}
	if pcm.SampleRate != uint32(sampleRate) {
		return nil, fmt.Errorf("Wav file has different sample rate (%d) to requested rate (%d)", pcm.SampleRate, sampleRate)
	}

	startFn := func() error { return nil }
	readFn  := func() (*pcmframe.Block, error) {
		samples, err := pcm.ReadSamples(block.Size())
		if err != nil {
			return nil, err
		}
		//log.Println(samples) // TESTING
		block.SetBuffer(samples)  // Copy slice over the top since they are the same data types
		block.UpdateReadCount(block.Size())
		return block, nil
	}
	closeFn := func() error {
		audio.Close()
		return cmd.Wait()
	}

	stream := Stream{
		Filename: filename,
		Buffer: block,
		blockSize: block.Size(),
		sampleRate: int(pcm.SampleRate),
		start: startFn,
		read:  readFn,
		close: closeFn,
	}

	return &stream, nil

}

func NewMicrophone(blockSize int, sampleRate int) (*Stream, error) {
	portaudio.Initialize()

	buffer := make([]float32, blockSize)
	block := pcmframe.NewBlock(buffer, sampleRate)

	paStream, err := portaudio.OpenDefaultStream(1, 0, float64(sampleRate), blockSize, buffer)
	if err != nil {
		return nil, err
	}

	closeFn := func() error {
		paStream.Close()
		return portaudio.Terminate()
	}

	readFn := func() (*pcmframe.Block, error) {
		err := paStream.Read()
		if err != nil {
			return block, err
		}
		block.UpdateReadCount(block.Size())

		return block, err

	}

	startFn := func() (error) {
		return paStream.Start()
	}

	stream := Stream{
		Filename:   "",
		Buffer:     block,
		blockSize:  block.Size(),
		sampleRate: sampleRate,
		start:      startFn,
		read:       readFn,
		close:      closeFn,
	}

	return &stream, nil
}
