package main

import (
	"encoding/gob"
	"fmt"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path"
	"path/filepath"
	"pattern-evaluator/pkg/bucket"
	"pattern-evaluator/pkg/evaluate"
	"pattern-evaluator/pkg/triplebarrier"
	"sort"
	"strings"
)

const (
	fontSize         = 34
	cellTargetWidth  = 180
	cellTargetHeight = 92
	borderWidth      = 2
)

var cellBackground = color.RGBA{R: 230, G: 230, B: 230, A: 255}

func interpolate(value1, value2 uint8, factor float64) uint8 {
	if factor <= 0 {
		return value1
	}
	if factor >= 1 {
		return value2
	}

	result := float64(value1)*(1-factor) + float64(value2)*factor
	return uint8(math.Round(result))
}

func interpolateColor(value float64) color.RGBA {
	if value < 0.5 {
		normalized := ((0.5 - value) / 0.5) * 5
		return color.RGBA{
			R: interpolate(255, 255, normalized),
			G: interpolate(255, 63, normalized),
			B: interpolate(255, 52, normalized),
			A: 255,
		}
	} else {
		normalized := ((value - 0.5) / 0.5) * 5
		return color.RGBA{
			R: interpolate(255, 11, normalized),
			G: interpolate(255, 232, normalized),
			B: interpolate(255, 129, normalized),
			A: 255,
		}
	}
}

func decodeTable(filePath string) *evaluate.MetricsTable {
	f, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var output evaluate.MetricsTable
	gob.Register(triplebarrier.BarrierMetrics{})
	gob.Register(bucket.BucketMetrics{})
	err = gob.NewDecoder(f).Decode(&output)
	if err != nil {
		fmt.Println(filePath)
		panic(err)
	}

	return &output
}

func makeHeatmap(table *evaluate.MetricsTable, outPath string, key string) {

	matrix := table.Values

	// Determine matrix dimensions
	rows := len(matrix) + 1
	cols := len(matrix[0]) + 1

	// Create an empty image
	titleOffset := 100
	imageWidth := cellTargetWidth*cols + borderWidth*(cols+1)
	imageHeight := cellTargetHeight*rows + borderWidth*(rows+1) + titleOffset
	img := image.NewRGBA(image.Rect(0, 0, imageWidth, imageHeight))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	// Calculate cell width and height
	cellWidth := float64(cellTargetWidth)
	cellHeight := float64(cellTargetHeight)

	// Create a freetype context for drawing text
	fnt := loadFont()
	ctx := freetype.NewContext()
	ctx.SetDst(img)
	ctx.SetClip(img.Bounds())
	ctx.SetSrc(image.Black)
	ctx.SetFont(fnt)

	ctx.SetFontSize(fontSize)

	fileName := strings.TrimSuffix(path.Base(outPath), path.Ext(outPath))
	fileName = strings.ReplaceAll(fileName, "_", " ")
	drawText(ctx, fileName, borderWidth+int(cellWidth+float64(borderWidth)), int(cellHeight/2)+fontSize/2)

	ctx.SetFontSize(fontSize / 8 * 7)

	for y := 1; y < rows; y++ {
		cellY := titleOffset + borderWidth + int(float64(y)*cellHeight+float64(y*borderWidth))
		draw.Draw(img, image.Rect(borderWidth, cellY, borderWidth+int(cellWidth), cellY+int(cellHeight)), &image.Uniform{C: cellBackground}, image.Point{}, draw.Src)
		textX := fontSize / 2
		textY := cellY + int(cellHeight/2) + fontSize/2
		drawText(ctx, table.Rows[y-1], textX, textY)
	}

	for x := 1; x < cols; x++ {
		cellX := borderWidth + int(float64(x)*cellWidth+float64(x*borderWidth))
		draw.Draw(img, image.Rect(cellX, titleOffset+borderWidth, cellX+int(cellWidth), titleOffset+borderWidth+int(cellHeight)), &image.Uniform{C: cellBackground}, image.Point{}, draw.Src)
		textX := cellX + fontSize/2
		textY := titleOffset + int(cellHeight/2) + fontSize/2
		drawText(ctx, table.Columns[x-1], textX, textY)
	}

	maxSize := 0
	totalEntries := make([]int, 0)
	if key == "size" {
		for y := 1; y < rows; y++ {
			for x := 1; x < cols; x++ {
				val := matrix[y-1][x-1]
				s := 0
				if val != nil {
					s = val.Size()
				}
				if s > maxSize {
					maxSize = s
				}
				totalEntries = append(totalEntries, s)
			}
		}
		sort.Ints(totalEntries)
	}

	// Draw heatmap
	for y := 1; y < rows; y++ {
		for x := 1; x < cols; x++ {

			// Calculate cell position
			cellX := borderWidth + int(float64(x)*cellWidth+float64(x*borderWidth))
			cellY := titleOffset + borderWidth + int(float64(y)*cellHeight+float64(y*borderWidth))

			val := matrix[y-1][x-1]
			if val == nil {
				continue
			}

			// Set the cell color in the image
			cellColor := interpolateColor(val.Value())
			if key == "size" {
				// green: rgb(11, 232, 129)
				// red: rgb(255, 63, 52)
				ref := totalEntries[len(totalEntries)/2]
				rel := math.Min(float64(val.Size())/float64(ref), 1.0)
				cellColor = color.RGBA{
					R: interpolate(255, 11, rel),
					G: interpolate(63, 232, rel),
					B: interpolate(52, 129, rel),
					A: 255,
				}
			}
			draw.Draw(img, image.Rect(cellX, cellY, cellX+int(cellWidth), cellY+int(cellHeight)), &image.Uniform{C: cellColor}, image.Point{}, draw.Src)

			// Draw the value text within the cell
			ctx.SetFontSize(fontSize)
			textX := cellX + fontSize/2
			textY := cellY + int(cellHeight/2) + fontSize/2

			if key != "size" {
				drawText(ctx, fmt.Sprintf("%.2f", val.Emit(key)), textX, textY)
			} else {
				drawText(ctx, fmt.Sprintf("%d", val.Size()), textX, textY)
			}
		}
	}

	// Save the image as a PNG file
	file, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	png.Encode(file, img)
}

