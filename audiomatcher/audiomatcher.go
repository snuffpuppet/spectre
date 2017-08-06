package audiomatcher

import (
	//"github.com/snuffpuppet/spectre/fingerprint"
	"fmt"
	"math"
	"github.com/snuffpuppet/spectre/lookup"
)

/*
 * audioMatcher:
 * Check for frequency / strength matches as well as temporal ones
 * If we have a frequency match, log the time info which can be checked later to determine if the frequency match
 * is in the right place in the song.
 * Temporal matches are currently just a simple list of matches that get checked
 */
type location struct {
	mic  float64
	song float64
}

type audioHit struct {
	filename string
	hitCount int
	totalHitCount int
}

type audioHits []audioHit


func (a audioHits) String() (s string) {
	s = ""
	for _, v := range a {
		pc := int(float64(v.hitCount) / float64(v.totalHitCount) * 100.0 + 0.5)
		s += fmt.Sprintf("%3d/%3d (%3d%%) - %s\n", v.hitCount, v.totalHitCount, pc, v.filename)
	}

	return
}

type AudioMatcher struct {
	timeThreshold  float64
	FingerprintLib lookup.Matches
	FrequencyHits  map[string][]location
}

func New(mappings lookup.Matches, timeThreshold float64) (*AudioMatcher) {
	am := AudioMatcher{
		timeThreshold: timeThreshold,
		FrequencyHits: make(map[string][]location),
		FingerprintLib: mappings,
	}
	return &am
}

// register a fingerprint with the audio matcher in order to log the timestamps
func (matcher *AudioMatcher) Register(key []byte, ts float64) {
	fpm, ok := matcher.FingerprintLib[string(key)]
	if !ok {
		return
	}
	// we have  frequency match, now add the match to the list
	timestamps, ok := matcher.FrequencyHits[fpm.Filename]
	if !ok {
		timestamps = make([]location, 1)
	}

	matcher.FrequencyHits[fpm.Filename] = append(timestamps, location{mic: ts, song: fpm.Timestamp})
	//fmt.Printf("Frequency match for %s at %.2f\n", fpm.Filename, fpm.Timestamp)
}

func (m *AudioMatcher) Stats() (s string) {
	hits, misses, totalHits, totalMisses := m.hitStats()
	header := fmt.Sprintf("Totals - hits: %d / osync: %d / total: %d", totalHits, totalMisses, totalHits + totalMisses)
	body := ""
	for k, v := range hits {
		mv := misses[k]
		body = fmt.Sprintf("%s\n%s: %d/%d/%d", body, k, v, mv, v + mv)
	}

	s = fmt.Sprintf("%s%s", header, body)

	return
}

func (m *AudioMatcher) hitStats() (hits, misses map[string]int, totalHits, totalMisses int) {
	hits = make(map[string]int)
	misses = make(map[string]int)
	totalHits, totalMisses= 0,0

	// Check through our frequency hit list to see if the time deltas match those of the file
	for filename, ts := range m.FrequencyHits {
		if len(ts) >1 {
			for i := 1; i < len(ts); i++ {
				songTimeDelta := ts[i].song - ts[i-1].song
				micTimeDelta := ts[i].mic - ts[i-1].mic

				if  math.Abs(songTimeDelta - micTimeDelta) < m.timeThreshold {
					hits[filename]++
					totalHits++
					//fmt.Printf("Time Delta match for %s (%d/%d)\n", filename, hits[filename], totalHits)
				} else {
					misses[filename]++
					totalMisses++
				}
			}
		}
	}

	return
}

// return a slice of audioHits that the caller can use to determine the probability of a match
func (matcher *AudioMatcher) GetHits() (orderedHits audioHits) {
	hits, _, totalHits, _ := matcher.hitStats()

	// Hits calculated, now provide a sorted list to the caller
	orderedHits = make([]audioHit, 0)
	for filename, hitCount := range hits {
		hit := audioHit{filename, hitCount, totalHits}
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

