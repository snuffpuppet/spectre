package ffmpeg

import (
	"os/exec"
	"fmt"
	"strconv"
	"io"
)

const (
	FMT_FLOAT32   = "f32"
	FMT_INT16     = "s16"
	CONTAINER_RAW = "raw"
	CONTAINER_WAV = "wav"
)

func Cmd(filename, containerType, pcmDataType string, sampleRate int) (*exec.Cmd, error) {
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
	case CONTAINER_RAW:
		// if raw then we need to set the format to the internal pcm data type
		format = ffmpegDataType
	case CONTAINER_WAV:
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

func StartStream(cmd *exec.Cmd) (io.ReadCloser, error) {
	audio, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return audio, nil
}

