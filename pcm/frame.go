package pcm

type Frame struct {
	data 		[]int16
	timestamp	float64
	blockId		int
}

func NewFrame(data []int16, blockId, sampleRate int) Frame {
	return Frame {
		data: data,
		blockId: blockId,
		timestamp: float64(blockId * len(data)) / float64(sampleRate),
	}
}

func (f Frame) AsFloat64() (f64 []float64) {
	f64 = make([]float64, len(f.data))
	for i, x := range f.data {
		f64[i] = float64(x)
	}

	return
}

func (f Frame) Timestamp() float64 {
	return f.timestamp
}

func (f Frame) BlockId() int {
	return f.blockId
}

func (f Frame) Data() []int16 {
	return f.data
}
