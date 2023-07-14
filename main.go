package main

import (
    "github.com/faceplate-kleo/pixelsorter/lib"

	"flag"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/gif"
    "image/draw"
	"image/png"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
    "sort"
    "sync"
)

var DEBUG bool = false 
var MASK_DEBUG bool = false 
var SOURCE_DEBUG bool = false

var ANIM = false
var WRITE_FRAMES = false

var DESCEND bool = false 
var CRUSH bool = false 
var CLEAN bool = false 
var INVERT bool = false

var MEAN_COMPARE bool = true
var GRAY_RED_COMPARE bool = false 

func load_image(path string) *image.NRGBA {
    
    imgFile, err := os.Open(path)
    if err != nil {
        log.Fatal(err)
    }
    defer imgFile.Close()

    imData, _, err := image.Decode(imgFile)
    if err != nil {
        fmt.Println(err)
    }

    if _, ok := imData.(*image.NRGBA); ok {
        return imData.(*image.NRGBA)
    } else {
        outbuf := image.NewNRGBA(imData.Bounds())
        draw.Draw(outbuf, outbuf.Rect, imData, imData.Bounds().Min, draw.Over)
        return outbuf 
    }
}

func write_file (imData *image.NRGBA, path string) {
    out, err := os.Create(path)
    if err != nil {
        log.Fatal(err)
    }
    png.Encode(out, imData.SubImage(imData.Rect))
    out.Close()
}

func calculate_luminance(rgbcolor color.Color) uint8 {
    var int_r, int_g, int_b, _ = rgbcolor.RGBA()
    r := float32(int_r)
    g := float32(int_g)
    b := float32(int_b)
    return uint8((0.2126 * r) + (0.7152 * g) + (0.0722 * b))
}

func create_contrast_mask(imData image.Image, threshold uint8) *image.NRGBA {
    bounds := imData.Bounds()
    mask := image.NewNRGBA(bounds)

    max_x := bounds.Dx() 
    max_y := bounds.Dy()

    for i := 0; i < max_x; i++ {
        for j := 0; j < max_y; j++ {
            currentColor := imData.At(i,j)
            //comparator := calculate_luminance(currentColor)
            comp_u32, _, _, _ := currentColor.RGBA()
            comparator := uint8(comp_u32)
            outColor := color.White 
            if INVERT {
                outColor = color.Black
            }
            if comparator < threshold {
                if !INVERT {
                    outColor = color.Black 
                } else {
                    outColor = color.White 
                }
            }

            if MASK_DEBUG {
                outColor = color.White
            }

            mask.Set(i, j, outColor)
        }
    }

    return mask
}

func read_contrast_mask(maskInPath string, bounds image.Rectangle) *image.NRGBA {
    output := image.NewNRGBA(bounds)

    //open and read file 
    maskFile, err := os.Open(maskInPath)
    if err != nil {
        log.Fatal(err)
    }
    defer maskFile.Close()

    //truncate mask 
    maskData, _, err := image.Decode(maskFile)
    if err != nil {
        log.Fatal(err)
    }

    bound_x := bounds.Dx() 
    bound_y := bounds.Dy()
    mask_x := maskData.Bounds().Dx() 
    mask_y := maskData.Bounds().Dy()

    final_x := intMin(bound_x, mask_x)
    final_y := intMin(bound_y, mask_y)

    //copy to output buffer 

    for i := 0; i < final_x; i++ {
        for j := 0; j < final_y; j++ {
            output.Set(i, j, maskData.At(i, j))
        }
    }

    return output
}

func color_is_white(toComp color.Color) bool {
    cr, cg, cb, ca := toComp.RGBA()
    return (cr+cg+cb+ca) == 65535*4
}

func color_is_init(toComp color.Color) bool {
    cr, cg, cb, ca := toComp.RGBA()
    return (cr+cg+cb+ca) == 0
}

