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
	//"github.com/snuffpuppet/spectre/audiomatcher"
	"bufio"
	"github.com/snuffpuppet/spectre/analysis"
	"github.com/snuffpuppet/spectre/audiomatcher"
)

const (
	_ = iota
	SA_PWELCH = iota
	SA_BESPOKE = iota
)

const SAMPLE_RATE = 11025
const BLOCK_SIZE  = 4096

//const REQUIRED_CANDIDATES = 4 	// required number of frequency candidates for a fingerprint entry
const LOWER_FREQ_CUTOFF = 318.0		// Lowest frequency acceptable for matching
const UPPER_FREQ_CUTOFF = 2000.0	// Highest frequency acceptable for matching

const TIME_DELTA_THRESHOLD = 0.2	// required minimum time diff between freq matches to be considered a hit

const FILE_SILENCE_THRESHOLD = 30.0
const MIC_SILENCE_THRESHOLD = 30.0

type Fingerprinter interface {
	Fingerprint() []byte
	//Timestamp()   float64
}

type Timestamper interface {
	Timestamp() float64
}

type Ider interface {
	BlockId() int
}

type IdTimestamper interface {
	Ider
	Timestamper
}

type TimeIdFLoat64Slicer interface {
	Timestamper
	Ider
	asFloat64() []float64
}


