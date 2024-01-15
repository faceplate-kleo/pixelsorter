package core

import (
    "fmt"
    "image"
    "image/color"
    "image/color/palette"
    "image/gif"
    "image/draw"
    "image/png"
    "log"
    "os"
    "math"
    "math/rand"
    "sort"
    "strings"
    "sync"


    "github.com/faceplate-kleo/pixelsorter/lib/wave"
    "github.com/faceplate-kleo/pixelsorter/lib/masks"
    "github.com/faceplate-kleo/pixelsorter/lib/nrgbautil"
    f "github.com/faceplate-kleo/pixelsorter/lib/flags"
    psmath "github.com/faceplate-kleo/pixelsorter/lib/math"

)

func SortNrgbaImage(
        imData_nrgb *image.NRGBA, 
        threshold int, 
        scalar float64, 
        noiseFactor int, 
        direction, maskInPath string, 
        signal []int, 
        mask *image.NRGBA,
        flags f.Flags,
    ) (*image.NRGBA, *image.NRGBA) {
    direction = strings.ToLower(direction)
    if direction == "up" {
        imData_nrgb = nrgbautil.RotateNrgba(imData_nrgb, 1)
    } else if direction == "down" {
        imData_nrgb = nrgbautil.RotateNrgba(imData_nrgb, 3)
    } else if direction == "left" {
        imData_nrgb = nrgbautil.FlipNrgba(imData_nrgb, true)
    }
    if maskInPath == "" {
        if mask == nil {
            mask = masks.CreateContrastMask(imData_nrgb, uint8(threshold), flags)
        }
    } else {
        mask = masks.ReadContrastMask(maskInPath, imData_nrgb.Bounds())
    }
    sorted := CreateSortedFromMask(imData_nrgb, mask, scalar, noiseFactor, signal, flags)

    if direction == "up" {
        imData_nrgb = nrgbautil.RotateNrgba(sorted, 3)
        sorted = nrgbautil.RotateNrgba(sorted, 3)
    } else if direction == "down" {
        imData_nrgb = nrgbautil.RotateNrgba(sorted, 1)
        sorted = nrgbautil.RotateNrgba(sorted, 1)
    } else if direction == "left" {
        imData_nrgb = nrgbautil.FlipNrgba(imData_nrgb, true)
        sorted = nrgbautil.FlipNrgba(sorted , true)
    }

    return sorted, mask 
}

func WaveAnimationFromSingleFrame(
        imData *image.NRGBA, 
        wavPath, maskPath, outPath, direction string, 
        threshold, noisefactor, framerate, num_buckets int, 
        scalar float64, 
        flags f.Flags,
    ) {
    waveStack, numFrames := wave.CreateWaveStack(wavPath, framerate, num_buckets)
    delay := int(1.0 / float64(framerate) * 100.0)

    res := imData.Bounds() 
    resY := res.Dy()

    paletted_anim := make([]*image.Paletted, numFrames) 
    raw_delay := make([]int, numFrames)

    //huge time save to do this only one time
    //TODO: make the rotation literally ANY less hacky
    mask_copy := image.NewNRGBA(imData.Bounds())
    draw.Draw(mask_copy, mask_copy.Rect, imData, imData.Bounds().Min, draw.Over)
    direction = strings.ToLower(direction)
    if direction == "up" {
        mask_copy = nrgbautil.RotateNrgba(mask_copy, 1)
    } else if direction == "down" {
        mask_copy = nrgbautil.RotateNrgba(mask_copy, 3)
    } else if direction == "left" {
        mask_copy = nrgbautil.FlipNrgba(mask_copy, true)
    }
    var master_mask *image.NRGBA 
    if maskPath == "" {
        master_mask = masks.CreateContrastMask(mask_copy, uint8(threshold), flags)
    } else {
        master_mask = masks.ReadContrastMask(maskPath, imData.Bounds())
    }

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

    var wg sync.WaitGroup
    for frame := 0; frame < numFrames; frame++ { 
        imData_copy := image.NewNRGBA(imData.Bounds())
        draw.Draw(imData_copy, imData_copy.Rect, imData, imData.Bounds().Min, draw.Over)
        wg.Add(1)
        go func(frame, resY int, imData *image.NRGBA) { 
            defer wg.Done()
            signal := make([]int, resY)
            for col := 0; col < resY; col++ {
                this_bucket := int((float64(col) / float64(resY)) * float64(num_buckets))
                amplitude := waveStack[frame][this_bucket]
                amplitude = int(float64(amplitude) / float64(max_amp) * float64(resY))
                signal[col] = amplitude
            }
            sorted, _ := SortNrgbaImage(imData, threshold, scalar, noisefactor, direction, "", signal, master_mask, flags)

            if flags.WRITE_FRAMES {
                frameID := "FRAME_" + fmt.Sprint(frame)
                fileout := "./frames/" + frameID + ".png"
                fileobj, err := os.Create(fileout)
                if err != nil {
                    log.Fatal(err)
                }
                png.Encode(fileobj, sorted)
            } else {
                frame_img := image.NewPaletted(res, palette.Plan9)
                draw.Draw(frame_img, frame_img.Rect, sorted, sorted.Bounds().Min, draw.Over)
                paletted_anim[frame] = frame_img
                raw_delay[frame] = delay
            }
        }(frame, resY, imData_copy)
    }
    if !flags.WRITE_FRAMES {
        wg.Wait()
        outGif := &gif.GIF{}
        outGif.Image = paletted_anim
        outGif.Delay = raw_delay 
        
        giffile, err := os.Create(outPath)
        if err != nil {
            log.Fatal(err)
        }
        gif.EncodeAll(giffile, outGif)
        giffile.Close()
    }
}

