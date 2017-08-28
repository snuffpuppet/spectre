# spectre
Audio fingerprinting in Go. 
Service side code for both fingerprinting and recognising audio snippets from larger audio files. 
Part of a larger project to recognise movie sound tracks to sync subtitles for the hard of hearing. Currently in developemnt.

There are four commands:

### sp_listen
Scan the files on the comand line to generate fingerprints and then listen to the microphone and print out any matches

### sp_record
Listen to the microphone and dump the raw audio data to the output file listed on the command line. Uses signed 16bit.

### sp_dump
Generate fingerprints for the listed audio files on the command line and print out fingerprinting info for a limited chunk of data

### sp_lookup
Match an audio file using fingerprints with others given on the command line. This allows not having to use the microphone each
time you want to test the fingerprinting algorythm. Use sp_record to capture the microphone audio, convert that to a wav file
and use it as input to sp_lookup with the original file as one of the match files.

## Current State
The current state of the project uses simple spectral analysis and peak analysis to generate fingerprints. The stronger signals
in the spectral analysis are pulled out and hashed to form a fingerprint. This technique is actually not as effective as many
articles written on the subect seem to indicate.

Part of the problem is peaks in music file that are out of the sensitivity range of either the laptop / mobile microphone 
or speakers. A frequency filter which improves things and increases hit rate but most of the fingerprints still do not match.

## Next Steps
It seems that fingerprinting a 10ms frame by picking the strongest frequencies is not a good matching strategey, especially for film soundtracks with a lot of voice. Using Dejavu's strategy of picking strong frequencies in a 2d array of multiple slices of time will work better

