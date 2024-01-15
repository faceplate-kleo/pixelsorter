package nrgbautil

import (
    "image"
    "image/draw"
    "image/png"
    "fmt"
    "log"
    "os"

    f "github.com/faceplate-kleo/pixelsorter/lib/flags"
    psmath "github.com/faceplate-kleo/pixelsorter/lib/math"
)

func LoadImage(path string) *image.NRGBA {
    
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

func WriteFile (imData *image.NRGBA, path string) {
    out, err := os.Create(path)
    if err != nil {
        log.Fatal(err)
    }
    png.Encode(out, imData.SubImage(imData.Rect))
    out.Close()
}

func DataToNrgba(imData image.Image, flags f.Flags) *image.NRGBA{
    out := image.NewNRGBA(imData.Bounds())
    max_x := imData.Bounds().Dx()
    max_y := imData.Bounds().Dy()

    for i := 0; i < max_x; i++ {
        for j := 0; j < max_y; j++ {
            if flags.SOURCE_DEBUG {
                out.Set(i, j, psmath.GetRandomColor())
            } else {
                out.Set(i, j, imData.At(i, j)) 
            }
        }
    }

    return out
}

func ReverseRowsNrgba(imData *image.NRGBA) {
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
func ReverseColsNrgba(imData *image.NRGBA) {
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

func RotateNrgba(imData *image.NRGBA, rotations int) *image.NRGBA {
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
        ReverseRowsNrgba(output)
        ReverseColsNrgba(output)
        return output
    }

    if rotations == 3 {
        ReverseRowsNrgba(imData)
    }

    if rotations % 2 != 0 {
        for i := 0; i < desired_y; i++ {
            for j := 0; j < desired_x; j++ {
                output.Set(j, i, imData.At(i,j))
            }
        }
    }

    if rotations == 1 {
        ReverseRowsNrgba(output)
    }


    if rotations == 3 {
        ReverseRowsNrgba(imData)
    }

    return output 
}

func FlipNrgba(imData *image.NRGBA, horizontal bool) *image.NRGBA {
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
        ReverseRowsNrgba(output)
    }

    return output 
}

