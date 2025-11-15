package match

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"sort"
	"xhgm_price_tool/good/templates"
)

// Config configuration struct
type Config struct {
	IgnoreTransparent bool    // Whether to ignore transparent pixels
	Threshold         float64 // Detection threshold
	StepSize          int     // Search step size
}

// TemplateStats template statistics
type TemplateStats struct {
	mean        float64
	stdDev      float64
	validPixels int
	pixelValues []float64
	positions   []struct{ x, y int }
}

var defaulThreshold = 0.9

var thresholdMap = map[int]float64{
	1: 0.92,
	0: 0.94,
	2: 0.89,
	4: 0.89,
	5: 0.89,
	3: 0.89,
}

var subMap = map[int]int{
	3: 2,
	5: 2,
	0: 2,
	6: 2,
	7: 2,
	8: 3,
	9: 2,
}

func GetPrice(priceClip image.Image, verbose bool) (price int) {
	config := Config{
		IgnoreTransparent: true, // Ignore transparent pixels
		Threshold:         defaulThreshold,
		StepSize:          1, // Search step size
	}
	arr := [][3]int{} // x, number, score
	for i := range 10 {
		if thre, ok := thresholdMap[i]; ok {
			config.Threshold = thre
		} else {
			config.Threshold = defaulThreshold
		}
		if subMap[i] > 0 {
			results := []struct {
				x, y  int
				score float64
			}{}
			for sub := range subMap[i] {
				if verbose {
					fmt.Printf("static/templates/%d_%d.png\n", i, sub)
				}
				templateImg, err := templates.GetEmbedTemplate(fmt.Sprintf("%d_%d", i, sub)) // Assume template may have transparent channel
				if err != nil {
					panic(err)
				}
				// robotgo.Save(templateImg, fmt.Sprintf("%d_%d", i, sub)+".png")

				// Pre-calculate template statistics (considering pixels to ignore)
				templateStats := CalculateTemplateStats(templateImg, config)

				result, _ := FindBestMatch(priceClip, templateImg, templateStats, config)
				if verbose {
					fmt.Printf("result: %+v\n", result)
					// if len(result) > 0 {
					// 	robotgo.Save(outputImage, fmt.Sprintf("%d_tmp.png", i))
					// }
				}
				results = append(results, result...)
			}
			// Aggregate results, group items within 2-pixel error range
			tmp := aggResult(results)
			for _, item := range tmp {
				item[1] = i
				arr = append(arr, item)
			}
			if verbose {
				fmt.Printf("arr: %v\n", arr)
			}
		} else {
			if verbose {
				fmt.Printf("static/templates/%d.png\n", i)
			}
			templateImg, err := templates.GetEmbedTemplate(fmt.Sprint(i)) // Assume template may have transparent channel
			if err != nil {
				panic(err)
			}
			// Pre-calculate template statistics (considering pixels to ignore)
			templateStats := CalculateTemplateStats(templateImg, config)

			result, _ := FindBestMatch(priceClip, templateImg, templateStats, config)
			if verbose {
				fmt.Printf("result: %+v\n", result)
				// if len(result) > 0 {
				// 	robotgo.Save(outputImage, fmt.Sprintf("%d_tmp.png", i))
				// }
			}
			// Aggregate results, group items within 2-pixel error range
			tmp := aggResult(result)
			for _, item := range tmp {
				item[1] = i
				arr = append(arr, item)
			}
			if verbose {
				fmt.Printf("arr: %v\n", arr)
			}
		}
	}
	hits := map[int][3]int{}
	find := func(i int) bool {
		return hits[i][0] > 0
	}
	for _, item := range arr {
		if verbose {
			fmt.Printf("item: %v\n", item)
		}
		var existItem [3]int
		var index int = -1
		for offset := range 3 {
			if verbose {
				fmt.Printf("hits: %+v\n", hits)
				fmt.Printf("(item[0] + offset): %v\n", (item[0] + offset))
				fmt.Printf("(item[0] - offset): %v\n", (item[0] - offset))
			}
			if find(item[0] + offset) {
				existItem = hits[item[0]+offset]
				index = item[0] + offset
				break
			} else if find(item[1] - offset) {
				existItem = hits[item[0]-offset]
				index = item[0] - offset
				break
			}
		}
		if index > 0 && item[2] > existItem[2] {
			hits[index] = item
		} else if index == -1 {
			hits[item[0]] = item
		}
	}
	if verbose {
		fmt.Printf("hits: %+v\n", hits)
	}
	arr = [][3]int{}
	for _, item := range hits {
		arr = append(arr, item)
	}
	sort.Slice(arr, func(i, j int) bool {
		return arr[i][0] < arr[j][0]
	})
	for _, tmp := range arr {
		price = price*10 + tmp[1]
	}
	return price
}

