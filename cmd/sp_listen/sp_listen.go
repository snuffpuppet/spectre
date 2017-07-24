package main

import (
	"flag"
	"github.com/snuffpuppet/spectre/analysis"
	"log"
	"os"
	"fmt"
	"github.com/snuffpuppet/spectre/pcm"
	"github.com/snuffpuppet/spectre/audiomatcher"
	"os/signal"
	"io"
	"github.com/snuffpuppet/spectre/fingerprint"
	"github.com/snuffpuppet/spectre/lookup"
)

func loadFiles(filenames []string, analyser analysis.SpectralAnalyser, optVerbose bool) (matches lookup.Matches, err error) {

	matches = lookup.New()

	for _, filename := range filenames {
		fmt.Printf("Processing fingerprints for %s...\n", filename)
		stream, err := pcm.NewFileStream(filename, fingerprint.SAMPLE_RATE, fingerprint.BLOCK_SIZE)
		if (err != nil) {
			return nil, err
		}

		matches, err = loadStream(filename, stream, matches, analyser, optVerbose)

		stream.Close()
	}

	return matches, nil
}

func listen(stream pcm.StartReader, matcher *audiomatcher.AudioMatcher, analyser analysis.SpectralAnalyser, optVerbose bool) error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill)

	if err := stream.Start(); err != nil {
		return fmt.Errorf("Error starting microphone recording: %s", err)
	}

	fmt.Printf("Listening for %d fingerprints.  Press Ctrl-C to stop\n", len(matcher.FingerprintLib))

	for {
		frame, err := stream.Read()
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			}
			log.Fatalf("Error reading microphone: %s", err)
		}

		fp := fingerprint.Generate(analyser, frame.AsFloat64(), fingerprint.MIC_SILENCE_THRESHOLD)

		if fp != nil {

			printStatus(fp, frame, optVerbose)

			matcher.Register(fp, frame.Timestamp())

			// Check every second to see if they are certain enough to be a match
			if frame.BlockId() % fingerprint.BLOCKS_PER_SECOND == 0 {
				hits := matcher.GetHits()
				if len(hits) > 0 {
					//fmt.Println(hits)
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

func loadStream(filename string, stream pcm.Reader, matches lookup.Matches, analyser analysis.SpectralAnalyser, optVerbose bool) (lookup.Matches, error){
	clashCount, fpCount := 0, 0
	for {
		frame, err := stream.Read()
		if (err != nil) {
			if (err == io.EOF || err == io.ErrUnexpectedEOF) {
				break
			}
			return matches, err
		}

		fp := fingerprint.Generate(analyser, frame.AsFloat64(), fingerprint.FILE_SILENCE_THRESHOLD)

		printStatus(fp, frame, optVerbose)

		if fp != nil {
			fpCount++
/*			if _, ok := matches[string(fp.Fingerprint())]; ok {
				clashCount++
			}
			matches[string(fp.Fingerprint())] = audiomatcher.Match{filename, frame.Timestamp()}
*/
			if _, ok := matches.Lookup(fp.Fingerprint()); ok {
				clashCount++
			}
			matches.Add(fp.Fingerprint(), filename, frame.Timestamp())
		}
	}

	log.Printf("%s:\tFingerprints %d, hash clashes: %d\n", filename, fpCount, clashCount)

	return matches, nil
}

func printStatus(fp fingerprint.Fingerprinter, frame *pcm.Frame, verbose bool) {
	if verbose {
		header := fmt.Sprintf("[%4d:%6.2f]", frame.BlockId(), frame.Timestamp())
		if fp == nil {
			fmt.Printf("%s fp: nil\n", header)
		} else {
			//fmt.Printf("%s %s\n", header, fp.Candidates)
			fmt.Printf("%s %s\n", header, fp)
			//fmt.Printf("%s -> Key: %v\n\n", header, fp.Fingerprint())
		}
	}
}

func main() {
	var optVerbose bool
	var optAnalyser, optInput string
	var analyser analysis.SpectralAnalyser

	flag.BoolVar(&optVerbose, "verbose", false, "Verbose output of spectral analysis data")
	flag.StringVar(&optAnalyser, "analyser", "bespoke", "Spectral analyser to use (pwelch | bespoke)")
	flag.StringVar(&optInput, "input", "", "Input file to use instead of microphone")

	flag.Parse()

	switch optAnalyser {
	case "bespoke":
		analyser = analysis.Amplitude
	case "pwelch":
		analyser = analysis.Pwelch
	default:
		flag.PrintDefaults()
		log.Fatalf("Unrecognised spectral analyser requested: '%s'", optAnalyser)

	}

	if (len(flag.Args()) == 0) {
		log.Println("Error: No audio files found to match against")
		flag.PrintDefaults()
		os.Exit(1)
	}

	filenames := flag.Args()

	fmt.Printf("Using '%s' analysis to generate fingerprints for %v\n", optAnalyser, filenames)

	fingerprints, err := loadFiles(filenames, analyser, optVerbose)
	if err != nil {
		log.Fatalf("Fatal Error generating fingerprints: %s", err)
	}

	var input pcm.StartReader
	if optInput != "" {
		input, err = pcm.NewFileStream(optInput, fingerprint.SAMPLE_RATE, fingerprint.BLOCK_SIZE)
	} else {
		input, err = pcm.NewMicStream(fingerprint.SAMPLE_RATE, fingerprint.BLOCK_SIZE)
	}
	if err != nil {
		log.Fatalf("Fatal Error opening stream: %s", err)
	}

	matcher := audiomatcher.New(fingerprints, fingerprint.TIME_DELTA_THRESHOLD)

	err = listen(input, matcher, analyser, optVerbose)
	if err != nil {
		log.Fatalf("Fatal Error listening to stream: %s", err)
	}

	fmt.Println(matcher.Stats())
}
