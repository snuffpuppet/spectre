package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"github.com/snuffpuppet/spectre/fingerprint"
	"github.com/snuffpuppet/spectre/pcmstream"
	"github.com/snuffpuppet/spectre/pcmframe"
	"github.com/snuffpuppet/spectre/audiomatcher"
)

const SAMPLE_RATE = 11025
const BUFFER_SIZE = 4096

func printStatus(fp *fingerprint.Fingerprint, block *pcmframe.Block, verbose bool) {
	if verbose {
		header := fmt.Sprintf("[%4d:%6.2f]", block.Id, block.Timestamp)
		fmt.Printf("%s %s\n", header, fp.Candidates)
		//fmt.Printf("%s %s\n", header, fp.Transcription)
		//fmt.Printf("%s -> Key: %v\n\n", header, fp.Key)
	}
}

func generateFingerprints(filenames []string, optSpectralAnalyser int, optFormat string, optVerbose bool) (fingerprints map[string]fingerprint.Mapping, err error) {

	fingerprints = make(map[string]fingerprint.Mapping)

	var buffer pcmframe.Buffer

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
		stream, err := pcmstream.NewBufferedWav(filename, buffer, SAMPLE_RATE)
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
			fp := fingerprint.New(block, SAMPLE_RATE, optSpectralAnalyser)
			if (err != nil) {
				return nil, err
			}
			if fp != nil {
				if _, ok := fingerprints[string(fp.Key)]; ok {
					clashCount++
				}
				fingerprints[string(fp.Key)] = fingerprint.Mapping{filename, block.Timestamp}

				printStatus(fp, block, optVerbose)

			}
		}
		stream.Close()
	}
	log.Printf("Fingerprint hash clashes: %d\n", clashCount)

	return fingerprints, nil
}

func listen(audioMappings map[string]fingerprint.Mapping, optSpectralAnalyser int, optVerbose bool) error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill)

	stream, err := pcmstream.NewMicrophone(BUFFER_SIZE, SAMPLE_RATE)
	if (err != nil) {
		return err
	}

	if err := stream.Start(); err != nil {
		return fmt.Errorf("Error starting microphone recording: %s", err)
	}

	fmt.Printf("Listening for %d fingerprints.  Press Ctrl-C to stop\n", len(audioMappings))

	matcher := audiomatcher.New(audioMappings)

	for {
		block, err := stream.ReadBlock()
		if err != nil {
			log.Fatalf("Error reading microphone: %s", err)
		}
		fp := fingerprint.New(block, SAMPLE_RATE, optSpectralAnalyser)

		if fp == nil {
			continue
		}

		printStatus(fp, block, optVerbose)

		matcher.Register(fp)

		// Check every second to see if they are certain enough to be a match
		if block.Id % 10 == 0 {
			hits := matcher.GetHits()
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
		optSpectralAnalyser = fingerprint.SA_BESPOKE
	case optPWelch:
		optSpectralAnalyser = fingerprint.SA_PWELCH
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
