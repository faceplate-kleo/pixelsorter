package lib

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
    "math"
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
    mark := string(header_buf[0:5])                             // RIFF File marker 
    fsiz := int(binary.LittleEndian.Uint32(header_buf[4:8]))    // Filesize - 8 bytes 
    ftyp := string(header_buf[8:12])                            // File type header (just "WAVE")
    chnk := string(header_buf[12:16])                           // Format chunk marker (just "fmt ", space intentional)
    flen := int(binary.LittleEndian.Uint32(header_buf[16:20]))  // Format data length
    pcmf := int(binary.LittleEndian.Uint16(header_buf[20:22]))  // Type of format (1 = PCM)
    chnl := int(binary.LittleEndian.Uint16(header_buf[22:24]))  // Number of channels
    rate := int(binary.LittleEndian.Uint32(header_buf[24:28]))  // Sample rate (Hertz)
    nrte := int(binary.LittleEndian.Uint32(header_buf[28:32]))  // (SampleRate * BitsPerSample * Channels) / 8
    widt := int(binary.LittleEndian.Uint16(header_buf[32:34]))  // (BitsPerSample * Channels)/8
    btps := int(binary.LittleEndian.Uint16(header_buf[34:36]))  // Bits per sample 
    dhed := string(header_buf[36:40])                           // Data chunk header (just "data")
    dsiz := int(binary.LittleEndian.Uint32(header_buf[40:44]))  // Size of the data section

    fmt.Println(mark, fsiz, ftyp, chnk, flen, pcmf, chnl, rate, nrte, widt, btps, dhed, dsiz)

    data := make([]byte, fsiz)
    n, err = wavfile.ReadAt(data,43)


    num_samples := (dsiz / widt) 
    bytes_per_sample := btps/8
    samples := make([][]byte, num_samples)

    
    ptr := 0
    for i := 0; i < num_samples; i++ {

        samples[i] = data[ptr:ptr+(bytes_per_sample)]
        ptr += bytes_per_sample
    }
    playtime := float32(num_samples) / float32(rate)
    fmt.Println("PLAYTIME: ", playtime)



    samples_per_frame := rate / frame_rate
    num_frames := int( playtime * float32(frame_rate))
    fmt.Println(num_frames, "frames at", samples_per_frame, "samples per frame")


    frame_samples := make([][][]byte, num_frames) 
    ptr = 0
    for frame := 0; frame < num_frames; frame++ {
        frame_samples[frame] = samples[ptr:ptr+samples_per_frame]
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

func BucketsFromSample(sample []byte, num_buckets, sample_rate int) []int{
    outdata := make([]int, num_buckets)
    floats := make([]float64, len(sample))
    for i := 0; i < len(sample); i++ {
        floats[i] = float64(sample[i])
    }


    transformed := fft.FFTReal(floats)

    reals := make([]float64, len(sample)-1)
    for j := 0; j < len(sample) -1; j++ {
        reals[j] = math.Abs(real(transformed[j+1]))
    }

    reals = reals[0:len(reals)/4]

    bucket_width := len(reals) / num_buckets

    for x := 0; x < num_buckets; x++ {
        ind := x * bucket_width
        data_slice := reals[ind:ind+bucket_width]
        //fmt.Println(data_slice)
        //mean or max?

        //max:
        slice_copy := make([]float64, len(data_slice))
        copy(slice_copy, data_slice)
        sort.Float64s(slice_copy)
        outdata[x] = int(slice_copy[len(data_slice)-1])
    }

    return outdata

}
