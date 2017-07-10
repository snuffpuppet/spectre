
package pcm
/*
import (
	"os/exec"
	"github.com/mjibson/go-dsp/wav"
	"io"
	"github.com/gordonklaus/portaudio"
	"fmt"
	"strconv"
	//"log"
)

/*
 * audioStream:
 * Provide abstraction over an audio stream source.
 * File streams are provided via ffmpeg decoding and microphone streams are provided through the portaudio library
 * The Stream struct abstracts the differences
 */

/*
type starter func() error
type reader  func() (*Buffer, error)
type closer  func() error

type Stream struct {
	Filename   string
	Buffer     *Buffer
	blockSize  int
	sampleRate int
	start	   starter	// function with closure to start the stream running (if needed)
	read	   reader       // function with closure to read a new data block from this stream
	close      closer       // function with closure to close the stream and cleanup
}

func (f *Stream) Close() (err error) {
	return f.close()
}

func (f *Stream) ReadFrame() (buf *Buffer, err error) {
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
	case FMT_INT16:
		ffmpegDataType = "s16le"
	case FMT_FLOAT32:
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

func NewWavStream(filename string, sampleRate int) (*Stream, error) {
	buf := NewIntBuffer(sampleRate)

	pcmFormat := buf.DataFormat()

	cmd, err := ffmpegCmd(filename, "wav", pcmFormat, sampleRate)
	if (err != nil) {
		return nil, err
	}

	audio, err := ffmpegStartStream(cmd)
	if (err != nil) {
		return nil, err
	}

	wstream, err := wav.New(audio)
	if err != nil {
		return nil, fmt.Errorf("Opening Wav file: %s", err)
	}
	if wstream.SampleRate != uint32(sampleRate) {
		return nil, fmt.Errorf("Wav file has different sample rate (%d) to requested rate (%d)", wstream.SampleRate, sampleRate)
	}

	startFn := func() error { return nil }
	readFn  := func() (*Buffer, error) {
		samples, err := wstream.ReadSamples(buf.Size())
		if err != nil {
			return nil, err
		}
		/*
		// first time around check to see if frame formats are compatible
		if (buf.empty) {
			if frameFormat(buf.frame) != frameFormat(samples) {
				log.Panicf("Incompatible frame formats buffer(%T), samples(%T)\n", buf.frame, samples)
			}
		}
		*//*
		buf.SetFrame(samples)

		return buf, nil
	}
	closeFn := func() error {
		audio.Close()
		return cmd.Wait()
	}

	stream := Stream{
		Filename: filename,
		Buffer: buf,
		blockSize: buf.Size(),
		sampleRate: int(wstream.SampleRate),
		start: startFn,
		read:  readFn,
		close: closeFn,
	}

	return &stream, nil

}

func NewMicStream(sampleRate int) (*Stream, error) {
	portaudio.Initialize()

	buf := NewIntBuffer(sampleRate)

	paStream, err := portaudio.OpenDefaultStream(1, 0, float64(sampleRate), buf.Size(), buf.Frame())
	if err != nil {
		return nil, err
	}

	closeFn := func() error {
		paStream.Close()
		return portaudio.Terminate()
	}

	readFn := func() (*Buffer, error) {
		err := paStream.Read()
		if err != nil {
			return buf, err
		}
		buf.UpdateReadCount(buf.Size())

		return buf, err

	}

	startFn := func() (error) {
		return paStream.Start()
	}

	stream := Stream{
		Filename:   "",
		Buffer:     buf,
		blockSize:  buf.Size(),
		sampleRate: sampleRate,
		start:      startFn,
		read:       readFn,
		close:      closeFn,
	}

	return &stream, nil
}


*/