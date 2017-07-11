package fingerprint

import (
	"fmt"
	"sort"
	"crypto/sha1"
	"io"
)

type candidate struct { 
	Freq float64
	Pxx float64
}

type candidates []candidate
func (c candidates) String() string {
	var s string
	for _, v := range c {
		s += fmt.Sprintf("%9.2f (%.2f)\t", v.Freq, v.Pxx)
	}
	return s
}

type ByPxx []candidate
func (a ByPxx) Len() int           { return len(a) }
func (a ByPxx) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByPxx) Less(i, j int) bool { return a[i].Pxx < a[j].Pxx }

type ByFreq []candidate
func (a ByFreq) Len() int           { return len(a) }
func (a ByFreq) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByFreq) Less(i, j int) bool { return a[i].Freq < a[j].Freq }

// Fingerprint info on a block of audio data
type Bandedprint struct {
	key           []byte
	candidates    candidates
}
func (b Bandedprint) String() string {
	return b.candidates.String()
}
func (b Bandedprint) Fingerprint() []byte {
	return b.key
}

// Use a basic frequency banding method for classifying frequencies and choosing candidates for the fingerprint
// Return the strongest frequency in each of four bands ordered by strength
func getBandedCandidates(Pxx, freqs []float64) (candidates) {
	const LOWER_FREQ_CUTOFF = 318.0
	const UPPER_FREQ_CUTOFF = 2000.0
	const NUM_BANDS = 6

	candidates := make([]candidate, 0)
	highScores := make(map[int]float64)
	highPoints := make(map[int]float64)

	var freqBand = func(f float64) int {
		//uLimit := 11025.0 / 2.0
		uLimit := UPPER_FREQ_CUTOFF
		a := f - LOWER_FREQ_CUTOFF
		b := uLimit - LOWER_FREQ_CUTOFF

		x := int(a / b * NUM_BANDS + 0.5)

		//fmt.Printf("%.2f => Band %d (a=%.2f, b=%.2f)\n", f, x, a, b)
		return x
	}

	// select only those stronger than the power threshold and higher than the frequency threshold
	for i, v := range Pxx {
		fb := freqBand(freqs[i])
		if v > highScores[fb] {
			highPoints[fb] = freqs[i]
			highScores[fb] = v
		}

	}

	// Now get the mean signal strength
	mean := 0.0
	for _, v := range highScores {
		mean += v
	}
	mean /= float64(len(highScores))

	for k, v := range highScores {
		if v >= mean {
			candidates = append(candidates, candidate{Freq: fuzzyFreq(highPoints[k]), Pxx: v})
		}
	}

	// Sort by Frequency to adjust for any minor signal strength variance between them
	sort.Sort(sort.Reverse(ByFreq(candidates)))

	return candidates
}

func NewBandedprint(Pxx, freqs []float64) (*Bandedprint) {
	const REQUIRED_NUM_CANDIDATES = 2

	candidates := getBandedCandidates(Pxx, freqs)
	if len(candidates) < REQUIRED_NUM_CANDIDATES {
		return nil        // no valid candidates
	}

	// Now copy over the ones that we are interested in and populate the hash string
	hash := sha1.New()
	for _, v := range candidates {
		io.WriteString(hash, fmt.Sprintf("%e", v.Freq))
	}

	key := hash.Sum(nil)

	bp := Bandedprint{
		key: 		key,
		candidates:	candidates,
	}

	return &bp
}