package main

import (
	"fmt"
	"log"
	"flag"
	"github.com/snuffpuppet/spectre/spectral"
	"os"
	"github.com/snuffpuppet/spectre/pcm"
	"github.com/snuffpuppet/spectre/fingerprint"
	"io"
)


func printSpectra(f *pcm.Frame, s fmt.Stringer, verbose bool) {
	if !verbose {
		return
	}

	fmt.Printf("[%4d:%6.2f] %s\n", f.BlockId(), f.Timestamp(), s)
}

func dumpPeaks(f *pcm.Frame, s spectral.Spectra, verbose bool) {
	s = s.Filter(
		func(freq, pwr float64) bool {
			return freq >= fingerprint.LOWER_FREQ_CUTOFF && freq <= fingerprint.UPPER_FREQ_CUTOFF && pwr > 10
		})

	s = s.Maxima()
	s = s.ByPxx()
	s = s.Tail(6)

	printSpectra(f, s, true)
}

func dumpBands(f *pcm.Frame, s spectral.Spectra, verbose bool) {
	fp := fingerprint.NewBandedprint(fingerprint.SAMPLE_RATE, s)

	printSpectra(f, fp, true)
}

func dumpFiles(filenames []string, analyser spectral.Analyser, optVerbose bool) (err error) {

	for _, filename := range filenames {
		fmt.Printf("Dumping %s...\n", filename)
		stream, err := pcm.NewFileStream(filename, fingerprint.SAMPLE_RATE, fingerprint.BLOCK_SIZE)
		if (err != nil) {
			return err
		}

		err = dumpStream(filename, stream, analyser, optVerbose)

		stream.Close()
	}

	return nil
}

func dumpStream(filename string, stream pcm.Reader, analyser spectral.Analyser, optVerbose bool) (error) {
	fnum := 0
	duration := 5

	for {
		frame, err := stream.Read()
		if (err != nil) {
			if (err == io.EOF || err == io.ErrUnexpectedEOF) {
				break
			}
			return err
		}

		fnum++
		spectra := analyser(frame.AsFloat64(), fingerprint.SAMPLE_RATE, fingerprint.NFFT, fingerprint.NOVERLAP, fingerprint.DB_SCALING)

		//dumpPeaks(frame, spectra, optVerbose)

		dumpBands(frame, spectra, optVerbose)

		if fnum >= fingerprint.BLOCKS_PER_SECOND * duration {
			break
		}
	}

	return nil
}


func main() {
	var optAnalyser string
	var optSeconds int
	var optVerbose bool
	var analyser spectral.Analyser

	flag.BoolVar(&optVerbose, "verbose", false, "Verbose output of spectral analysis data")
	flag.StringVar(&optAnalyser, "analyser", "pwelch", "Spectral analyser to use (pwelch | bespoke)")
	flag.IntVar(&optSeconds, "seconds", 0, "Limit scan to number of seconds")

	flag.Parse()

	switch optAnalyser {
	case "bespoke":
		analyser = spectral.Amplitude
	case "pwelch":
		analyser = spectral.Pwelch
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

	err := dumpFiles(filenames, analyser, optVerbose)
	if err != nil {
		log.Fatalf("Fatal Error dumping: %s", err)
	}

}
