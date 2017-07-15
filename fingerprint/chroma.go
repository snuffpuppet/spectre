package fingerprint

import (
	"fmt"
	"log"
	"io"
	"crypto/sha1"
	"math"
)

// Chroma based Fingerprint info on a block of audio data
type Chromaprint struct {
	Key           []byte
	//Timestamp     float64
	//Candidates    candidates
	Transcription Transcription
}

// To be a Fingerprinter, we must satisfy the interface
func (c Chromaprint) Fingerprint() []byte  { return c.Key }
//func (c Chromaprint) Timestamp()   float64 { return c.Timestamp }

// To anable debugging we satisfy the Stringer interface
func (c Chromaprint) String()    string  { return c.Transcription.String() }

// For the Chroma identification method of matching:
// ref: http://musicweb.ucsd.edu/~sdubnov/CATbox/Reader/ThumbnailingMM05.pdf
const (
	A_NOTE = iota
	AS_NOTE = iota
	B_NOTE = iota
	C_NOTE = iota
	CS_NOTE = iota
	D_NOTE = iota
	DS_NOTE = iota
	E_NOTE = iota
	F_NOTE = iota
	FS_NOTE = iota
	G_NOTE = iota
	GS_NOTE = iota
	MAX_NOTE = iota
)

type note int

func (n note) String() (s string) {
	switch int(n) {
	case A_NOTE:
		s = "A"
	case AS_NOTE:
		s = "A#"
	case B_NOTE:
		s = "B"
	case C_NOTE:
		s = "C"
	case CS_NOTE:
		s = "C#"
	case D_NOTE:
		s = "D"
	case DS_NOTE:
		s = "D#"
	case E_NOTE:
		s = "E"
	case F_NOTE:
		s = "F"
	case FS_NOTE:
		s = "F#"
	case G_NOTE:
		s = "G"
	case GS_NOTE:
		s = "G#"
	default:
		log.Panicf("Unrecognised note enumertation %d", n)

	}

	return
}

// Logn(s^(1/12)) - used for Equal Tempered scale measurement in equalTempSteps function
const LOGNA = 0.05776226504666185940

// calculate the number of note semitones that the frequency is away from the 440Hz base tone
// Using the Equal Tempered Scale with A4 = 440Hz
//ref: http://www.phy.mtu.edu/~suits/notefreqs.html
func noteSteps(freq float64) float64 {
	return math.Log(freq/440.0)/ LOGNA
}

// find out to which note this frequency corresponds. Returns a number between 0 and 11
func freqNote(freq float64) int {
	n := int(noteSteps(freq) + 0.5) % MAX_NOTE
	if n < 0 {
		n += MAX_NOTE
	}

	return n
}

// A bucket of frequencies that make up a musical note
type Chroma struct {
	Note     note
	Freq     float64
	Strength float64
}

func (c Chroma) String() string {
	return fmt.Sprintf("[%s] %6.1f", c.Note, c.Freq)
}

type Transcription []Chroma

func (t Transcription) String() string {
	s := ""
	for _, v := range t {
		s += fmt.Sprintf("%s ", v)
	}

	return s
}

func (t Transcription) meanStrength() (m float64) {
	m = 0.0
	for _, v := range t {
		m += v.Strength
	}
	m /= float64(len(t))

	return
}

// Convert the frequency/power data into buckets of musical notes based on strength of signal
func transcribe(freqs, Pxx []float64) (t Transcription) {
	chromaCount := 0
	t = make([]Chroma, MAX_NOTE)
	for i, v := range freqs {
		n := freqNote(v)
		if Pxx[i] > t[n].Strength {
			//log.Printf("*** Set %d(%s) -> %.1f(%.1f)\n", n, note(n), v, Pxx[i])
			t[n].Note = note(n)
			t[n].Freq = fuzzyFreq(v)
			t[n].Strength = Pxx[i]
			chromaCount++
		} else {
				//fmt.Printf("*** Rejected: %f(%.2f)\n", fuzzyFreq(v), Pxx[i])
		}

	}

	if chromaCount == 0 {
		t = nil
	}

	return
}

// Generate a fingerprint based on the musical transcription of the frequencies in the audio frame
func audioKey(t Transcription) (key []byte) {
	// The Powerkey method uses a scaled strength of each of the 12 notes to generate the key
	// The frequency hash method uses the strongest frequencies for each of the notes to create a hash
	optPowerKey := false

	if t == nil {
		return nil
	}

	key = make([]byte, len(t))

	maxPxx := 0.0
	if optPowerKey {
		for _, v := range t {
			if v.Strength > maxPxx {
				maxPxx = v.Strength
			}
		}
		for i, v := range t {
			key[i] = byte(int(v.Strength/maxPxx * 8.0 + 0.5))
		}
	} else {

		hash := sha1.New()

		for _, v := range t {
			io.WriteString(hash, fmt.Sprintf("%e", v.Freq))
		}

		key = hash.Sum(nil)
	}
	return
}

func highPassFilter(t Transcription) (Transcription) {
	if t == nil {
		return t
	}

	mean := t.meanStrength()
	//log.Printf("highPassFIlter: mean = %f\n", mean)

	f := make([]Chroma, MAX_NOTE)

	for i, v := range t {
		f[i] = t[i]
		if v.Strength < mean {
			f[i].Strength, f[i].Freq = 0, 0
		}
	}

	return f
}

func NewChromaprint(Pxx, freqs []float64) (*Chromaprint) {
	transcription := transcribe(freqs, Pxx)
	//log.Printf("NewChromaPrint1: %s\n", transcription)

	transcription = highPassFilter(transcription)
	//log.Printf("NewChromaPrint2: %s\n", transcription)

	key := audioKey(transcription)
	//log.Printf("fp key: %s\n", key)

	if key == nil {
		return nil
	}

	cp := Chromaprint{
		Key: key,
		Transcription: transcription,
	}

	return &cp
}
