package ascii

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/kovetskiy/mark/v16/attachment"
)

const (
	baseFontSize = 14.0
	padding      = 10
)

// ProcessASCII renders ASCII art text as a PNG image and returns it as an
// attachment. The scale parameter controls font size (14pt * scale).
func ProcessASCII(title string, art []byte, scale float64) (attachment.Attachment, error) {
	text := strings.TrimRight(string(art), "\n")
	lines := strings.Split(text, "\n")

	// Find the widest line by rune count.
	maxWidth := 0
	for _, line := range lines {
		if w := utf8.RuneCountInString(line); w > maxWidth {
			maxWidth = w
		}
	}

	// Parse the embedded Go Mono font.
	face, err := loadFace(scale)
	if err != nil {
		return attachment.Attachment{}, err
	}

	// Measure cell dimensions from the font metrics.
	metrics := face.Metrics()
	cellHeight := metrics.Ascent + metrics.Descent
	lineHeight := metrics.Height // includes leading
	cellWidth := font.MeasureString(face, "M")

	// Compute image dimensions.
	imgWidth := cellWidth.Ceil()*maxWidth + 2*padding
	imgHeight := lineHeight.Ceil()*len(lines) + 2*padding

	// Ensure minimum dimensions.
	if imgWidth < 1 {
		imgWidth = 1
	}
	if imgHeight < 1 {
		imgHeight = 1
	}

	// Create image with white background.
	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	// Draw each line of text.
	drawer := &font.Drawer{
		Dst:  img,
		Src:  &image.Uniform{color.Black},
		Face: face,
	}

	for i, line := range lines {
		// Position: padding offset + baseline (ascent from top of cell).
		x := fixed.I(padding)
		y := fixed.I(padding) + lineHeight*fixed.Int26_6(i) + cellHeight - metrics.Descent
		drawer.Dot = fixed.Point26_6{X: x, Y: y}
		drawer.DrawString(line)
	}

	// Encode PNG.
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return attachment.Attachment{}, err
	}

	// Compute checksum from source + scale (matches mermaid/d2 pattern).
	scaleBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(scaleBytes, math.Float64bits(scale))
	checksumInput := append(art, scaleBytes...)

	checksum, err := attachment.GetChecksum(bytes.NewReader(checksumInput))
	if err != nil {
		return attachment.Attachment{}, err
	}

	if title == "" {
		title = checksum
	}

	return attachment.Attachment{
		ID:        "",
		Name:      title,
		Filename:  title + ".png",
		FileBytes: buf.Bytes(),
		Checksum:  checksum,
		Replace:   title,
		Width:     strconv.Itoa(imgWidth),
		Height:    strconv.Itoa(imgHeight),
	}, nil
}

func loadFace(scale float64) (font.Face, error) {
	f, err := opentype.Parse(gomono.TTF)
	if err != nil {
		return nil, err
	}

	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    baseFontSize * scale,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}

	return face, nil
}
