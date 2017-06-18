package audioFingerprint

import (
	"crypto/sha1"
	"github.com/mjibson/go-dsp/spectral"
	"fmt"
	"sort"
	"io"
	"github.com/snuffpuppet/spectre/audioBuffer"
	"github.com/mjibson/go-dsp/fft"
	"math/cmplx"
)

const PWELCH_DATA_POINTS = 1024
const NUM_CANDIDATES = 3 		// required number of frequency candidates for a fingerprint entry
const LOWER_FREQ_CUTOFF = 1500.0
const LOWER_POWER_CUTOFF = 0.5
const TIME_DELTA_THRESHOLD = 0.1	// reqeuired minimum time diff between freq matches to be considered a hit

const (
	_ = iota
	SA_PWELCH = iota
	SA_BESPOKE = iota
)


/*
 * Spectral Analysis
 */
type candidate struct { Freq float64
			Pxx float64
}

type ByPxx []candidate
func (a ByPxx) Len() int           { return len(a) }
func (a ByPxx) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByPxx) Less(i, j int) bool { return a[i].Pxx < a[j].Pxx }


type Mapping struct {
	Filename    string
	Timestamp   float64
}


func pwelchAnalysis(sampleBlock *audioBuffer.Block, sampleRate int) (Pxx, freqs []float64, err error) {
	// 'block' contains our data block, get a spectral analysis of this section of the audio
	var opts spectral.PwelchOptions // default values are used
	opts.Noverlap = 0
	opts.NFFT = PWELCH_DATA_POINTS
	opts.Scale_off = true

	samples, err := sampleBlock.Float64Data()
	if (err != nil) {
		return nil, nil, err
	}

	Pxx, freqs = spectral.Pwelch(samples, float64(sampleRate), &opts)

	return
}

func bespokeAnalysis(sampleBlock *audioBuffer.Block, sampleRate int) (Pxx, freqs []float64, err error) {
	// 'block' contains our data block, get a spectral analysis of this section of the audio

	samples, err := sampleBlock.Float64Data()
	if (err != nil) {
		return nil, nil, err
	}

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

func getCandidates(freqs, Pxx []float64) ([]candidate, error) {
	candidates := make([]candidate, 0)

	// select only those stronger than the power threshold and higher than the frequency threshold
	for i, v := range Pxx {
		if v > LOWER_POWER_CUTOFF && freqs[i] > LOWER_FREQ_CUTOFF {
			candidates = append(candidates, candidate{Freq: freqs[i], Pxx: v})
		}
	}

	// Sort the list in descending order
	sort.Sort(sort.Reverse(ByPxx(candidates)))

	var topCandidates []candidate
	if len(candidates) < NUM_CANDIDATES {
		topCandidates = candidates
	} else {
		topCandidates = candidates[:NUM_CANDIDATES]
	}
	return topCandidates, nil
}

func PrintCandidates(blockId int, blockTime float64, candidates []candidate) {
	f, p, s := "", "", ""
	for _, v := range candidates {
		f += fmt.Sprintf("%9.2f", v.Freq)
		p += fmt.Sprintf("%9.4f", v.Pxx)
		s += fmt.Sprintf("%9.2f (%.2f)\t", v.Freq, v.Pxx)
	}
	//fmt.Printf("[%4d:%6.2f] %s\n              %s\n", sampleBlock.Id, sampleBlock.Timestamp, f, p)
	fmt.Printf("\t[%4d:%6.2f] %s\n", blockId, blockTime, s)
}



func New(sampleBlock *audioBuffer.Block, sampleRate int, optSpectralAnalyser int) ([]byte, []candidate, error) {
	var err error
	var Pxx, freqs []float64
	switch optSpectralAnalyser {
	case SA_PWELCH:
		Pxx, freqs, err = pwelchAnalysis(sampleBlock, sampleRate)
	case SA_BESPOKE:
		Pxx, freqs, err = bespokeAnalysis(sampleBlock, sampleRate)
	}
	if (err != nil) {
		return nil, nil, err
	}

	candidates, err := getCandidates(freqs, Pxx)
	if (err != nil) {
		return nil, nil, err
	}

	if len(candidates) < NUM_CANDIDATES {
		return nil, nil, nil		// no valid candidates
	}

	// Now copy over the ones that we are interested in and populate the hash string
	hash := sha1.New()
	for _, v := range candidates {
		io.WriteString(hash, fmt.Sprintf("%e", v.Freq))
	}
	// Add in the time difference between the samples


	return hash.Sum(nil), candidates, nil
}


