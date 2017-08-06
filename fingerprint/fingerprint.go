package fingerprint

import (
	"github.com/snuffpuppet/spectre/spectral"
	"fmt"
	"io"
	"crypto/sha1"
)

/*
 * const SAMPLE_RATE = 11025
 * const BLOCK_SIZE  = 2048
 * const NFFT 	     = 512
 * const NOVERLAP    = 384
 * const DB_SCALING  = true
 * gets us 43/178/221 on Brad
 */
const SAMPLE_RATE = 11025
const BLOCK_SIZE  = 2048
//const SAMPLE_RATE = 44100
//const BLOCK_SIZE  = 4096
const NFFT 	  = 1024
const NOVERLAP    = 512
const DB_SCALING = true			// Scale the amplitude output to dB

const BLOCKS_PER_SECOND = SAMPLE_RATE / BLOCK_SIZE

//const REQUIRED_CANDIDATES = 4 	// required number of frequency candidates for a fingerprint entry
const LOWER_FREQ_CUTOFF = 1000.0	// Lowest frequency acceptable for matching
const UPPER_FREQ_CUTOFF = 2000.0	// Highest frequency acceptable for matching

//const LOWER_FREQ_CUTOFF = 0.0	// Lowest frequency acceptable for matching
//const UPPER_FREQ_CUTOFF = SAMPLE_RATE / 2.0	// Highest frequency acceptable for matching

const TIME_DELTA_THRESHOLD = 0.5	// required minimum time diff between freq matches to be considered a hit

const FILE_SILENCE_THRESHOLD = 30.0
const MIC_SILENCE_THRESHOLD = 30.0

const REQUIRED_NUM_CANDIDATES = 2

type FingerprintStringer interface {
	Fingerprint() []float64
	String()      string
}

// Apply an approximation to the frequency to help with inacuracies with matching later
func fuzzyFreq(f float64) float64 {
	return float64(int(f*10 + 0.5))/10
}

func Hash(fp []float64) []byte {
	hash := sha1.New()
	for _, v := range fp {
		io.WriteString(hash, fmt.Sprintf("%e", fuzzyFreq(v)))
	}

	return hash.Sum(nil)
}

func Generate(analyser spectral.Analyser, samples []float64, silenceThreshold float64) (FingerprintStringer) {
	//s := ""

	spectra := analyser(samples, SAMPLE_RATE, NFFT, NOVERLAP, DB_SCALING)
	//log.Printf("Raw Samples:\n%v\n%v\n\n", spectra.Freqs, spectra.Pxx)
	//s = fmt.Sprintf("%s -> samples=%d", s, len(spectra.Freqs))

	spectra = spectra.Filter(
		func(freq, pwr float64) bool {
//			return freq >= LOWER_FREQ_CUTOFF && freq <= UPPER_FREQ_CUTOFF && pwr > silenceThreshold
			return freq >= 30 && freq <= 5500 && pwr > silenceThreshold
		})

	//spectra = spectra.Maxima()

	//spectra = spectra.HighPass()

	//log.Println(s)

	//fp := NewChromaprint(spectra)
	return NewBandedprint(SAMPLE_RATE, spectra)
}
