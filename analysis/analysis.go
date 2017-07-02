package analysis

import (
	"math"
	"math/cmplx"
	"github.com/mjibson/go-dsp/fft"
	"github.com/mjibson/go-dsp/window"
	"github.com/mjibson/go-dsp/spectral"
	"github.com/snuffpuppet/spectre/pcm"
)

/*
 * Use the PWelch algorithm to determine Spectral Density of the time series data
 */
func PwelchAnalysis(sampleBlock *pcm.Buffer, sampleRate int) (Pxx, freqs []float64) {
	// 'block' contains our data block, get a spectral analysis of this section of the audio
	var opts spectral.PwelchOptions // default values are used
	opts.Noverlap = 512
	opts.NFFT = 1024
	opts.Scale_off = true

	samples := sampleBlock.AsFloat64()

	//seg := spectral.Segment(samples, opts.NFFT, opts.Noverlap)
	//fmt.Printf("Segs: %v\n", seg)

	Pxx, freqs = spectral.Pwelch(samples, float64(sampleRate), &opts)

	if false {
		// Now convert Pxx (Power per unit freq) to dB
		for i, x := range Pxx {
			if x < 1 {
				Pxx[i] = 0
			} else {
				Pxx[i] = 10 * math.Log10(x)
			}
		}
	}

	return
}

/*
 * Use a basic non windowed algorithm to get frequencies and power levels
 */
func BespokeAnalysis(sampleBlock *pcm.Buffer, sampleRate int) (Pxx, freqs []float64) {
	samples := make([]float64, sampleBlock.Size())
	copy(samples, sampleBlock.AsFloat64())

	// construct a slice of complex numbers containing the sample data & imaginary part as 0
	complexSamples := make([]complex128, len(samples))
	for i, v := range samples {
		complexSamples[i] = complex(float64(v), 0.0)
	}

	fftResults := fft.FFT(complexSamples)

	l2 := int(float64(len(fftResults)) / 2.0 + 0.5)  // round to nearest integer
	fftRelevent := fftResults[1:l2]

	freqs = make([]float64, len(fftRelevent))
	Pxx = make([]float64, len(fftRelevent))

	maxFreq := float64(sampleRate) / 2.0
	for i, v := range fftRelevent {
		Pxx[i] = cmplx.Abs(v)
		freqs[i] = float64(i) / float64(l2) * maxFreq
	}

	return

}

/*
 * Use overlapping windows to adjust for spectral leakage when using the FFT
 */
func OverlapAnalysis(sampleBlock *pcm.Buffer, sampleRate int) (Pxx, freqs []float64) {
	// 'block' contains our data block, get a spectral analysis of this section of the audio

	const NFFT = 512
	const NOVERLAP = 384
	const NORMALISING_ENABLED = false 	// disable normalising for the moment as it seems hide strong signals

	wf := window.Hann

	samples := make([]float64, sampleBlock.Size())
	copy(samples, sampleBlock.AsFloat64())

	segs := spectral.Segment(samples, NFFT, NOVERLAP)

	lp := NFFT / 2 + 1

	Pxx = make([]float64, lp)

	for _, x := range segs {
		window.Apply(x, wf)
		pgram := fft.FFTReal(x)

		for i := range Pxx {
			Pxx[i] += cmplx.Abs(pgram[i])
		}
	}

	if NORMALISING_ENABLED {
		w := wf(NFFT)
		var norm float64
		for _, x := range w {
			norm += math.Pow(x, 2)
		}

		for i := range Pxx {
			Pxx[i] /= norm
		}
	}

	freqs = make([]float64, lp)
	coef := float64(sampleRate) / float64(NFFT)
	for i := range freqs {
		freqs[i] = float64(i) * coef
	}

	return
}


