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

type band struct {
	start, end int
}

type frange struct {
	start, end float64
}

type bands struct {
	ranges []band
	freqs  []frange
	fs     int
}

func newBands(fs int) bands {
	b := []band {
		band{ 30,  40  },
		band{ 40,  80  },
		band{ 80,  120 },
		band{ 120, 180 },
		band{ 180, 300 },
		band{ 300, 512 },
	}

	freqStep := float64(fs) / 2 / 512
	r := make([]frange, len(b))
	for i, v := range b {
		r[i] = frange{float64(v.start) * freqStep, float64(v.end) * freqStep}
	}

	return bands{
		ranges: b,
		freqs:  r,
		fs:     fs,
	}
}

type highPoint struct {
	freq, pxx float64
}

type highPoints struct {
	points []highPoint
	fbands bands
}

func newHighPoints(fs int) highPoints {
	fb := newBands(fs)
	return highPoints{
		fbands: fb,
		points: make([]highPoint, len(fb.freqs)),
	}
}

func (hp highPoints) add(f, pxx float64) {
	for i, v := range hp.fbands.freqs {
		if f >= v.start && f < v.end {
			if pxx > hp.points[i].pxx {
				hp.points[i].freq = f
				hp.points[i].pxx = pxx
				break
			}
		}
	}
}

func (hp highPoints) String() (s string) {
	s = ""
	for i := range hp.points {
		s = fmt.Sprintf("%s[%d] %7.2f(%6.2f) ", s, i, hp.points[i].freq, hp.points[i].pxx)
	}

	return
}

func (hp highPoints) header() (s string) {
	s = ""
	for i, v := range hp.fbands.freqs {
		s = fmt.Sprintf("%s[%d] %7.2f - %7.2f ", s, i, v.start, v.end)
	}

	return
}

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
	hp := newHighPoints(fingerprint.SAMPLE_RATE)

	for i := range s.Freqs {
		hp.add(s.Freqs[i], s.Pxx[i])
	}

	printSpectra(f, hp, true)
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
	hp := newHighPoints(fingerprint.SAMPLE_RATE)
	fmt.Println(hp.header())

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
