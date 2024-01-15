package masks

import (

    f "github.com/faceplate-kleo/pixelsorter/lib/flags"
    psmath "github.com/faceplate-kleo/pixelsorter/lib/math"

    "image"
    "image/color"
    "log"
    "os"
)

func CreateContrastMask(imData image.Image, threshold uint8, flags f.Flags) *image.NRGBA {
	bounds := imData.Bounds()
	mask := image.NewNRGBA(bounds)

	max_x := bounds.Dx()
	max_y := bounds.Dy()

	for i := 0; i < max_x; i++ {
		for j := 0; j < max_y; j++ {
			currentColor := imData.At(i, j)
			//comparator := calculate_luminance(currentColor)
			comp_u32, _, _, _ := currentColor.RGBA()
			comparator := uint8(comp_u32)
			outColor := color.White
			if flags.INVERT {
				outColor = color.Black
			}
			if comparator < threshold {
				if flags.INVERT {
					outColor = color.Black
				} else {
					outColor = color.White
				}
			}

			if flags.MASK_DEBUG {
				outColor = color.White
			}

			mask.Set(i, j, outColor)
		}
	}

	return mask
}

func ReadContrastMask(maskInPath string, bounds image.Rectangle) *image.NRGBA {
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

	final_x := psmath.IntMin(bound_x, mask_x)
	final_y := psmath.IntMin(bound_y, mask_y)

	//copy to output buffer

	for i := 0; i < final_x; i++ {
		for j := 0; j < final_y; j++ {
			output.Set(i, j, maskData.At(i, j))
		}
	}

	return output
}

func ColorIsWhite(toComp color.Color) bool {
	cr, cg, cb, ca := toComp.RGBA()
	return (cr + cg + cb + ca) == 65535*4
}

func ColorIsInit(toComp color.Color) bool {
	cr, cg, cb, ca := toComp.RGBA()
	return (cr + cg + cb + ca) == 0
}

func GetMaskSpan(mask *image.NRGBA, start_x, start_y, max_x, max_y int) (int, int) {
	if !ColorIsWhite(mask.At(start_x, start_y)) {
		return start_x, start_y
	}

	relevant_bound := max_x
	relevant_constant := start_y

	for i := start_x; i < relevant_bound; i++ {
		if !ColorIsWhite(mask.At(i, relevant_constant)) {
			return i, start_y
		}
	}

	return relevant_bound, start_y
}
