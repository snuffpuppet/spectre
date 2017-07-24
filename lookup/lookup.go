package lookup


// The data that the fingerprint maps to
type Match struct {
	Filename    string
	Timestamp   float64
}

type Matches map[string]Match

func (m Matches) Add(fp []byte, filename string, ts float64) {
	m[string(fp)] = Match{ filename, ts }
}

func (m Matches) Lookup(fp []byte) (*Match, bool) {
	 v, ok := m[string(fp)]

	return &v, ok
}

func New() Matches {
	return make(Matches)
}

