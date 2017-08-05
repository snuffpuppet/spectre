package fingerprint

import (
	"github.com/snuffpuppet/spectre/spectral"
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

type Fingerprinter interface {
	Fingerprint() []byte
}

// Apply an approximation to the frequency to help with inacuracies with matching later
func fuzzyFreq(f float64) float64 {
	return float64(int(f*10 + 0.5))/10
	//fuzzyFreq -= fuzzyFreq%2
}

func Generate(analyser spectral.Analyser, samples []float64, silenceThreshold float64) (Fingerprinter) {
	//s := ""

	spectra := analyser(samples, SAMPLE_RATE, NFFT, NOVERLAP, DB_SCALING)
	//log.Printf("Raw Samples:\n%v\n%v\n\n", spectra.Freqs, spectra.Pxx)
	//s = fmt.Sprintf("%s -> samples=%d", s, len(spectra.Freqs))

	spectra = spectra.Filter(
		func(freq, pwr float64) bool {
			return freq >= LOWER_FREQ_CUTOFF && freq <= UPPER_FREQ_CUTOFF && pwr > silenceThreshold
		})

	//s = fmt.Sprintf("%s -> audible=%d", s, len(spectra.Freqs))

	spectra = spectra.Maxima()
	//s = fmt.Sprintf("%s -> maxima=%d", s, len(spectra.Freqs))

	//spectra = spectra.HighPass()
	//s = fmt.Sprintf("%s -> highPass=%d", s, len(spectra.Freqs))

	//log.Println(s)

	//fp := NewChromaprint(spectra)
	fp := NewBandedprint(spectra)

	if fp == nil {
		return nil
	}

	return fp
}

/*
import (
	_ "crypto/sha1"
	"fmt"
	"sort"
	"github.com/snuffpuppet/spectre/pcm"
	"github.com/snuffpuppet/spectre/analysis"
	"math"
	"log"
	"crypto/sha1"
	"io"
)

type candidate struct { Freq float64
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
type Fingerprint struct {
	Key           []byte
	Candidates    candidates
}



// return te strongest (REQUIRED_CANDIDATES) frequencies in the frequency data
func getTopCandidates(freqs, Pxx []float64) (candidates) {
	candidates := make([]candidate, 0)

	// select only those stronger than the power threshold and higher than the frequency threshold
	for i, v := range Pxx {

		if v > LOWER_POWER_CUTOFF && freqs[i] > LOWER_FREQ_CUTOFF {
			candidates = append(candidates, candidate{Freq: fuzzyFreq(freqs[i]), Pxx: v})
		}
	}

	// Sort the list in descending order
	sort.Sort(sort.Reverse(ByPxx(candidates)))

	var topCandidates []candidate
	if len(candidates) < REQUIRED_CANDIDATES {
		return nil
	}

	// Get the strongest signals
	topCandidates = candidates[:REQUIRED_CANDIDATES]

	// Sort by Frequency to adjust for any minor signal strength variance between them
	sort.Sort(sort.Reverse(ByFreq(topCandidates)))

	return topCandidates
}

// Use a basic frequency banding method for classifying frequencies and choosing candidates for the fingerprint
// Return the strongest frequency in each of four bands ordered by strength
func getBandedCandidates(freqs, Pxx []float64) (candidates) {

	candidates := make([]candidate, 0)
	highScores := make(map[int]float64)
	highPoints := make(map[int]float64)

	var freqBand = func(f float64) int {
		uLimit := 11025.0 / 2.0
		a := f - LOWER_FREQ_CUTOFF
		b := uLimit - LOWER_FREQ_CUTOFF

		x := int(a / b * 4 + 0.5)

		//fmt.Printf("%.2f => Band %d (a=%.2f, b=%.2f)\n", f, x, a, b)
		return x
	}

	// select only those stronger than the power threshold and higher than the frequency threshold
	for i, v := range Pxx {
		if v > LOWER_POWER_CUTOFF && freqs[i] > LOWER_FREQ_CUTOFF {
			fb := freqBand(freqs[i])
			if v > highScores[fb] {
				highPoints[fb] = freqs[i]
				highScores[fb] = v
			}
		}
	}

	for k, v := range highPoints {
		candidates = append(candidates, candidate{Freq: fuzzyFreq(v), Pxx: highScores[k]})
	}

	// Sort by Frequency to adjust for any minor signal strength variance between them
	sort.Sort(sort.Reverse(ByFreq(candidates)))

	return candidates
}

/*
func PrintCandidates(blockId int, blockTime float64, candidates []candidate) {
	s := ""
	for _, v := range candidates {
		//f += fmt.Sprintf("%9.2f", v.Freq)
		//p += fmt.Sprintf("%9.4f", v.Pxx)
		s += fmt.Sprintf("%9.2f (%.2f)\t", v.Freq, v.Pxx)
	}
	//fmt.Printf("[%4d:%6.2f] %s\n              %s\n", sampleBlock.Id, sampleBlock.Timestamp, f, p)
	fmt.Printf("\t[%4d:%6.2f] %s\n", blockId, blockTime, s)
}
*//*

// log some frequency distribution data for the given spectrum
func logSamples(verbose bool, freqs, Pxx []float64) {
	var top, bottom, avg, topf, bottomf float64
	var count int

	if !verbose {
		return
	}

	bottom = -1.0
	for i, x := range Pxx {
		if x > LOWER_POWER_CUTOFF && freqs[i] > LOWER_FREQ_CUTOFF {
			if x > top {
				top = x
				topf = freqs[i]
			}
			if x < bottom {
				bottom = x
				bottomf = freqs[i]
			}
			avg += x
			count++
		}
	}

	if count > 0 {
		log.Printf("#S:%3d T: [%7.1f] %7.1f\tB: [%7.1f] %7.1f\tA: %7.1f", count, topf, top, bottomf, bottom, avg / float64(len(Pxx)))
	}
}


func New(sampleBlock *pcm.Buffer, sampleRate int, spectral analysis.SpectralAnalyser, optVerbose bool) (*Fingerprint) {
	var Pxx, freqs []float64
	switch optSpectralAnalyser {
	case SA_PWELCH:
		Pxx, freqs = analysis.PwelchAnalysis(sampleBlock, sampleRate)
	case SA_BESPOKE:
		Pxx, freqs = analysis.OverlapAnalysis(sampleBlock, sampleRate)
	default:
		log.Panicf("Unrecognised spectral analyser %d\n", optSpectralAnalyser)
	}

	optMethod :=  "transcribe" //"freqbands" // "transcribe", "topfreq"

	//logSamples(optVerbose, freqs, Pxx)

	var key []byte
	var fp Fingerprint

	switch (optMethod) {
	case "transcribe":
		transcription := transcribe(freqs, Pxx)
		//log.Printf("fp transscription: %s\n", transcription)

		key := audioKey(transcription)
		//log.Printf("fp key: %s\n", key)

		if key == nil {
			return nil
		}
		fp = Fingerprint{
			Key: key,
			Timestamp: sampleBlock.Timestamp,
			Candidates: nil,
			Transcription: transcription,
		}
	case "topfreqs":
		candidates := getTopCandidates(freqs, Pxx)

		if len(candidates) < REQUIRED_CANDIDATES {
			return nil        // no valid candidates
		}

		// Now copy over the ones that we are interested in and populate the hash string
		hash := sha1.New()
		for _, v := range candidates {
			io.WriteString(hash, fmt.Sprintf("%e", v.Freq))
		}

		key = hash.Sum(nil)

		fp = Fingerprint{
			Key: key,
			Timestamp: sampleBlock.Timestamp,
			Candidates: candidates,
			Transcription: nil,
		}
	case "freqbands":
		candidates := getBandedCandidates(freqs, Pxx)

		if len(candidates) < REQUIRED_CANDIDATES {
			return nil        // no valid candidates
		}

		// Now copy over the ones that we are interested in and populate the hash string
		hash := sha1.New()
		for _, v := range candidates {
			io.WriteString(hash, fmt.Sprintf("%e", v.Freq))
		}

		key = hash.Sum(nil)

		fp = Fingerprint{
			Key: key,
			Timestamp: sampleBlock.Timestamp,
			Candidates: candidates,
			Transcription: nil,
		}
	default:
		log.Panicf("Fingerprint: Unknown key generaion method: %s", optMethod)
	}


	return &fp
}


*/