func printStatus(fp Fingerprinter, frame IdTimestamper, verbose bool) {
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

type PcmReader interface {
	Read() (*pcm.Frame, error)
}

func getFingerprint(analyser analysis.SpectralAnalyser, samples []float64, silenceThreshold float64) (Fingerprinter) {
	//s := ""

	spectra := analyser(samples, SAMPLE_RATE)
	//log.Printf("Raw Samples:\n%v\n%v\n\n", spectra.Freqs, spectra.Pxx)
	//s = fmt.Sprintf("%s -> samples=%d", s, len(spectra.Freqs))

	spectra = spectra.Filter(LOWER_FREQ_CUTOFF, UPPER_FREQ_CUTOFF, silenceThreshold)
	//s = fmt.Sprintf("%s -> audible=%d", s, len(spectra.Freqs))

	spectra = spectra.Maxima()
	//s = fmt.Sprintf("%s -> maxima=%d", s, len(spectra.Freqs))

	spectra = spectra.HighPass()
	//s = fmt.Sprintf("%s -> highPass=%d", s, len(spectra.Freqs))

	//log.Println(s)

	//fp := fingerprint.NewChromaprint(Pxx, freqs)
	fp := fingerprint.NewChromaprint(spectra)
	
	if fp == nil {
		return nil
	}

	return fp
}


func loadStream(filename string, stream PcmReader, matches audiomatcher.Matches, analyser analysis.SpectralAnalyser, optVerbose bool) (audiomatcher.Matches, error){
	clashCount, fpCount := 0, 0
	for {
		frame, err := stream.Read()
		if (err != nil) {
			if (err == io.EOF || err == io.ErrUnexpectedEOF) {
				break
			}
			return matches, err
		}

		fp := getFingerprint(analyser, frame.AsFloat64(), FILE_SILENCE_THRESHOLD)

		printStatus(fp, frame, optVerbose)

		if fp != nil {
			fpCount++
			if _, ok := matches[string(fp.Fingerprint())]; ok {
				clashCount++
			}
			matches[string(fp.Fingerprint())] = audiomatcher.Match{filename, frame.Timestamp()}
		}
	}

	log.Printf("%s:\tFingerprints %d, hash clashes: %d\n", filename, fpCount, clashCount)

	return matches, nil
}

func loadFiles(filenames []string, analyser analysis.SpectralAnalyser, optVerbose bool) (matches audiomatcher.Matches, err error) {

	matches = make(audiomatcher.Matches)

	for _, filename := range filenames {
		fmt.Printf("Processing fingerprints for %s...\n", filename)
		stream, err := pcm.NewFileStream(filename, SAMPLE_RATE, BLOCK_SIZE)
		if (err != nil) {
			return nil, err
		}

		matches, err = loadStream(filename, stream, matches, analyser, optVerbose)

		stream.Close()
	}

	return matches, nil
}

func listen(audioMappings audiomatcher.Matches, analyser analysis.SpectralAnalyser, optVerbose bool) error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill)

	stream, err := pcm.NewMicStream(SAMPLE_RATE, BLOCK_SIZE)
	if (err != nil) {
		return err
	}

	if err := stream.Start(); err != nil {
		return fmt.Errorf("Error starting microphone recording: %s", err)
	}

	fmt.Printf("Listening for %d fingerprints.  Press Ctrl-C to stop\n", len(audioMappings))

	matcher := audiomatcher.New(audioMappings)

	for {
		frame, err := stream.Read()
		if err != nil {
			log.Fatalf("Error reading microphone: %s", err)
		}

		fp := getFingerprint(analyser, frame.AsFloat64(), MIC_SILENCE_THRESHOLD)

		if fp != nil {

			printStatus(fp, frame, optVerbose)

			matcher.Register(fp, frame.Timestamp())

			// Check every second to see if they are certain enough to be a match
			if frame.BlockId() % 10 == 0 {
				hits := matcher.GetHits(TIME_DELTA_THRESHOLD)
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

	in, err := pcm.NewMicStream(SAMPLE_RATE, BLOCK_SIZE)
	if (err != nil) {
		return err
	}

	if err := in.Start(); err != nil {
		return fmt.Errorf("Error starting microphone recording: %s", err)
	}

	for {
		frame, err := in.Read()
		if err != nil {
			log.Fatalf("Error reading microphone: %s", err)
		}

		// write a chunk
		if err := binary.Write(w, binary.LittleEndian, frame.Data()); err != nil {
			panic(err)
		}

		select {
		case <-sig:
			return nil
		default:
		}
	}

}

func dumpBlock(frame *pcm.Frame) {
	header := fmt.Sprintf("[%4d:%6.2f] ", frame.BlockId(), frame.Timestamp())
	//fmt.Printf("%s %s\n", header, fp.Candidates)
	fmt.Printf("%s\n%v\n", header, frame.Data())
}


func dumpData(filenames []string) {
	for _, filename := range filenames {

		stream, err := pcm.NewFileStream(filename, SAMPLE_RATE, BLOCK_SIZE)

		fmt.Printf("Dumping data for %s...\n", filename)

		if (err != nil) {
			return
		}
		for {
			frame, err := stream.Read()
			if (err != nil) {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					// EOF is ok, just break & go to next file
					break
				}
				return
			}
			dumpBlock(frame)
		}
	}
}


func main() {
	var optListen, optVerbose, optDump, optRecord bool
	var optOutFile, optAnalyser string
	var analyser analysis.SpectralAnalyser

	flag.BoolVar(&optListen, "listen", false, "Listen to microphone for song matches")
	flag.BoolVar(&optVerbose, "verbose", false, "Verbose output of spectral analysis data")
	flag.BoolVar(&optRecord, "record", false, "Record data from Mic into -output file")
	//flag.StringVar(&optFormat, "format", "float32", "Format of data to use for spectral analysis (float32|int16)")
	flag.StringVar(&optAnalyser, "analyser", "bespoke", "Spectral analyser to use (pwelch | bespoke)")
	flag.StringVar(&optOutFile, "output", "", "Output file for recording")
	flag.BoolVar(&optDump, "dump", false, "Dump data in raw format from input file")

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

	if (len(flag.Args()) == 0 && !optRecord) {
		flag.PrintDefaults()
		os.Exit(1)
	}

	filenames := flag.Args()

	if optDump {
		dumpData(filenames)
	}

	fmt.Printf("Using '%s' analysis to generate fingerprints for %v\n", optAnalyser, filenames)

	fingerprints, err := loadFiles(filenames, analyser, optVerbose)
	if err != nil {
		log.Fatalf("Fatal Error generating fingerprints: %s", err)
	}

	switch {
	case optListen:
		if optRecord {
			flag.PrintDefaults()
			break
		}
		err := listen(fingerprints, analyser, optVerbose)
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
