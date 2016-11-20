package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	svgHeaderTmpl = `<?xml version="1.0" standalone="no"?>
    <svg version="1.1"
    xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" xml:space="preserve"
    style="background-color: %s;" width="%fcm" height="%fcm">`
	svgTextTmpl = `<text x="%fcm" y="%fcm" style="font-family: %s; font-size: %fcm; fill: %s;"><![CDATA[%s]]></text>`
	svgFooter   = `</svg>`
)

func main() {
	var (
		srcPathFlag            = flag.String("src", "", "path to source code")
		outPathFlag            = flag.String("out", "", "path to image output [svg]")
		outWidthFlag           = flag.Float64("width", 10, "width [cm]")
		outHeightFlag          = flag.Float64("height", 12, "height [cm]")
		maskPathFlag           = flag.String("mask", "", "path to image mask [png]")
		maskDPIFlag            = flag.Int("maskdpi", 72, "DPI of image mask")
		maskScaleFlag          = flag.Float64("maskscale", 1.0, "scale of image mask [%]")
		maskAlphaThresholdFlag = flag.Float64("alphathreshold", 0.5, "alpha threshold for image mask [%]")
		fontNameFlag           = flag.String("fontname", "monospace", "font name")
		fontSizeFlag           = flag.Float64("fontsize", 0.1, "font size [cm]")
		fontSpacingFlag        = flag.Float64("fontspacing", 0.0, "font glyph spacing [%]")
		fgColorFlag            = flag.String("fgcolor", "#808080", "foreground/text color [rgba]")
		bgColorFlag            = flag.String("bgcolor", "#ffffff", "foreground/text color [rgba]")
		monochromeFlag         = flag.Bool("monochrome", false, "monochrome mode")
		debugFlag              = flag.Bool("debug", false, "debug mode")
	)
	flag.Parse()

	files, err := findFiles(*srcPathFlag, "*.go")
	if err != nil {
		log.Printf("Warn: problems searching files: %v", err)
	}
	text := extractText(files)

	var imgMask image.Image = image.Transparent
	if *maskPathFlag != "" {
		fMask, openErr := os.Open(*maskPathFlag)
		if openErr != nil {
			log.Fatalln("Error: opening mask:", openErr)
		}
		defer fMask.Close()

		imgMask, _, err = image.Decode(fMask)
		if err != nil {
			log.Fatalln("Error: decoding mask:", err)
		}
	}

	f := os.Stdout
	if *outPathFlag != "" {
		f, err = os.Create(*outPathFlag)
		if err != nil {
			log.Println("Error: opening out:", err)
		}
		defer f.Close()
	}

	var (
		fontName       = *fontNameFlag
		fontSize       = *fontSizeFlag
		out            = float64Rect(0, 0, *outWidthFlag, *outHeightFlag)
		glyphSize      = float64Pt(fontSize/1.5, fontSize)
		glyphSpacing   = glyphSize.Mul(*fontSpacingFlag)
		marginSize     = glyphSize.Mul(1.0)
		margin         = float64Rectangle{Min: marginSize, Max: out.Max.Sub(marginSize)}
		alphaThreshold = uint32(65535 * *maskAlphaThresholdFlag)
		maskSizeScaled = float64PtFromPixel(imgMask.Bounds().Max, *maskDPIFlag).Mul(*maskScaleFlag)
		maskScaled     = float64Rect(0, 0, maskSizeScaled.X, maskSizeScaled.Y)
		inmaskScaled   = maskScaled.Sub(maskScaled.Center()).Add(out.Center())
		glyphsPerLine  = int((out.Max.X - 2*marginSize.X) / (glyphSize.X + glyphSpacing.X))
	)

	s := fmt.Sprintf(svgHeaderTmpl, *bgColorFlag, *outWidthFlag, *outHeightFlag)
	f.WriteString(s)

	lines := splitSubN(text, glyphsPerLine)
	cursorOrigin := margin.Min
	cursorOrigin.Y += glyphSize.Y / 2
	for _, line := range lines {
		cursorOrigin.X = margin.Min.X
		if !cursorOrigin.In(out) {
			// No need to render invisible text
			break
		}

		for _, char := range line {
			charCol := *fgColorFlag
			// Calculate if and which mask color should be chosen
			if cursorOrigin.In(inmaskScaled) {
				var (
					charAbsCenter  = cursorOrigin.Add(float64Point{glyphSize.X / 2, -glyphSize.Y / 2})
					charRelCenter  = charAbsCenter.Sub(inmaskScaled.Min)
					imgMaskPt      = charRelCenter.Div(*maskScaleFlag)
					imgMaskPtPixel = imgMaskPt.ToPixel(*maskDPIFlag)
				)
				col := imgMask.At(imgMaskPtPixel.X, imgMaskPtPixel.Y)
				if _, _, _, a := col.RGBA(); a >= alphaThreshold {
					if *monochromeFlag {
						charCol = "#000000"
					} else {
						charCol = hexColor(col)
					}
				} else {
					if *debugFlag {
						charCol = "#ff0000"
					}
				}
			}

			s := fmt.Sprintf(svgTextTmpl, cursorOrigin.X, cursorOrigin.Y, fontName, fontSize, charCol, string(char))
			f.WriteString(s)

			cursorOrigin = cursorOrigin.Add(float64Point{glyphSize.X + glyphSpacing.X, 0})
		}
		cursorOrigin = cursorOrigin.Add(float64Point{0, glyphSize.Y + glyphSpacing.Y})
	}

	f.WriteString(svgFooter)
}

