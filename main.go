package main

import (
	"bufio"
	"bytes"
	"flag"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/inconsolata"
	"golang.org/x/image/math/fixed"
)

const (
	defaultDPI = 72
	printDPI   = 600
)

func main() {
	srcPath := flag.String("src", "", "path to source code")
	maskPath := flag.String("mask", "", "path to image mask")
	outPath := flag.String("out", "", "path to image output")
	width := flag.Int("width", 50, "width (cm)")
	height := flag.Int("height", 70, "height (cm)")
	flag.Parse()

	_ = maskPath

	matches, err := filepath.Glob(filepath.Join(*srcPath, "*.go"))
	if err != nil {
		log.Fatalln("Error:", err)
	}

	var text string
	r := regexp.MustCompile(`\s+`)
	for _, m := range matches {
		f, osErr := os.Open(m)
		if osErr != nil {
			log.Printf("Warn: failed to read file: %s: %v", m, osErr)
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			// Condense lines
			line := scanner.Text()
			s := strings.TrimSpace(line)
			s = r.ReplaceAllString(s, " ")
			if s != "" {
				text += s + " "
			}
		}
		if scanErr := scanner.Err(); err != nil {
			log.Printf("Warn: failed to scan file: %v", scanErr)
		}
	}

	bgCol := color.RGBA{255, 255, 255, 255}
	fgCol := color.RGBA{192, 192, 192, 255}

	outW := cmToPixel(*width, printDPI)
	outH := cmToPixel(*height, printDPI)

	img := image.NewRGBA(image.Rect(0, 0, outW, outH))
	draw.Draw(img, img.Bounds(), image.NewUniform(bgCol), image.ZP, draw.Src)

	fMask, err := os.Open(*maskPath)
	if err != nil {
		log.Fatalln("Error:", err)
	}
	defer fMask.Close()

	imgMask, _, err := image.Decode(fMask)
	if err != nil {
		log.Fatalln("Error:", err)
	}

	bounds := imgMask.Bounds()
	maskW, maskH := bounds.Max.X, bounds.Max.Y
	_, _ = maskW, maskH

	// scaleW := float64(outW) / float64(maskW)
	// scaleH := float64(outH) / float64(maskH)
	// scale := scaleW
	// if scaleH < scaleW {
	// 	scale = scaleH
	// }

	boundLeft := (outW - maskW) / 2
	boundTop := (outH - maskH) / 2
	boundRight := (outW + maskW) / 2
	boundBottom := (outH + maskH) / 2

	charW := outW / 8
	lines := SplitSubN(text, charW-1) // allow some margin

	for y, line := range lines {
		for x, char := range line {
			point := fixed.Point26_6{
				X: fixed.Int26_6(6 + (x+1)*(64+446)), // 446 ~ (64*7) ?
				Y: fixed.Int26_6((y + 1) * 16 * 64),
			}

			charCol := image.NewUniform(fgCol)

			if point.X.Round() >= boundLeft && point.X.Round() <= boundRight &&
				point.Y.Round() >= boundTop && point.Y.Round() <= boundBottom {
				charCenterX := (point.X.Round() - boundLeft) + 8
				charCenterY := (point.Y.Round() - boundTop) + 16
				// Get color under mask
				c := imgMask.At(charCenterX, charCenterY)
				_, _, _, a := c.RGBA()
				if a == 65535 { // Don't render partial transparency
					charCol = image.NewUniform(c)
				}
			}

			d := &font.Drawer{
				Dst:  img,
				Src:  charCol,
				Face: inconsolata.Regular8x16, // basicfont.Face7x13,
				Dot:  point,
			}
			d.DrawString(string(char))
		}
	}

	f, err := os.Create(*outPath)
	if err != nil {
		log.Println("Error:", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		log.Println("Error:", err)
	}
}

func cmToPixel(cm, dpi int) int {
	// Go currently can only do 72 DPI, so use that as reference.
	dpiNormalizer := float64(dpi) / float64(defaultDPI)
	// Not sure where this comes from
	unknownNormalizer := 0.12
	return int(float64(cm) * 0.393701 * dpiNormalizer * float64(dpi) * unknownNormalizer)
}

func SplitSubN(s string, n int) []string {
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
