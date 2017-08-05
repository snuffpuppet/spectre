package spectral

import (
	"fmt"
	"sort"
)

type Spectra struct {
	Pxx	[]float64
	Freqs	[]float64
}

// scan the spectral analysis for local maxima defined by:
// if m is a local maxima (at position i in both Freqs and Pxx)then
//   Pxx[i-1], Pxx[i+1] < Pxx[i] and
//   Pxx[i-2] < Pxx[i-1] and
//   Pxx[i+2] < Pxx[i+1]
func (s Spectra) Maxima() Spectra {
	freqs := make([]float64, 0, len(s.Freqs))
	pxx := make([]float64, 0, len(s.Pxx))

	if len(s.Freqs) < 5 {
		return Spectra{ Freqs: freqs, Pxx: pxx }
	}

	for i := 2; i < len(s.Pxx) - 2; i++ {
		if lmax(s.Pxx[i-2], s.Pxx[i-1], s.Pxx[i], s.Pxx[i+1], s.Pxx[i+2]) {
			freqs = append(freqs, s.Freqs[i])
			pxx = append(pxx, s.Pxx[i])
			i += 2				// we can ignore the next 3 since they cannot be a maxima
		}
	}

	return NewSpectra(freqs, pxx)
}

func (s Spectra) ByPxx() Spectra {
	sort.Sort(ByPxx(s))
	return s
}

func (s Spectra) Tail(n int) Spectra {
	if n >= len(s.Freqs) {
		return s
	}

	nPxx := append([]float64(nil), s.Pxx[len(s.Pxx)-n-1:]...)
	nfreqs := append([]float64(nil), s.Freqs[len(s.Freqs)-n-1:]...)

	return NewSpectra(nfreqs, nPxx)
}

type Filterer func(freq, power float64) bool

func (s Spectra) Filter(f Filterer) Spectra {
	nPxx := make([]float64, 0, len(s.Pxx))
	nfreqs := make([]float64, 0, len(s.Freqs))
	for i, x := range s.Freqs {
		if f(s.Freqs[i], s.Pxx[i]) {
			nfreqs = append(nfreqs, x)
			nPxx = append(nPxx, s.Pxx[i])
		}
	}

	//log.Printf("filter: %d samples -> %d\n", len(Freqs), len(nfreqs))

	return NewSpectra(nfreqs, nPxx)
}

// filter out all the signals with a strength lower than the mean strength of all the samples
func (s Spectra) HighPass() (Spectra) {
	nPxx := make([]float64, 0, len(s.Pxx))
	nfreqs := make([]float64, 0, len(s.Freqs))

	mean := s.meanStrength()
	//log.Printf("highPassFIlter: mean = %f\n", mean)

	for i, v := range s.Pxx {
		if v >= mean {
			nfreqs = append(nfreqs, s.Freqs[i])
			nPxx = append(nPxx, v)
		}
	}

	return NewSpectra(nfreqs, nPxx)
}

// Calculate the mean strength of all the samples in the spectra
func (s Spectra) meanStrength() (m float64) {
	m = 0.0
	for _, v := range s.Pxx {
		m += v
	}
	m /= float64(len(s.Pxx))

	return
}

func NewSpectra(freqs, pxx []float64) Spectra {
	return Spectra{
		Pxx: pxx,
		Freqs: freqs,
	}
}

// check if x satisfies the criteria for a local maxima
func lmax(v1, v2, x, v4, v5 float64) bool {
	return v2 < x && v4 < x && v1 < v2 && v5 < v4
}

func (x Spectra) String() (s string) {
	s = ""
	for i := range x.Freqs {
		s = fmt.Sprintf("%s[%d] %7.2f(%5.2f)  ", s, i, x.Freqs[i], x.Pxx[i])
	}

	return
}

type ByPxx Spectra
func (a ByPxx) Len() int           { return len(a.Freqs) }
func (a ByPxx) Swap(i, j int)      { a.Freqs[i], a.Freqs[j] = a.Freqs[j], a.Freqs[i]
	a.Pxx[i], a.Pxx[j] = a.Pxx[j], a.Pxx[i] }
func (a ByPxx) Less(i, j int) bool { return a.Pxx[i] < a.Pxx[j] }

type ByFreq Spectra
func (a ByFreq) Len() int           { return len(a.Freqs) }
func (a ByFreq) Swap(i, j int)      { a.Freqs[i], a.Freqs[j] = a.Freqs[j], a.Freqs[i]
	a.Pxx[i], a.Pxx[j] = a.Pxx[j], a.Pxx[i] }
func (a ByFreq) Less(i, j int) bool { return a.Freqs[i] < a.Freqs[j] }

