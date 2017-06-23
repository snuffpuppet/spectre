package audiomatcher

import (
	"github.com/snuffpuppet/spectre/fingerprint"
	"fmt"
	"math"
)

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

func (a audioHits) String() (s string) {
	s = ""
	for _, v := range a {
		pc := int(float64(v.hitCount) / float64(v.totalHitCount) * 100.0 + 0.5)
		s += fmt.Sprintf("%3d/%3d (%3d%%) - %s\n", v.hitCount, v.totalHitCount, pc, v.filename)
	}

	return
}

type AudioMatcher struct {
	FingerprintLib map[string]fingerprint.Mapping
	FrequencyHits  map[string][]matchedTimestamp
}

func New(mappings map[string]fingerprint.Mapping) (*AudioMatcher) {
	am := AudioMatcher{
		FrequencyHits: make(map[string][]matchedTimestamp),
		FingerprintLib: mappings,
	}
	return &am
}

func (matcher *AudioMatcher) Register(fp *fingerprint.Fingerprint) {
	fpm, ok := matcher.FingerprintLib[string(fp.Key)]
	if !ok {
		return
	}
	// we have  frequency match, now add the match to the list
	timestamps, ok := matcher.FrequencyHits[fpm.Filename]
	if !ok {
		timestamps = make([]matchedTimestamp, 1)
	}

	matcher.FrequencyHits[fpm.Filename] = append(timestamps, matchedTimestamp{mic: fp.Timestamp, song: fpm.Timestamp})
	fmt.Printf("Frequency match for %s at %.2f\n", fpm.Filename, fpm.Timestamp)
}

func (matcher *AudioMatcher) GetHits() (orderedHits audioHits) {
	hits := make(map[string]int)
	totalHitcount := 0

	// Check through our frequency hit list to see if the time deltas match those of the file
	for filename, ts := range matcher.FrequencyHits {
		if len(ts) >1 {
			for i := 1; i < len(ts); i++ {
				songTimeDelta := ts[i].song - ts[i-1].song
				micTimeDelta := ts[i].mic - ts[i-1].mic

				if  math.Abs(songTimeDelta - micTimeDelta) < fingerprint.TIME_DELTA_THRESHOLD {
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

