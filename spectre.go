package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/signal"
	"github.com/snuffpuppet/spectre/audioFingerprint"
	"github.com/snuffpuppet/spectre/audioStream"
	"github.com/snuffpuppet/spectre/audioBuffer"
)

const SAMPLE_RATE = 11025
const BUFFER_SIZE = 1024


func generateFingerprints(filenames []string, optSpectralAnalyser int, optFormat string, optVerbose bool) (fingerprints map[string]audioFingerprint.Mapping, err error) {

	fingerprints = make(map[string]audioFingerprint.Mapping)

	var buffer audioBuffer.Buffer

	switch optFormat {
	case "int16":
		buffer = make([]int16, BUFFER_SIZE)
	case "float32":
		buffer = make([]float32, BUFFER_SIZE)
	default:
		log.Fatalf("Unrecognised data format in -format flag '%s'", optFormat)
	}
	clashCount := 0

	for _, filename := range filenames {
		fmt.Printf("Processing fingerprints for %s...\n", filename)
		stream, err := audioStream.NewBufferedWav(filename, buffer, SAMPLE_RATE)
		if (err != nil) {
			return nil, err
		}
		for {
			block, err := stream.ReadBlock()
			if (err != nil) {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					// EOF is ok, just break & go to next file
					break
				}
				return nil, err
			}
			fp, candidates := audioFingerprint.New(block, SAMPLE_RATE, optSpectralAnalyser)
			if (err != nil) {
				return nil, err
			}
			if fp != nil {
				if _, ok := fingerprints[string(fp.Hash)]; ok {
					clashCount++
				}
				fingerprints[string(fp.Hash)] = audioFingerprint.Mapping{filename, block.Timestamp}

			}
			if optVerbose {
				audioFingerprint.PrintCandidates(block.Id, block.Timestamp, candidates)
			}
		}
		stream.Close()
	}
	log.Printf("Fingerprint hash clashes: %d\n", clashCount)

	return fingerprints, nil
}

type matchedTimestamp struct {
	mic  float64
	song float64
}

type AudioMatcher struct {
	FingerprintLib map[string]audioFingerprint.Mapping
	FrequencyHits  map[string][]matchedTimestamp
}

func NewAudioMatcher(mappings map[string]audioFingerprint.Mapping) (*AudioMatcher){
	am := AudioMatcher{
		FrequencyHits: make(map[string][]matchedTimestamp),
		FingerprintLib: mappings,
	}
	return &am
}

func (matcher *AudioMatcher) register(fp *audioFingerprint.Fingerprint) {
	fpm, ok := matcher.FingerprintLib[string(fp.Hash)]
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

type audioHit struct {
	filename string
	hitCount int
	totalHitCount int
}

type audioHits []audioHit

func (a audioHits) String() (s string) {
	s = ""
	for _, v := range a {
		pc := float64(v.hitCount) / float64(v.totalHitCount) * 100.0
		s += fmt.Sprintf("%3d/%3d (%5.2f) - %s\n", v.hitCount, v.totalHitCount, pc, v.filename)
	}

	return
}

func (matcher *AudioMatcher) getHits() (orderedHits audioHits) {
	hits := make(map[string]int)
	totalHitcount := 0

	// Check through our frequency hit list to see if the time deltas match those of the file
	for filename, ts := range matcher.FrequencyHits {
		if len(ts) >1 {
			for i := 1; i < len(ts); i++ {
				songTimeDelta := ts[i].song - ts[i-1].song
				micTimeDelta := ts[i].mic - ts[i-1].mic

				if  math.Abs(songTimeDelta - micTimeDelta) < audioFingerprint.TIME_DELTA_THRESHOLD {
					hits[filename]++
					totalHitcount++
					log.Printf("Time Delta match for %s (%d/%d)\n", filename, hits[filename], totalHitcount)
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

func listen(audioMappings map[string]audioFingerprint.Mapping, optSpectralAnalyser int, optVerbose bool) error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill)

	stream, err := audioStream.NewMicrophone(BUFFER_SIZE, SAMPLE_RATE)
	if (err != nil) {
		return err
	}

	if err := stream.Start(); err != nil {
		return fmt.Errorf("Error starting microphone recording: %s", err)
	}

	fmt.Printf("Listening for %d fingerprints.  Press Ctrl-C to stop\n", len(audioMappings))

	audioMatcher := NewAudioMatcher(audioMappings)

	for {
		block, err := stream.ReadBlock()
		if err != nil {
			log.Fatalf("Error reading microphone: %s", err)
		}
		fingerprint, candidates := audioFingerprint.New(block, SAMPLE_RATE, optSpectralAnalyser)

		if fingerprint == nil {
			continue
		}

		if optVerbose {
			audioFingerprint.PrintCandidates(block.Id, block.Timestamp, candidates)
		}

		audioMatcher.register(fingerprint)

		// Check every second to see if they are certain enough to be a match
		if block.Id % 10 == 0 {
			hits := audioMatcher.getHits()
			if len(hits) > 0 {
				fmt.Println(hits)
			}
		}

		select {
		case <-sig:
			return nil
		default:
		}
	}

}

func main() {
	var optBespoke, optPWelch, optListen, optVerbose bool
	var optFormat string
	var optSpectralAnalyser int

	flag.BoolVar(&optBespoke, "bespoke", true, "Use bespoke spectral analyser")
	flag.BoolVar(&optPWelch, "pwelch", false, "Use PWelch spectral analyser")
	flag.BoolVar(&optListen, "listen", false, "Listen to microphone and generate spectral analysis")
	flag.BoolVar(&optVerbose, "verbose", false, "Verbose output of spectral analysis data")
	flag.StringVar(&optFormat, "format", "float32", "Format of data to use for spectral analysis (float32|int16)")

	flag.Parse()


	switch {
	case optBespoke:
		optSpectralAnalyser = audioFingerprint.SA_BESPOKE
	case optPWelch:
		optSpectralAnalyser = audioFingerprint.SA_PWELCH
	}

	if (len(flag.Args()) == 0) {
		log.Fatal("Usage: bespokesa [-bespoke] [-pwelch] [-format int16|float32] file [file ...]")
	}

	filenames := flag.Args()

	fingerprints, err := generateFingerprints(filenames, optSpectralAnalyser, optFormat, optVerbose)
	if err != nil {
		log.Fatalf("Fatal Error generating fingerprints: %s", err)
	}

	if optListen {
		err := listen(fingerprints, optSpectralAnalyser, optVerbose)
		if err != nil {
			log.Fatalf("Fatal Error getting stream: %s", err)
		}
	}
}