func get_mask_span(mask *image.NRGBA, start_x , start_y, max_x, max_y int) (int, int) {
    if !color_is_white(mask.At(start_x, start_y)) {
        return start_x, start_y
    }

    relevant_bound := max_x
    relevant_constant := start_y

    for i := start_x; i < relevant_bound; i++ {
        if !color_is_white(mask.At(i, relevant_constant)) {
           return i, start_y 
        }
    }

    return relevant_bound, start_y
}

func mean_compare(colorA, colorB color.Color) bool {
    
    ar, ag, ab, _ := colorA.RGBA()
    br, bg, bb, _ := colorB.RGBA()
    mean_a := (ar + ag + ab) / 3 
    mean_b := (br + bg + bb) / 3 
    if DESCEND {
        return mean_a > mean_b
    } else {
        return mean_a < mean_b
    }
}

func red_compare(colorA, colorB color.Color) bool {
    ar, _, _, _ := colorA.RGBA()
    br, _, _, _ := colorB.RGBA()
    if DESCEND{
        return ar > br
    } else {
        return ar < br
    }
}

func merge(a []color.Color, iLeft, iRight, iEnd int, b []color.Color) []color.Color {
    i := iLeft 
    j := iRight 

    for k := iLeft; k < iEnd; k++ {
        comp := false 
        if i < len(a) && j < len(a) {
            if MEAN_COMPARE {
                comp = mean_compare(a[i], a[j])
            }
            if GRAY_RED_COMPARE {
               comp = red_compare(a[i], a[j]) 
            }
        }

        if i < iRight && (j >= iEnd || comp) {
            b[k] = a[i]
            i++
        } else {
            if CRUSH {
                b[i] = a[j]
            } else {
                b[k]= a[j]
            }
            j++
        }
    }
    return b

}

func intMin(a, b int) int {
    if a < b {
        return a 
    } else {
        return b
    }
}
func intMax(a, b int) int {
    if a > b {
        return a 
    } else {
        return b
    }
}

func span_mergesort(span []color.Color) []color.Color {
    n := len(span)
    workArr := make([]color.Color, n)

    copy(workArr, span)

    for width := 1; width < n; width = 2*width {
        for i := 0; i < n; i = i + 2*width {
            workArr = merge(span, i, intMin(i+width, n), intMin(i+2*width, n), workArr)
        }
        copy(span, workArr)
    }

    return span

}

func get_random_color() color.RGBA {
    ra := uint8(rand.Intn(255))
    rg := uint8(rand.Intn(255))
    rb := uint8(rand.Intn(255))

    return color.RGBA{ra, rg, rb, 255}
    
}

func sort_span(imData image.Image, start_x, start_y, end_x, end_y int, output *image.NRGBA) {
    //create a fast-sortable slice
    n := end_x - start_x + 1 
    toSort := make([]color.Color, n)
    for i := 0; i < n; i++ {
        toSort[i] = imData.At(start_x + i, start_y)
    }
    //sort it 

    sorted := span_mergesort(toSort)

    //write the slice back out 
   
    spanColor := get_random_color()
    
    for j := 0; j < n; j++ {
        colorToWrite := sorted[j]
        
        if DEBUG {
            colorToWrite = spanColor
        }
        if MASK_DEBUG {
            r, _, _, _ := colorToWrite.RGBA()
            cr := uint8(r)
            
            colorToWrite = color.RGBA{cr, cr, cr, 255}
        }
        output.Set(start_x + j, start_y, colorToWrite)
    }
}

func sample_signal(sourceIndex, domain, signalSize int, signal []int) float64 {
    ratio := float64(signalSize) / float64(domain) 
    //ratio := float64(domain) / float64(signalSize)
    adjusted_index := int(float64(sourceIndex) * ratio)

    adjusted_index = intMax(adjusted_index, 0)
    adjusted_index = intMin(adjusted_index, signalSize-1)

    return float64(signal[adjusted_index])
}