func aggResult(result []struct {
	x, y  int
	score float64
}) (arr [][3]int) {
	agg := map[int]struct {
		x, y  int
		score float64
	}{}
	offsetLimit := 3
	for _, row := range result {
		existIndex := -1
		for xOffset := range offsetLimit {
			if _, ok := agg[row.x+xOffset]; ok {
				existIndex = row.x + xOffset
				break
			}
			if _, ok := agg[row.x-xOffset]; ok {
				existIndex = row.x - xOffset
				break
			}
		}
		if existIndex > 0 && row.score > agg[existIndex].score {
			agg[existIndex] = row
		} else if existIndex == -1 {
			agg[row.x] = row
		}
	}
	for key, item := range agg {
		arr = append(arr, [3]int{key, -1, int(item.score * 10000)})
	}
	return arr
}

// CalculateTemplateStats calculates template statistics, skipping specified pixels
func CalculateTemplateStats(template image.Image, config Config) TemplateStats {
	width, height := template.Bounds().Dx(), template.Bounds().Dy()

	var validPixels []float64
	var validPositions []struct{ x, y int }
	var sum float64

	// First pass: collect valid pixels
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if shouldSkipPixel(template.At(x, y), config) {
				continue
			}

			gray := getGrayValue(template.At(x, y))
			validPixels = append(validPixels, gray)
			validPositions = append(validPositions, struct{ x, y int }{x, y})
			sum += gray
		}
	}

	validCount := len(validPixels)
	if validCount == 0 {
		return TemplateStats{}
	}

	mean := sum / float64(validCount)

	// Calculate standard deviation
	var sumSq float64
	for _, pixel := range validPixels {
		diff := pixel - mean
		sumSq += diff * diff
	}
	stdDev := math.Sqrt(sumSq / float64(validCount))

	return TemplateStats{
		mean:        mean,
		stdDev:      stdDev,
		validPixels: validCount,
		pixelValues: validPixels,
		positions:   validPositions,
	}
}

// shouldSkipPixel determines whether this pixel should be skipped
func shouldSkipPixel(c color.Color, config Config) bool {
	// Check transparent pixels
	if config.IgnoreTransparent {
		_, _, _, a := c.RGBA()
		if a < 0xFFFF { // Not fully opaque
			return true
		}
	}

	// Check specific color
	// if config.IgnoreColor != nil {
	// 	r1, g1, b1, a1 := c.RGBA()
	// 	r2, g2, b2, a2 := config.IgnoreColor.RGBA()
	// 	if r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2 {
	// 		return true
	// 	}
	// }

	return false
}

// FindBestMatch finds the best match location
func FindBestMatch(main, template image.Image, stats TemplateStats, config Config) (result []struct {
	x, y  int
	score float64
}, outputImage image.Image) {
	mW, mH := main.Bounds().Dx(), main.Bounds().Dy()
	// fmt.Println(mW, mH)
	tW, tH := template.Bounds().Dx(), template.Bounds().Dy()

	// Create a copy of the output image
	outputImage = image.NewRGBA(main.Bounds())
	draw.Draw(outputImage.(*image.RGBA), main.Bounds(), main, image.Point{}, draw.Src)

	// Define red border color
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}

	for y := 0; y <= mH-tH; y += config.StepSize {
		for x := 0; x <= mW-tW; x += config.StepSize {
			score := calculateNCC(main, x, y, stats)
			if score > config.Threshold {
				result = append(result, struct {
					x, y  int
					score float64
				}{x, y, score})

				// Draw red border at match location
				drawRedBorder(outputImage.(*image.RGBA), x, y, tW, tH, red)
			}
		}
	}

	return result, outputImage
}

// drawRedBorder draws a 1px red border at the specified location
func drawRedBorder(img *image.RGBA, x, y, width, height int, borderColor color.RGBA) {
	// Draw top border
	for i := x; i < x+width; i++ {
		if i >= 0 && i < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
			img.Set(i, y, borderColor)
		}
	}

	// Draw bottom border
	for i := x; i < x+width; i++ {
		if i >= 0 && i < img.Bounds().Dx() && y+height-1 >= 0 && y+height-1 < img.Bounds().Dy() {
			img.Set(i, y+height-1, borderColor)
		}
	}

	// Draw left border
	for j := y; j < y+height; j++ {
		if x >= 0 && x < img.Bounds().Dx() && j >= 0 && j < img.Bounds().Dy() {
			img.Set(x, j, borderColor)
		}
	}

	// Draw right border
	for j := y; j < y+height; j++ {
		if x+width-1 >= 0 && x+width-1 < img.Bounds().Dx() && j >= 0 && j < img.Bounds().Dy() {
			img.Set(x+width-1, j, borderColor)
		}
	}
}