func findFiles(srcPath, srcPattern string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(srcPath, srcPattern))
	if err != nil {
		return []string{}, err
	}

	files := matches
	err = filepath.Walk(srcPath, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			matches, err := filepath.Glob(filepath.Join(path, srcPattern))
			if err != nil {
				return err
			}
			files = append(files, matches...)
		}
		return nil
	})
	return files, err
}

func extractText(files []string) string {
	var text bytes.Buffer
	for _, m := range files {
		f, osErr := os.Open(m)
		if osErr != nil {
			log.Printf("Warn: failed to read file: %s: %v", m, osErr)
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			if line := condense(scanner.Text()); line != "" {
				text.WriteString(line)
				text.WriteString(" ")
			}
		}
		f.Close()
		if err := scanner.Err(); err != nil {
			log.Printf("Warn: failed to scan file: %v", err)
		}

	}

	return text.String()
}

var r = regexp.MustCompile(`\s+`)

func condense(s string) string {
	return r.ReplaceAllString(strings.TrimSpace(s), " ")
}

func splitSubN(s string, n int) []string {
	sub := ""
	subs := []string{}

	runes := bytes.Runes([]byte(s))
	l := len(runes)
	for i, r := range runes {
		sub = sub + string(r)
		if (i+1)%n == 0 {
			subs = append(subs, sub)
			sub = ""
		} else if (i + 1) == l {
			subs = append(subs, sub)
		}
	}

	return subs
}

func hexColor(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r/257, g/257, b/257)
}

func float64Pt(x, y float64) float64Point {
	return float64Point{x, y}
}

func float64PtFromPixel(p image.Point, dpi int) float64Point {
	return float64Pt(pixelToCm(p.X, dpi), pixelToCm(p.Y, dpi))
}

type float64Point struct {
	X, Y float64
}

func (p float64Point) Add(q float64Point) float64Point {
	return float64Pt(p.X+q.X, p.Y+q.Y)
}

func (p float64Point) Sub(q float64Point) float64Point {
	return float64Pt(p.X-q.X, p.Y-q.Y)
}

func (p float64Point) Mul(k float64) float64Point {
	return float64Pt(p.X*k, p.Y*k)
}

func (p float64Point) Div(k float64) float64Point {
	return float64Pt(p.X/k, p.Y/k)
}

func (p float64Point) In(r float64Rectangle) bool {
	return p.Y >= r.Min.Y && p.Y <= r.Max.Y && p.X >= r.Min.X && p.X <= r.Max.X
}

func (p float64Point) ToPixel(dpi int) image.Point {
	return image.Pt(cmToPixel(p.X, dpi), cmToPixel(p.Y, dpi))
}

func float64Rect(x0, y0, x1, y1 float64) float64Rectangle {
	return float64Rectangle{float64Pt(x0, y0), float64Pt(x1, y1)}
}

type float64Rectangle struct {
	Min, Max float64Point
}

func (r float64Rectangle) Add(p float64Point) float64Rectangle {
	return float64Rect(r.Min.X+p.X, r.Min.Y+p.Y, r.Max.X+p.X, r.Max.Y+p.Y)
}

func (r float64Rectangle) Sub(p float64Point) float64Rectangle {
	return float64Rect(r.Min.X-p.X, r.Min.Y-p.Y, r.Max.X-p.X, r.Max.Y-p.Y)
}

func (r float64Rectangle) Mul(k float64) float64Rectangle {
	return float64Rect(r.Min.X*k, r.Min.Y*k, r.Max.X*k, r.Max.Y*k)
}

func (r float64Rectangle) Center() float64Point {
	return float64Pt((r.Max.X-r.Min.X)/2, (r.Max.Y-r.Min.Y)/2)
}

func (r float64Rectangle) Contains(p float64Point) bool {
	return p.Y >= r.Min.Y && p.Y <= r.Max.Y && p.X >= r.Min.X && p.X <= r.Max.X
}

func pixelToCm(pixel int, dpi int) float64 {
	const cmToDPI = 0.393701
	return float64(pixel) / (cmToDPI * float64(dpi))
}

func cmToPixel(cm float64, dpi int) int {
	const cmToDPI = 0.393701
	return int(float64(cm) * (cmToDPI * float64(dpi)))
}
