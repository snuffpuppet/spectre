package analysis

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
			i += 3				// we can ignore the next 3 since they cannot be a maxima
		}
	}

	return NewSpectra(freqs, pxx)
}

func (s Spectra) Filter(lowFreq, highFreq, lowPower float64) Spectra {
	nPxx := make([]float64, 0, len(s.Pxx))
	nfreqs := make([]float64, 0, len(s.Freqs))
	for i, x := range s.Freqs {
		if x >= lowFreq && x <= highFreq && s.Pxx[i] > lowPower {
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