func CreateSortedFromMask(
        imData image.Image, 
        mask *image.NRGBA, 
        scalar float64, 
        noiseFactor int, 
        signal []int,
        flags f.Flags,
    ) *image.NRGBA {
    output := image.NewNRGBA(imData.Bounds())
    horizontal_domain := mask.Bounds().Dx()
    vertical_domain := mask.Bounds().Dy()

    outer_bound := vertical_domain 
    inner_bound := horizontal_domain

    j_written := make(map[int]bool)

    for i := 0; i < outer_bound; i++ {
        j_written = make(map[int]bool)
        for j := 0; j < inner_bound; j++ {
            if masks.ColorIsWhite(mask.At(j,i)){
                adjusted_j := j 
                span_x, span_y := masks.GetMaskSpan(mask, j, i, horizontal_domain, vertical_domain)            

                noiseAmt := 0.0
                if noiseFactor != 0 {
                    if noiseFactor > 0 {
                        noiseAmt = float64(rand.Intn(noiseFactor))
                    } else {
                        pos_noise := noiseFactor * -1
                        half_noise := pos_noise / 2 

                        noiseRaw := rand.Intn(pos_noise)
                        noiseAmt = float64(half_noise - noiseRaw)
                        if noiseAmt < 0 {
                            adjusted_j = psmath.IntMax (j + int(noiseAmt), 0)
                        }
                    }
                }


                signal_amt := 0.0
                if signal != nil {
                    signal_amt = psmath.SampleSignal(i, outer_bound, len(signal), signal)
                }

                float_j := float64(adjusted_j)
                float_span := float64(span_x - j)
                noised_span := float_span + noiseAmt
                final_span := math.Max(0.0, noised_span + signal_amt)
                calculated_domain := int(float_j + final_span * scalar)

                desired_span := psmath.IntMin(calculated_domain, horizontal_domain-1)
                if flags.MASK_DEBUG {
                    desired_span = horizontal_domain
                }
                if flags.CLEAN {
                    desired_span = span_x 
                }

                SortSpan(imData, adjusted_j, i, desired_span, span_y, output, flags)
                for x := adjusted_j; x < desired_span; x++ {
                    j_written[x] = true
                }
                j = desired_span

            } else {
                if !j_written[j] {
                    output.Set(j, i, imData.At(j,i))
                }
            }
        }
    }


    return output
}

func SortSpan(imData image.Image, start_x, start_y, end_x, end_y int, output *image.NRGBA, flags f.Flags) {
    //create a fast-sortable slice
    n := end_x - start_x + 1 
    toSort := make([]color.Color, n)
    for i := 0; i < n; i++ {
        toSort[i] = imData.At(start_x + i, start_y)
    }
    //sort it 

    sorted := psmath.SpanMergesort(toSort, flags)

    //write the slice back out 
   
    spanColor := psmath.GetRandomColor()
    
    for j := 0; j < n; j++ {
        colorToWrite := sorted[j]
        
        if flags.DEBUG {
            colorToWrite = spanColor
        }
        if flags.MASK_DEBUG {
            r, _, _, _ := colorToWrite.RGBA()
            cr := uint8(r)
            
            colorToWrite = color.RGBA{cr, cr, cr, 255}
        }
        output.Set(start_x + j, start_y, colorToWrite)
    }
}
