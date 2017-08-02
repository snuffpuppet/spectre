package fingerprint

import "fmt"

type CandidateFilter func(candidate) bool

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

func (cs candidates) filter(f CandidateFilter) (fc candidates) {
	fc = make(candidates, 0, 5)
	for _, c := range cs {
		if (f(c)) {
			fc = append(fc, c)
		}
	}

	return
}

func NewCandidates(Pxx, freqs []float64) (c candidates) {
	c = make(candidates, 0)
	for i, x := range freqs {
		c = append(c, candidate{Freq: x, Pxx: Pxx[i]})
	}

	return
}

type ByPxx []candidate
func (a ByPxx) Len() int           { return len(a) }
func (a ByPxx) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByPxx) Less(i, j int) bool { return a[i].Pxx < a[j].Pxx }

type ByFreq []candidate
func (a ByFreq) Len() int           { return len(a) }
func (a ByFreq) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByFreq) Less(i, j int) bool { return a[i].Freq < a[j].Freq }