// calculateNCC improved NCC calculation, using only valid pixels
func calculateNCC(main image.Image, startX, startY int, stats TemplateStats) float64 {
	if stats.validPixels == 0 {
		return 0.0
	}

	// Calculate mean of valid pixels in main image region
	regionMean := calculateRegionMean(main, startX, startY, stats.positions)
	// fmt.Printf("regionMean: %v\n", regionMean)

	// Calculate covariance and standard deviation of main image region
	var covSum, mainSumSq float64

	for i, pos := range stats.positions {
		templatePixel := stats.pixelValues[i]
		mainPixel := getGrayValue(main.At(startX+pos.x, startY+pos.y))

		// Covariance term
		cov := (templatePixel - stats.mean) * (mainPixel - regionMean)
		covSum += cov

		// Variance term of main image region
		diff := mainPixel - regionMean
		mainSumSq += diff * diff
	}

	// Calculate standard deviation of main image region
	regionStdDev := math.Sqrt(mainSumSq / float64(stats.validPixels))

	if stats.stdDev == 0 || regionStdDev == 0 {
		return 0.0
	}

	// Calculate NCC
	ncc := covSum / (float64(stats.validPixels) * stats.stdDev * regionStdDev)
	return ncc
}

// calculateRegionMean calculates the mean of valid pixels in a specific region of the main image
func calculateRegionMean(main image.Image, startX, startY int, positions []struct{ x, y int }) float64 {
	var sum float64
	for _, pos := range positions {
		sum += getGrayValue(main.At(startX+pos.x, startY+pos.y))
	}
	return sum / float64(len(positions))
}

// getGrayValue converts color to grayscale value
func getGrayValue(c color.Color) float64 {
	// Handle different types of color models
	switch col := c.(type) {
	case color.RGBA:
		return 0.299*float64(col.R) + 0.587*float64(col.G) + 0.114*float64(col.B)
	case color.RGBA64:
		return 0.299*float64(col.R>>8) + 0.587*float64(col.G>>8) + 0.114*float64(col.B>>8)
	case color.NRGBA:
		return 0.299*float64(col.R) + 0.587*float64(col.G) + 0.114*float64(col.B)
	case color.NRGBA64:
		return 0.299*float64(col.R>>8) + 0.587*float64(col.G>>8) + 0.114*float64(col.B>>8)
	case color.Gray:
		return float64(col.Y)
	case color.Gray16:
		return float64(col.Y >> 8)
	default:
		// Generic handling
		r, g, b, a := c.RGBA()
		if a == 0 {
			return 0 // Fully transparent
		}
		// Convert to 0-255 range and calculate grayscale
		return 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
	}
}

func loadImageViaBytes(b []byte) (img image.Image, err error) {
	reader := bytes.NewReader(b)
	img, err = png.Decode(reader)
	if err != nil {
		return nil, err
	}
	return img, nil
}

// func loadImage(path string) image.Image {
// 	file, err := os.Open(path)
// 	if err != nil {
// 		panic("Unable to open image: " + path)
// 	}
// 	defer file.Close()

// 	// Try to decode as JPEG or PNG
// 	img, err := jpeg.Decode(file)
// 	if err != nil {
// 		file.Seek(0, 0) // Reset file pointer
// 		img, err = png.Decode(file)
// 		if err != nil {
// 			panic("Unable to decode image: " + path)
// 		}
// 	}
// 	return ConvertDarkPixelsToBlack(img, 80)
// }

// ConvertDarkPixelsToBlack converts pixels darker than the specified threshold to black
func ConvertDarkPixelsToBlack(img image.Image, threshold uint8) image.Image {
	// Create a new image, maintaining the original image size and format
	bounds := img.Bounds()
	result := image.NewRGBA(bounds)

	// Iterate through each pixel
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Get original pixel color
			originalColor := img.At(x, y)
			r, g, b, a := originalColor.RGBA()

			// Convert 32-bit color value to 8-bit
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)
			a8 := uint8(a >> 8)

			// Check if pixel is darker than threshold
			if isDarkerThanThreshold(r8, g8, b8, threshold) {
				// Convert to black, maintaining original transparency
				result.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: a8})
			} else {
				// Keep original color
				result.Set(x, y, originalColor)
			}
		}
	}

	return result
}

// isDarkerThanThreshold checks if color is darker than threshold
// Uses RGB average to determine brightness
func isDarkerThanThreshold(r, g, b, threshold uint8) bool {
	// Calculate RGB average as brightness
	brightness := (uint32(r) + uint32(g) + uint32(b)) / 3
	return brightness < uint32(threshold)
}
