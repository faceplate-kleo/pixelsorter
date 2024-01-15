package gif

import (
    "fmt"
    "image"
    "image/color"
    "image/color/palette"
    "image/gif"
    "image/draw"
    "log"
    "os"
    "sort"


    "github.com/faceplate-kleo/pixelsorter/lib/wave"
    "github.com/faceplate-kleo/pixelsorter/src/core"
    
    f "github.com/faceplate-kleo/pixelsorter/lib/flags"
)

func AnimationFromSingleFrame(
        imData_nrgb *image.NRGBA, 
        inPath, outPath, direction string, 
        threshold, noiseFactor, frames int, 
        scalar float64,
        flags f.Flags,
    ) {
    raw_anim := make([]*image.Paletted, frames) 
    raw_delay := make([]int, frames)
    for frame := 0; frame < frames; frame++ {
        sorted, _ := core.SortNrgbaImage(imData_nrgb, threshold, scalar, noiseFactor, direction, "", nil, nil, flags)
        paletted := image.NewPaletted(sorted.Bounds(), palette.WebSafe)
        draw.Draw(paletted, paletted.Rect, sorted, sorted.Bounds().Min, draw.Over)

        raw_anim[frame] = paletted
        raw_delay[frame] = 0
    }

    outGif := &gif.GIF{}

    outGif.Image = raw_anim
    outGif.Delay = raw_delay


    giffile, err := os.Create("./out.gif")
    if err != nil {
        log.Fatal(err)
    }
    gif.EncodeAll(giffile, outGif)
    giffile.Close()
}

func GifVisualization(inPath, outPath string, framerate, num_buckets int) {
    waveStack, numFrames := wave.CreateWaveStack("./resources/test4.wav", framerate, num_buckets)
    fmt.Println(len(waveStack[0]))
    delay := int(1.0 / float64(framerate) * 100.0)
    res := 512
    std_bounds := image.Rect(0,0,res,res)

    raw_anim := make([]*image.Paletted, numFrames) 
    raw_delay := make([]int, numFrames)

    max_amp := -1 
    for frame := 0; frame < numFrames; frame++ {
        buckets_clone := make([]int, num_buckets)
        copy(buckets_clone, waveStack[frame])
        sort.Ints(buckets_clone)
        frame_peak := buckets_clone[num_buckets-1]
        if frame_peak > max_amp {
            max_amp = frame_peak
        }
    }

    for frame := 0; frame < numFrames; frame++ {
        frame_img := image.NewPaletted(std_bounds, palette.Plan9)
        fmt.Println(waveStack[frame])
        for col := 0; col < res; col++ {
            this_bucket := int((float64(col) / float64(res)) * float64(num_buckets))
            amplitude := waveStack[frame][this_bucket]
            amplitude = int(float64(amplitude) / float64(max_amp) * float64(res))
            for row := 0; row < amplitude; row++ {
                frame_img.Set(col, res-row, color.White)
            }
            for row := amplitude; row < res; row++ {
                hue := color.NRGBA{0,0,0,0}
                if frame == 0 {
                    hue = color.NRGBA{0,255,0,255}
                }
                frame_img.Set(col, res-row, hue)
            }
        }
        raw_anim[frame] = frame_img
        raw_delay[frame] = delay
    }
    outGif := &gif.GIF{}
    outGif.Image = raw_anim
    outGif.Delay = raw_delay 
    
    giffile, err := os.Create(outPath)
    if err != nil {
        log.Fatal(err)
    }
    gif.EncodeAll(giffile, outGif)
    giffile.Close()
}
