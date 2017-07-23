package pcm

import "github.com/gordonklaus/portaudio"

type MicStream struct {
	blockSize  int
	sampleRate int
	mic	   *portaudio.Stream
	buf	   []int16
	blockId	   int
	empty	   bool
}

func (m *MicStream) Close() (err error) {
	m.mic.Close()
	return portaudio.Terminate()
}

func (m *MicStream) Read() (*Frame, error) {
	err := m.mic.Read()
	if err != nil {
		return nil, err
	}

	if m.empty {
		m.empty = false
	} else {
		m.blockId++
	}

	frame := NewFrame(m.buf, m.blockId, m.sampleRate)

	return &frame, nil
}

func (m *MicStream) Start() (err error) {
	return m.mic.Start()
}

func NewMicStream(sampleRate, blockSize int) (*MicStream, error) {
	portaudio.Initialize()

	buf := make([]int16, blockSize)

	mic, err := portaudio.OpenDefaultStream(1, 0, float64(sampleRate), blockSize, buf)

	if err != nil {
		return nil, err
	}

	stream := MicStream{
		buf:        buf,
		blockSize:  blockSize,
		sampleRate: sampleRate,
		mic:        mic,
	}

	return &stream, nil
}

