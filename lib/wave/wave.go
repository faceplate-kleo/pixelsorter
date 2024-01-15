package wave

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"
	"sort"

	"github.com/mjibson/go-dsp/fft"
)

func ReadWav(filepath string, frame_rate int) ([][][]byte, int) {
	wavfile, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err)
	}
	defer wavfile.Close()

	header_buf := make([]byte, 44)

	n, err := wavfile.Read(header_buf)

	if n != 44 {
		log.Fatal("Fatal: .wav read failure")
	}

	//byte info from https://docs.fileformat.com/audio/wav/

	fmt.Println("Reading WAV...")
	mark := string(header_buf[0:5])                            // RIFF File marker
	fsiz := int(binary.LittleEndian.Uint32(header_buf[4:8]))   // Filesize - 8 bytes
	ftyp := string(header_buf[8:12])                           // File type header (just "WAVE")
	chnk := string(header_buf[12:16])                          // Format chunk marker (just "fmt ", space intentional)
	flen := int(binary.LittleEndian.Uint32(header_buf[16:20])) // Format data length
	pcmf := int(binary.LittleEndian.Uint16(header_buf[20:22])) // Type of format (1 = PCM)
	chnl := int(binary.LittleEndian.Uint16(header_buf[22:24])) // Number of channels
	rate := int(binary.LittleEndian.Uint32(header_buf[24:28])) // Sample rate (Hertz)
	nrte := int(binary.LittleEndian.Uint32(header_buf[28:32])) // (SampleRate * BitsPerSample * Channels) / 8
	widt := int(binary.LittleEndian.Uint16(header_buf[32:34])) // (BitsPerSample * Channels)/8
	btps := int(binary.LittleEndian.Uint16(header_buf[34:36])) // Bits per sample
	dhed := string(header_buf[36:40])                          // Data chunk header (just "data")
	dsiz := int(binary.LittleEndian.Uint32(header_buf[40:44])) // Size of the data section

	fmt.Println(mark, fsiz, ftyp, chnk, flen, pcmf, chnl, rate, nrte, widt, btps, dhed, dsiz)

	data := make([]byte, fsiz)
	n, err = wavfile.ReadAt(data, 43)

	num_samples := (dsiz / widt)

	bytes_per_sample := btps / 8
	samples := make([][]byte, num_samples)

	ptr := 0
	for i := 0; i < num_samples; i++ {

		samples[i] = data[ptr : ptr+(bytes_per_sample*chnl)]
		ptr += bytes_per_sample * chnl
	}
	playtime := float32(num_samples) / float32(rate)
	fmt.Println("PLAYTIME: ", playtime)

	samples_per_frame := rate / frame_rate
	num_frames := int(playtime * float32(frame_rate))
	fmt.Println(num_samples, "samples at", bytes_per_sample, "bytes per sample")
	fmt.Println(num_frames, "frames at", samples_per_frame, "samples per frame")

	frame_samples := make([][][]byte, num_frames)
	ptr = 0
	for frame := 0; frame < num_frames; frame++ {
		frame_samples[frame] = samples[ptr : ptr+samples_per_frame]
		ptr += samples_per_frame
	}

	return frame_samples, rate
}

func SampleWav(wavData [][][]byte, index, channel int) []byte {
	sampleWidth := len(wavData[index])
	outdata := make([]byte, sampleWidth)

	for i := 0; i < sampleWidth; i++ {
		outdata[i] = wavData[index][i][channel]
	}
	return outdata
}

func ToLogarithmic(n, max, exp int) int {
	log := int(float64(max) * math.Log10(float64(n)/float64(exp)))
	if log < 0 {
		log = 0
	}
	return log
}

func BucketsFromSample(sample []byte, num_buckets, sample_rate int) []int {
	outdata := make([]int, num_buckets)
	floats := make([]float64, len(sample))
	for i := 0; i < len(sample); i++ {
		floats[i] = float64(sample[i])
	}

	transformed := fft.FFTReal(floats)
	transformed = transformed[0 : (len(transformed)/2)-1]

	reals := make([]float64, len(transformed)-1)
	for j := 0; j < len(transformed)-1; j++ {
		reals[j] = math.Sqrt(math.Pow(real(transformed[j+1]), 2) + math.Pow(imag(transformed[j+1]), 2))
	}

	bucket_width := len(reals) / num_buckets
	if bucket_width < 1 {
		bucket_width = 1
	}

	for x := 0; x < num_buckets; x++ {
		//ind := int(math.Log(float64(x * bucket_width)))
		ind := x * bucket_width
		if ind < 0 {
			ind = 0
		} else if ind > len(reals) {
			ind = len(reals)
		}
		slice_end := ind + bucket_width
		data_slice := reals[ind:slice_end]
		//mean or max?

		//max:
		slice_copy := make([]float64, len(data_slice))
		copy(slice_copy, data_slice)
		sort.Float64s(slice_copy)
		outdata[x] = int(slice_copy[len(data_slice)-1])
	}

	return outdata

}

func CreateWaveStack(waveIn string, framerate, num_buckets int) ([][]int, int) {
    wavData, sampleRate := ReadWav(waveIn, framerate)
    output := make([][]int, len(wavData))

    all_buckets := make([][]int, len(wavData))
    for samp := 0; samp < len(wavData); samp++ {
        sample := SampleWav(wavData, samp, 0)
        buckets := BucketsFromSample(sample, num_buckets, sampleRate)
        all_buckets[samp] = buckets
    }

    for frame := 0; frame < len(wavData); frame++ {
        output[frame] = make([]int, num_buckets)
        for col := 0; col < num_buckets; col++ {
            val_adjusted := all_buckets[frame][col]
            output[frame][col] = val_adjusted
        }
    }

    return output, len(wavData)
}