func create_sorted_from_mask(imData image.Image, mask *image.NRGBA, scalar float64, noiseFactor int, signal []int) *image.NRGBA {
    output := image.NewNRGBA(imData.Bounds())
    horizontal_domain := mask.Bounds().Dx()
    vertical_domain := mask.Bounds().Dy()

    outer_bound := vertical_domain 
    inner_bound := horizontal_domain

    j_written := make(map[int]bool)

    for i := 0; i < outer_bound; i++ {
        j_written = make(map[int]bool)
        for j := 0; j < inner_bound; j++ {
            if color_is_white(mask.At(j,i)){
                adjusted_j := j 
                span_x, span_y := get_mask_span(mask, j, i, horizontal_domain, vertical_domain)            

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
                            adjusted_j = intMax (j + int(noiseAmt), 0)
                        }
                    }
                }


                signal_amt := 0.0
                if signal != nil {
                    signal_amt = sample_signal(i, outer_bound, len(signal), signal)
                }

                float_j := float64(adjusted_j)
                float_span := float64(span_x - j)
                noised_span := float_span + noiseAmt
                final_span := math.Max(0.0, noised_span + signal_amt)
                calculated_domain := int(float_j + final_span * scalar)

                desired_span := intMin(calculated_domain, horizontal_domain-1)
                if MASK_DEBUG {
                    desired_span = horizontal_domain
                }
                if CLEAN {
                    desired_span = span_x 
                }

                sort_span(imData, adjusted_j, i, desired_span, span_y, output)
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

func data_to_nrgba(imData image.Image) *image.NRGBA{
    out := image.NewNRGBA(imData.Bounds())
    max_x := imData.Bounds().Dx()
    max_y := imData.Bounds().Dy()

    for i := 0; i < max_x; i++ {
        for j := 0; j < max_y; j++ {
            if SOURCE_DEBUG {
                out.Set(i, j, get_random_color())
            } else {
                out.Set(i, j, imData.At(i, j)) 
            }
        }
    }

    return out
}

func reverse_rows_nrgba(imData *image.NRGBA) {
    for i := 0; i < imData.Rect.Dy(); i++ {
        a := 0 
        b := imData.Rect.Dx()

        for a < b {
            tmp := imData.At(a, i)
            imData.Set(a, i, imData.At(b, i))
            imData.Set(b, i, tmp)
            a++
            b--
        }
    }
}
func reverse_cols_nrgba(imData *image.NRGBA) {
    for i := 0; i < imData.Rect.Dx(); i++ {
        a := 0 
        b := imData.Rect.Dy()

        for a < b {
            tmp := imData.At(a, i)
            imData.Set(i, a, imData.At(b, i))
            imData.Set(i, b, tmp)
            a++
            b--
        }
    }
}

func rotate_nrgba(imData *image.NRGBA, rotations int) *image.NRGBA {
    init_x := imData.Bounds().Dx()
    init_y := imData.Bounds().Dy()

    desired_x := init_x
    desired_y := init_y

    if rotations % 2 != 0 {
        desired_x = init_y 
        desired_y = init_x
    }

    output := image.NewNRGBA(image.Rect(0, 0, desired_x, desired_y))

    if rotations == 2 {

        for i := 0; i < desired_y; i++ {
            for j := 0; j < desired_x; j++ {
                output.Set(j, i, imData.At(j,i))
            }
        }
        reverse_rows_nrgba(output)
        reverse_cols_nrgba(output)
        return output
    }

    if rotations == 3 {
        reverse_rows_nrgba(imData)
    }

    if rotations % 2 != 0 {
        for i := 0; i < desired_y; i++ {
            for j := 0; j < desired_x; j++ {
                output.Set(j, i, imData.At(i,j))
            }
        }
    }

    if rotations == 1 {
        reverse_rows_nrgba(output)
    }


    if rotations == 3 {
        reverse_rows_nrgba(imData)
    }

    return output 
}

func flip_nrgba(imData *image.NRGBA, horizontal bool) *image.NRGBA {
    output := image.NewNRGBA(imData.Bounds())
    bound := imData.Bounds().Dy()
    altbound := imData.Bounds().Dx()

    if !horizontal {
        bound = imData.Bounds().Dx()
        altbound = imData.Bounds().Dy()
    }

    for i := 0; i < bound; i++ {
        for j := 0; j < altbound; j++ {
            output.Set(j, i, imData.At(j, i))        
        }
    }

    if horizontal {
        reverse_rows_nrgba(output)
    }

    return output 
}

func sort_nrgba_image(imData_nrgb *image.NRGBA, threshold int, scalar float64, noiseFactor int, direction, maskInPath string, signal []int, mask *image.NRGBA) (*image.NRGBA, *image.NRGBA) {
    direction = strings.ToLower(direction)
    if direction == "up" {
        imData_nrgb = rotate_nrgba(imData_nrgb, 1)
    } else if direction == "down" {
        imData_nrgb = rotate_nrgba(imData_nrgb, 3)
    } else if direction == "left" {
        imData_nrgb = flip_nrgba(imData_nrgb, true)
    }
    if maskInPath == "" {
        if mask == nil {
            mask = create_contrast_mask(imData_nrgb, uint8(threshold))
        }
    } else {
        mask = read_contrast_mask(maskInPath, imData_nrgb.Bounds())
    }
    sorted := create_sorted_from_mask(imData_nrgb, mask, scalar, noiseFactor, signal)

    if direction == "up" {
        imData_nrgb = rotate_nrgba(sorted, 3)
        sorted = rotate_nrgba(sorted, 3)
    } else if direction == "down" {
        imData_nrgb = rotate_nrgba(sorted, 1)
        sorted = rotate_nrgba(sorted, 1)
    } else if direction == "left" {
        imData_nrgb = flip_nrgba(imData_nrgb, true)
        sorted = flip_nrgba(sorted , true)
    }

    return sorted, mask 
}

func wave_animation_from_single(imData *image.NRGBA, wavPath, maskPath, outPath, direction string, threshold, noisefactor, framerate, num_buckets int, scalar float64, frames_out bool) {
    waveStack, numFrames := create_wave_stack(wavPath, framerate, num_buckets)
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
        mask_copy = rotate_nrgba(mask_copy, 1)
    } else if direction == "down" {
        mask_copy = rotate_nrgba(mask_copy, 3)
    } else if direction == "left" {
        mask_copy = flip_nrgba(mask_copy, true)
    }
    var master_mask *image.NRGBA 
    if maskPath == "" {
        master_mask = create_contrast_mask(mask_copy, uint8(threshold))
    } else {
        master_mask = read_contrast_mask(maskPath, imData.Bounds())
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
            sorted, _ := sort_nrgba_image(imData, threshold, scalar, noisefactor, direction, "", signal, master_mask)

            if frames_out {
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
    if !frames_out {
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

func animation_from_single(imData_nrgb *image.NRGBA, inPath, outPath, direction string, threshold, noiseFactor, frames int, scalar float64) {
    raw_anim := make([]*image.Paletted, frames) 
    raw_delay := make([]int, frames)
    for frame := 0; frame < frames; frame++ {
        sorted, _ := sort_nrgba_image(imData_nrgb, threshold, scalar, noiseFactor, direction, "", nil, nil)
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

func create_wave_stack(waveIn string, framerate, num_buckets int) ([][]int, int) {
    wavData, sampleRate := lib.ReadWav(waveIn, framerate)
    output := make([][]int, len(wavData))

    all_buckets := make([][]int, len(wavData))
    for samp := 0; samp < len(wavData); samp++ {
        sample := lib.SampleWav(wavData, samp, 0)
        buckets := lib.BucketsFromSample(sample, num_buckets, sampleRate)
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

func gif_visualization(inPath, outPath string, framerate, num_buckets int) {
    waveStack, numFrames := create_wave_stack("./resources/test4.wav", framerate, num_buckets)
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

func main() {
    inPath  := "./resources/skull2.png"
    outPath := "./out.png"
    maskInPath := "./mask.png"
    maskOutPath := "./mask.png"
    threshold := 110
    scalar := 2.0 
    noiseFactor := 0
    direction := "right"
    frames := 10
    wavein := ""
    framerate := 25
    buckets := 128

    flag.BoolVar(&CRUSH, "crush", false, "Crush the output (bug turned feature)")
    flag.BoolVar(&MASK_DEBUG, "mask_debug", false, "White-out the mask for debugging")
    flag.BoolVar(&SOURCE_DEBUG, "source_debug", false, "Replace the input data with random color noise for debugging")
    flag.BoolVar(&DEBUG, "span_debug", false, "Fill spans with random colors for debugging")
    flag.BoolVar(&DESCEND, "descend", false, "Sort pixels in descending order")
    flag.BoolVar(&CLEAN, "clean", false, "Limit sorting to only within mask, with no bleeding")
    flag.BoolVar(&INVERT, "invert", false, "Invert the contrast mask")
    flag.BoolVar(&MEAN_COMPARE, "mean_compare", true, "Base pixel comparisons on R+G+B/3")
    flag.BoolVar(&GRAY_RED_COMPARE, "red_compare", false, "Base pixel comparions on just R - defaults false, overrides mean_compare")
    flag.StringVar(&inPath, "in", "", "Path to file to sort - REQUIRED")
    flag.StringVar(&outPath, "out", "./sorted.png", "Path to output file")
    flag.StringVar(&maskOutPath, "mask_out", "", "Path to mask output file - does not write if unspecified")
    flag.StringVar(&maskInPath, "mask", "", "Path to mask input file - skips mask generation step")
    flag.IntVar(&threshold, "threshold", 110, "Red channel threshold for the contrast mask")
    flag.StringVar(&direction, "direction", "right", "Direction of sort smear (up, down, left, right)")
    flag.Float64Var(&scalar, "scalar", 3.0, "Scale factor of sort span sizing")
    flag.IntVar(&noiseFactor, "noise", 0, "Random noise span offset amount in pixels")

    flag.BoolVar(&ANIM, "anim", false, "Create a .gif animation")
    flag.BoolVar(&WRITE_FRAMES, "write_frames", false, "Write all frames generated by -anim as individual .pngs. Omitting this option will generate a .GIF instead")
    flag.IntVar(&frames, "frames", 10, "The number of frames to generate when -anim is enabled")
    flag.StringVar(&wavein, "wav", "", "Filepath of a .wav file")
    flag.IntVar(&framerate, "framerate", 25, "Desired framerate of output .GIF (Warning: values n for 100 % n != 0 will cause time drift with audio!)")
    flag.IntVar(&buckets, "buckets", 128, "The number of frequency bands to divide .wav signal into")


    flag.Parse()

    if inPath == "" {
        fmt.Println("FATAL: no input file specified! ( -in )")
        flag.Usage()
        return
    }
    if !ANIM {
        imData := load_image(inPath)
        imData_nrgb := data_to_nrgba(imData)
        sorted, mask := sort_nrgba_image(imData_nrgb, threshold, scalar, noiseFactor, direction, maskInPath, nil, nil)

        if maskOutPath != "" {
            write_file(mask, maskOutPath)
        }
        write_file(sorted, outPath)
    } else {
        imData := load_image(inPath)
        imData_nrgb := data_to_nrgba(imData)
        if wavein == "" {
            animation_from_single(imData_nrgb, inPath, outPath, direction, threshold, noiseFactor, frames, scalar)
        } else {
            if inPath == "" {
                gif_visualization(wavein, "./visualization.gif", framerate, buckets)
            }
            wave_animation_from_single(imData_nrgb, wavein, maskInPath, "./sorted.gif", direction, threshold, noiseFactor, framerate, buckets, scalar, WRITE_FRAMES)
        }
    }   
}
