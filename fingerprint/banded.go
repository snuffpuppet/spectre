package fingerprint

import (
	"fmt"
	"crypto/sha1"
	"io"
	"github.com/snuffpuppet/spectre/spectral"
	_ "log"
)

const NUM_BANDS = 6

// Fingerprint info on a block of audio data
type Bandedprint struct {
	key           []byte
	bands	      spectral.Spectra
}

func (b Bandedprint) String() string {
	return b.bands.String()
}
func (b Bandedprint) Fingerprint() []byte {
	if string(b.key) == "" {
		hash := sha1.New()
		for _, v := range b.bands.Freqs {
			io.WriteString(hash, fmt.Sprintf("%e", v))
		}

		b.key = hash.Sum(nil)

	}
	return b.key
}

func NewBandedprint(spectra spectral.Spectra) (*Bandedprint) {
	bands := getBandedCandidates(spectra)
	numBands := 0
	for x := range bands.Freqs {
		if x > 0 {
			numBands++
		}
	}

	if numBands < REQUIRED_NUM_CANDIDATES {
		return nil
	}

	bp := Bandedprint{
		bands:	bands,
	}

	return &bp
}

// Use a basic frequency banding method for classifying frequencies and choosing candidates for the fingerprint
// Return the strongest frequency in each of four bands ordered by strength
func getBandedCandidates(spectra spectral.Spectra) (s spectral.Spectra) {
	highScores := make([]float64, NUM_BANDS)
	highPoints := make([]float64, NUM_BANDS)

	// find strongest frequency in each band
	//log.Printf("Banded Spectra: len(Pxx), len(Freqs) = %d, %d\n", len(spectra.Pxx), len(spectra.Freqs))
	for i, v := range spectra.Pxx {
		//log.Printf("Banded Spectra: i=%d, len(spectra.Freqs=%d", i, len(spectra.Freqs))
		fb := freqBand(spectra.Freqs[i])
		if v > highScores[fb] {
			highPoints[fb] = spectra.Freqs[i]
			highScores[fb] = v
		}

	}

	return spectral.NewSpectra(highPoints, highScores)
}

func meanStrength(c candidates) (mean float64) {
	// Now get the mean signal strength
	mean = 0.0
	for _, v := range c {
		mean += v.Pxx
	}
	mean /= float64(len(c))

	return
}

func freqBand(f float64) int {
	//uLimit := 11025.0 / 2.0
	uLimit := UPPER_FREQ_CUTOFF
	a := f - LOWER_FREQ_CUTOFF
	b := uLimit - LOWER_FREQ_CUTOFF

	x := int(a / b * (NUM_BANDS-1) + 0.5)

	//log.Printf("%.2f => Band %d (a=%.2f, b=%.2f)\n", f, x, a, b)
	return x
}