func drawText(ctx *freetype.Context, text string, x, y int) {
	pt := freetype.Pt(x, y)
	_, err := ctx.DrawString(text, pt)
	if err != nil {
		panic(err)
	}
}

func loadFont() *truetype.Font {

	fontPath := "./assets/fonts/Helvetica.ttf"

	fontData, err := os.ReadFile(fontPath)
	if err != nil {
		panic(err)
	}

	ttf, err := truetype.Parse(fontData)
	if err != nil {
		panic(err)
	}

	return ttf
}

func main() {

	_ = os.MkdirAll("./output/png/balanced", 0755)
	_ = os.MkdirAll("./output/png/worst", 0755)
	_ = os.MkdirAll("./output/png/size", 0755)
	_ = os.MkdirAll("./output/png/diff", 0755)
	_ = os.MkdirAll("./output/csv/balanced", 0755)
	_ = os.MkdirAll("./output/csv/worst", 0755)
	_ = os.MkdirAll("./output/csv/size", 0755)
	_ = os.MkdirAll("./output/csv/wins", 0755)

	tableDir := "./output/tables"

	err := filepath.Walk(tableDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(info.Name(), ".gob") {
			return nil
		}
		if !info.IsDir() {

			table := decodeTable(p)
			fileName := strings.TrimSuffix(path.Base(p), path.Ext(p))
			fmt.Println(fileName)

			xs := strings.Split(fileName, "_")
			fileRandom := strings.Join(xs[:len(xs)-1], "_") + "_random" + ".gob"
			dirName := filepath.Dir(p)
			randPath := filepath.Join(dirName, fileRandom)

			tableRandom := decodeTable(randPath)

			diffTable := evaluate.DiffMetricsTables(table, tableRandom)

			outPath := filepath.Join(".", "output", "png", "diff", fileName+".png")
			makeHeatmap(diffTable, outPath, "balanced")

			outPath = filepath.Join(".", "output", "png", "balanced", fileName+".png")
			makeHeatmap(table, outPath, "balanced")

			outPath = filepath.Join(".", "output", "png", "worst", fileName+".png")
			makeHeatmap(table, outPath, "worst")

			outPath = filepath.Join(".", "output", "png", "size", fileName+".png")
			makeHeatmap(table, outPath, "size")

			outPath = filepath.Join(".", "output", "csv", "balanced", fileName+".csv")
			evaluate.DumpMetrics(table.Values, "balanced", outPath, table.Rows, table.Columns)

			outPath = filepath.Join(".", "output", "csv", "worst", fileName+".csv")
			evaluate.DumpMetrics(table.Values, "worst", outPath, table.Rows, table.Columns)

			outPath = filepath.Join(".", "output", "csv", "size", fileName+".csv")
			evaluate.DumpMetrics(table.Values, "size", outPath, table.Rows, table.Columns)

			outPath = filepath.Join(".", "output", "csv", "wins", fileName+".csv")
			evaluate.DumpMetrics(table.Values, "wins", outPath, table.Rows, table.Columns)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
