package audiomatcher

import (
	//"github.com/snuffpuppet/spectre/fingerprint"
	"fmt"
	"math"
)

/*
 * audioMatcher:
 * Check for frequency / strength matches as well as temporal ones
 * If we have a frequency match, log the time info which can be checked later to determine if the frequency match
 * is in the right place in the song.
 * Temporal matches are currently just a simple list of matches that get checked
 */
type matchedTimestamp struct {
	mic  float64
	song float64
}

type audioHit struct {
	filename string
	hitCount int
	totalHitCount int
}

type audioHits []audioHit

// The data that the fingerprint maps to
type Match struct {
	Filename    string
	Timestamp   float64
}

type Matches map[string]Match

type Fingerprinter interface {
	Fingerprint() []byte
}

func (a audioHits) String() (s string) {
	s = ""
	for _, v := range a {
		pc := int(float64(v.hitCount) / float64(v.totalHitCount) * 100.0 + 0.5)
		s += fmt.Sprintf("%3d/%3d (%3d%%) - %s\n", v.hitCount, v.totalHitCount, pc, v.filename)
	}

	return
}

type AudioMatcher struct {
	FingerprintLib Matches
	FrequencyHits  map[string][]matchedTimestamp
}

func New(mappings Matches) (*AudioMatcher) {
	am := AudioMatcher{
		FrequencyHits: make(map[string][]matchedTimestamp),
		FingerprintLib: mappings,
	}
	return &am
}

// register a fingerprint with the audio matcher in order to log the timestamps
func (matcher *AudioMatcher) Register(fp Fingerprinter, ts float64) {
	fpm, ok := matcher.FingerprintLib[string(fp.Fingerprint())]
	if !ok {
		return
	}
	// we have  frequency match, now add the match to the list
	timestamps, ok := matcher.FrequencyHits[fpm.Filename]
	if !ok {
		timestamps = make([]matchedTimestamp, 1)
	}

	matcher.FrequencyHits[fpm.Filename] = append(timestamps, matchedTimestamp{mic: ts, song: fpm.Timestamp})
	fmt.Printf("Frequency match for %s at %.2f\n", fpm.Filename, fpm.Timestamp)
}

// return a slice of audioHits that the caller can use to determine the probability of a match
func (matcher *AudioMatcher) GetHits(timeDeltaThreshold float64) (orderedHits audioHits) {
	hits := make(map[string]int)
	totalHitcount := 0

	// Check through our frequency hit list to see if the time deltas match those of the file
	for filename, ts := range matcher.FrequencyHits {
		if len(ts) >1 {
			for i := 1; i < len(ts); i++ {
				songTimeDelta := ts[i].song - ts[i-1].song
				micTimeDelta := ts[i].mic - ts[i-1].mic

				if  math.Abs(songTimeDelta - micTimeDelta) < timeDeltaThreshold {
					hits[filename]++
					totalHitcount++
					fmt.Printf("Time Delta match for %s (%d/%d)\n", filename, hits[filename], totalHitcount)
				}
			}
		}
	}

	// Hits calculated, now provide a sorted list to the caller
	orderedHits = audioHits(make([]audioHit, 0))
	for filename, hitCount := range hits {
		hit := audioHit{filename, hitCount, totalHitcount}
		inserted := false
		if len(orderedHits) > 0 {
			for i := 0; i < len(orderedHits); i++ {
				if hit.hitCount > orderedHits[i].hitCount {
					fmt.Println(orderedHits[i])
					orderedHits = append(orderedHits[:i], append([]audioHit{hit}, orderedHits[i:]...)...)
					inserted = true
					break
				}
			}
		}
		if !inserted {
			orderedHits = append(orderedHits, hit)
		}
	}

	return
}

