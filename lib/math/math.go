package math

import (
    f "github.com/faceplate-kleo/pixelsorter/lib/flags"
   "image/color"
   "math/rand"
)

func MeanCompare(colorA, colorB color.Color, flags f.Flags) bool {
    
    ar, ag, ab, _ := colorA.RGBA()
    br, bg, bb, _ := colorB.RGBA()
    mean_a := (ar + ag + ab) / 3 
    mean_b := (br + bg + bb) / 3 
    if flags.DESCEND {
        return mean_a > mean_b
    } else {
        return mean_a < mean_b
    }
}

func RedCompare(colorA, colorB color.Color, flags f.Flags) bool {
    ar, _, _, _ := colorA.RGBA()
    br, _, _, _ := colorB.RGBA()
    if flags.DESCEND{
        return ar > br
    } else {
        return ar < br
    }
}

func Merge(a []color.Color, iLeft, iRight, iEnd int, b []color.Color, flags f.Flags) []color.Color {
    i := iLeft 
    j := iRight 

    for k := iLeft; k < iEnd; k++ {
        comp := false 
        if i < len(a) && j < len(a) {
            if flags.MEAN_COMPARE {
                comp = MeanCompare(a[i], a[j], flags)
            }
            if flags.GRAY_RED_COMPARE {
               comp = RedCompare(a[i], a[j], flags) 
            }
        }

        if i < iRight && (j >= iEnd || comp) {
            b[k] = a[i]
            i++
        } else {
            if flags.CRUSH {
                b[i] = a[j]
            } else {
                b[k]= a[j]
            }
            j++
        }
    }
    return b

}

func IntMin(a, b int) int {
    if a < b {
        return a 
    } else {
        return b
    }
}
func IntMax(a, b int) int {
    if a > b {
        return a 
    } else {
        return b
    }
}

func SpanMergesort(span []color.Color, flags f.Flags) []color.Color {
    n := len(span)
    workArr := make([]color.Color, n)

    copy(workArr, span)

    for width := 1; width < n; width = 2*width {
        for i := 0; i < n; i = i + 2*width {
            workArr = Merge(
                        span, 
                        i, 
                        IntMin(i+width, n), 
                        IntMin(i+2*width, n), 
                        workArr, 
                        flags)
        }
        copy(span, workArr)
    }

    return span

}

func GetRandomColor() color.RGBA {
    ra := uint8(rand.Intn(255))
    rg := uint8(rand.Intn(255))
    rb := uint8(rand.Intn(255))

    return color.RGBA{ra, rg, rb, 255}
    
}

func SampleSignal(sourceIndex, domain, signalSize int, signal []int) float64 {
    ratio := float64(signalSize) / float64(domain) 
    //ratio := float64(domain) / float64(signalSize)
    adjusted_index := int(float64(sourceIndex) * ratio)

    adjusted_index = IntMax(adjusted_index, 0)
    adjusted_index = IntMin(adjusted_index, signalSize-1)

    return float64(signal[adjusted_index])
}

func CalculateLuminance(rgbcolor color.Color) uint8 {
    var int_r, int_g, int_b, _ = rgbcolor.RGBA()
    r := float32(int_r)
    g := float32(int_g)
    b := float32(int_b)
    return uint8((0.2126 * r) + (0.7152 * g) + (0.0722 * b))
}
