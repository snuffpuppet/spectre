package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"github.com/snuffpuppet/spectre/fingerprint"
	"github.com/snuffpuppet/spectre/pcm"
	"github.com/snuffpuppet/spectre/audiomatcher"
	"bufio"
)

const SAMPLE_RATE = 11025


func printStatus(fp *fingerprint.Fingerprint, block *pcm.Buffer, verbose bool) {
	if verbose {
		header := fmt.Sprintf("[%4d:%6.2f]", block.Id, block.Timestamp)
		if fp == nil {
			fmt.Printf("%s fp: nil\n", header)
		} else {
			//fmt.Printf("%s %s\n", header, fp.Candidates)
			fmt.Printf("%s %s\n", header, fp.Transcription)
			fmt.Printf("%s -> Key: %v\n\n", header, fp.Key)
		}
	}
}

func generateFingerprints(filenames []string, optSpectralAnalyser int, optVerbose bool) (fingerprints map[string]fingerprint.Mapping, err error) {

	fingerprints = make(map[string]fingerprint.Mapping)

	var clashCount, fpCount int

	for _, filename := range filenames {
		fmt.Printf("Processing fingerprints for %s...\n", filename)
		stream, err := pcm.NewWavStream(filename, SAMPLE_RATE)
		if (err != nil) {
			return nil, err
		}
		for {
			block, err := stream.ReadFrame()
			if (err != nil) {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					// EOF is ok, just break & go to next file
					break
				}
				return nil, err
			}
			fp := fingerprint.New(block, SAMPLE_RATE, optSpectralAnalyser, optVerbose)
			if (err != nil) {
				return nil, err
			}

			printStatus(fp, block, optVerbose)

			if fp != nil {
				fpCount++
				if _, ok := fingerprints[string(fp.Key)]; ok {
					clashCount++
				}
				fingerprints[string(fp.Key)] = fingerprint.Mapping{filename, block.Timestamp}
			}
		}
		stream.Close()
	}
	log.Printf("Fingerprints: %d, hash clashes: %d\n", fpCount, clashCount)

	return fingerprints, nil
}

func listen(audioMappings map[string]fingerprint.Mapping, optSpectralAnalyser int, optVerbose bool) error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill)

	stream, err := pcm.NewMicStream(SAMPLE_RATE)
	if (err != nil) {
		return err
	}

	if err := stream.Start(); err != nil {
		return fmt.Errorf("Error starting microphone recording: %s", err)
	}

	fmt.Printf("Listening for %d fingerprints.  Press Ctrl-C to stop\n", len(audioMappings))

	matcher := audiomatcher.New(audioMappings)

	for {
		buf, err := stream.ReadFrame()
		if err != nil {
			log.Fatalf("Error reading microphone: %s", err)
		}
		fp := fingerprint.New(buf, SAMPLE_RATE, optSpectralAnalyser, optVerbose)

		if fp != nil {

			printStatus(fp, buf, optVerbose)

			matcher.Register(fp)

			// Check every second to see if they are certain enough to be a match
			if buf.Id % 10 == 0 {
				hits := matcher.GetHits()
				if len(hits) > 0 {
					fmt.Println(hits)
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

func record(outfile string) error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill)

	// open output file
	fo, err := os.Create(outfile)
	if err != nil {
		panic(err)
	}

	// close fo on exit and check for its returned error
	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()

	// make a write buffer
	w := bufio.NewWriter(fo)

	in, err := pcm.NewMicStream(SAMPLE_RATE)
	if (err != nil) {
		return err
	}

	if err := in.Start(); err != nil {
		return fmt.Errorf("Error starting microphone recording: %s", err)
	}

	for {
		buf, err := in.ReadFrame()
		if err != nil {
			log.Fatalf("Error reading microphone: %s", err)
		}

		// write a chunk
		if err := binary.Write(w, binary.LittleEndian, buf.Frame()); err != nil {
			panic(err)
		}

		select {
		case <-sig:
			return nil
		default:
		}
	}

}

func dumpBlock(buf *pcm.Buffer) {
	header := fmt.Sprintf("[%4d:%6.2f] (%s)", buf.Id, buf.Timestamp, buf.DataFormat())
	//fmt.Printf("%s %s\n", header, fp.Candidates)
	fmt.Printf("%s\n%v\n", header, buf.Frame())
}


func dumpData(optFormat string, filenames []string) {
	for _, filename := range filenames {
		fmt.Printf("Dumping data for %s using %s...\n", filename, optFormat)
		stream, err := pcm.NewWavStream(filename, SAMPLE_RATE)
		if (err != nil) {
			return
		}
		for {
			block, err := stream.ReadFrame()
			if (err != nil) {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					// EOF is ok, just break & go to next file
					break
				}
				return
			}
			dumpBlock(block)
		}
	}
}


func main() {
	var optListen, optVerbose, optDump, optRecord bool
	var optOutFile, optAnalyser string
	var optSpectralAnalyser int

	flag.BoolVar(&optListen, "listen", false, "Listen to microphone for song matches")
	flag.BoolVar(&optVerbose, "verbose", false, "Verbose output of spectral analysis data")
	flag.BoolVar(&optRecord, "record", false, "Record data from Mic into -output file")
	//flag.StringVar(&optFormat, "format", "float32", "Format of data to use for spectral analysis (float32|int16)")
	flag.StringVar(&optAnalyser, "analyser", "pwelch", "Spectral analyser to use (pwelch | bespoke)")
	flag.StringVar(&optOutFile, "output", "", "Output file for recording")
	flag.BoolVar(&optDump, "dump", false, "Dump data in raw format from input file")

	flag.Parse()

	switch optAnalyser {
	case "bespoke":
		optSpectralAnalyser = fingerprint.SA_BESPOKE
	case "pwelch":
		optSpectralAnalyser = fingerprint.SA_PWELCH
	default:
		flag.PrintDefaults()
		log.Fatalf("Unrecognised spectral analyser requested: '%s'", optAnalyser)

	}

	if (len(flag.Args()) == 0 && !optRecord) {
		flag.PrintDefaults()
		os.Exit(1)
	}

	filenames := flag.Args()

	if optDump {
		dumpData(optFormat, filenames)
	}

	fmt.Printf("Using '%s' analysis to generate fingerprints for %v\n", optAnalyser, filenames)

	fingerprints, err := generateFingerprints(filenames, optSpectralAnalyser, optVerbose)
	if err != nil {
		log.Fatalf("Fatal Error generating fingerprints: %s", err)
	}

	switch {
	case optListen:
		if optRecord {
			flag.PrintDefaults()
			break
		}
		err := listen(fingerprints, optSpectralAnalyser, optVerbose)
		if err != nil {
			log.Fatalf("Fatal Error getting stream: %s", err)
		}
	case optRecord:
		if optListen {
			flag.PrintDefaults()
			break
		}
		record(optOutFile)
	}
}
