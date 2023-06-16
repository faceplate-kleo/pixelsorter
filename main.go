package main

import (
    "fmt"
    "image"
    "image/png"
    "image/color"
    "log"
    "os"
    "math/rand"
    "flag"
)

var DEBUG bool = false 
var MASK_DEBUG bool = false 
var SOURCE_DEBUG bool = false

var DESCEND bool = false 
var CRUSH bool = false 
var CLEAN bool = false 

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
    return imData.(*image.NRGBA)
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
            if comparator < threshold {
                outColor = color.Black 
            }

            if MASK_DEBUG {
                outColor = color.White
            }

            mask.Set(i, j, outColor)
        }
    
    }

    return mask
}

func color_is_white(toComp color.Color) bool {
    cr, cg, cb, ca := toComp.RGBA()
    return (cr+cg+cb+ca) == 65535*4
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

func create_sorted_from_mask(imData image.Image, mask *image.NRGBA) *image.NRGBA {
    output := image.NewNRGBA(imData.Bounds())
    horizontal_domain := mask.Bounds().Dx()
    vertical_domain := mask.Bounds().Dy()

    outer_bound := vertical_domain 
    inner_bound := horizontal_domain

    for i := 0; i < outer_bound; i++ {
        for j := 0; j < inner_bound; j++ {
            if color_is_white(mask.At(j,i)){
                
                span_x, span_y := get_mask_span(mask, j, i, horizontal_domain, vertical_domain)            

                //desired_span := intMin(span_x * 2, horizontal_domain)
                desired_span := intMin(j+((span_x - j)*3), horizontal_domain)
                if MASK_DEBUG {
                    desired_span = horizontal_domain
                }
                if CLEAN {
                    desired_span = span_x 
                }

                sort_span(imData, j, i, desired_span, span_y, output)
                j = desired_span

            } else {
                output.Set(j, i, imData.At(j,i))
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


func main() {
    inPath  := "./resources/skull2.png"
    outPath := "./out.png"
    maskPath := "./mask.png"
    threshold := 110

    flag.BoolVar(&CRUSH, "crush", false, "Crush the output (bug turned feature)")
    flag.BoolVar(&MASK_DEBUG, "mask_debug", false, "White-out the mask for debugging")
    flag.BoolVar(&SOURCE_DEBUG, "source_debug", false, "Replace the input data with random color noise for debugging")
    flag.BoolVar(&DESCEND, "descend", false, "Sort pixels in descending order")
    flag.BoolVar(&CLEAN, "clean", false, "Limit sorting to only within mask, with no bleeding")
    flag.BoolVar(&MEAN_COMPARE, "mean_compare", true, "Base pixel comparisons on R+G+B/3")
    flag.BoolVar(&GRAY_RED_COMPARE, "red_compare", false, "Base pixel comparions on just R - defaults false, overrides mean_compare")
    flag.StringVar(&inPath, "in", "", "Path to file to sort - REQUIRED")
    flag.StringVar(&outPath, "out", "./sorted.png", "Path to output file")
    flag.StringVar(&maskPath, "mask", "", "Path to mask output file - does not write if unspecified")
    flag.IntVar(&threshold, "threshold", 110, "Red channel threshold for the contrast mask")

    flag.Parse()

    if inPath == "" {
        fmt.Println("FATAL: no input file specified! ( -in )")
        return 
    }


    imData := load_image(inPath)
    imData_nrgb := data_to_nrgba(imData)
    mask := create_contrast_mask(imData_nrgb, uint8(threshold))
    sorted := create_sorted_from_mask(imData_nrgb, mask)

    if maskPath != "" {
        write_file(mask, maskPath)
    }
    write_file(sorted, outPath)

}
