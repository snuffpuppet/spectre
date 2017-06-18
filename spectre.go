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
		log.Fatal("Unrecognised data format in -format flag '%s'", optFormat)
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
			hash, candidates, err := audioFingerprint.New(block, SAMPLE_RATE, optSpectralAnalyser)
			if (err != nil) {
				return nil, err
			}
			if hash != nil {
				if _, ok := fingerprints[string(hash)]; ok {
					clashCount++
				}
				fingerprints[string(hash)] = audioFingerprint.Mapping{filename, block.Timestamp}

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

	matchedTimestamps := make(map[string][]matchedTimestamp)

	for {
		block, err := stream.ReadBlock()
		if err != nil {
			log.Fatalf("Error reading microphone: %s", err)
		}
		fingerprint, candidates, err := audioFingerprint.New(block, SAMPLE_RATE, optSpectralAnalyser)
		if (err != nil) {
			return err
		}

		if optVerbose {
			audioFingerprint.PrintCandidates(block.Id, block.Timestamp, candidates)
		}

		am, ok := audioMappings[string(fingerprint)]
		if ok {
			matches, ok := matchedTimestamps[am.Filename]
			if !ok {
				matches = make([]matchedTimestamp, 1)
			}
			matchedTimestamps[am.Filename] = append(matches, matchedTimestamp{mic: block.Timestamp, song: am.Timestamp})
			fmt.Printf("Frequencymatch for %s at %.2f\n", am.Filename, am.Timestamp)
		}

		// every second check to see if we have timestamp matches for our frequencies
		if block.Id % 10 == 0 {
			for filename, matches := range matchedTimestamps {
				//matches := matchedTimestamps[k]
				hitCount := 0
				hitTime := matches[0].song + block.Timestamp - matches[0].mic
				if len(matches) > 2 {
					for i := 1; i < len(matches); i++ {
						songTimeDelta := matches[i].song - matches[i-1].song
						micTimeDelta := matches[i].mic - matches[i-1].mic

						if  math.Abs(songTimeDelta - micTimeDelta) < audioFingerprint.TIME_DELTA_THRESHOLD {
							hitCount++
						}
					}
				}
				if hitCount > 2 {
					fmt.Printf("matches %2d, hits %2d at %6.2f for %s \n", len(matches), hitCount, hitTime, filename)
				}
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
