package render

import (
	_ "embed"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

//go:embed OpenSans-Regular.ttf
var fontBytes []byte

type ItemType uint8

const (
	TypeInvalid ItemType = iota
	TypeText
	TypeImage
)

type Item struct {
	Type   ItemType
	Img    image.Image
	Height int
	Lines  []string
	Color  color.Color
}

type Layout struct {
	CanvasWidth int
	FontSize    float64
	LineHeight  int
	PaddingX    int
	PaddingY    int
	BgColor     color.Color
}

type Renderer struct {
	layout Layout
	font   *font.Drawer
}

func New(layout Layout) (*Renderer, error) {
	f, err := opentype.Parse(fontBytes)
	if err != nil {
		return nil, fmt.Errorf("image: parse font: %w", err)
	}

	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    layout.FontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("image: create font face: %w", err)
	}

	return &Renderer{
		layout: layout,
		font:   &font.Drawer{Face: face},
	}, nil
}

func (r *Renderer) Render(queue []Item) (*image.RGBA, error) {
	contentHeight := 0
	for _, item := range queue {
		contentHeight += item.Height
	}

	if contentHeight == 0 {
		return nil, errors.New("image.Render: height == 0")
	}

	totalCanvasHeight := contentHeight + (r.layout.PaddingY * 2)

	canvas := image.NewRGBA(image.Rect(0, 0, r.layout.CanvasWidth, totalCanvasHeight))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(r.layout.BgColor), image.Point{}, draw.Src)
	r.font.Dst = canvas

	y := r.layout.PaddingY

	for _, item := range queue {
		switch item.Type {
		case TypeText:
			if item.Color != nil {
				r.font.Src = image.NewUniform(item.Color)
			} else {
				// Fallback to inverse background color
				red, green, blue, _ := r.layout.BgColor.RGBA()
				inverseColor := color.RGBA64{R: ^uint16(red), G: ^uint16(green), B: ^uint16(blue), A: 0xffff}
				r.font.Src = image.NewUniform(inverseColor)
			}

			yOff := y + int(r.layout.FontSize)
			for _, line := range item.Lines {
				r.font.Dot = fixed.Point26_6{
					X: fixed.I(r.layout.PaddingX),
					Y: fixed.I(yOff),
				}
				r.font.DrawString(line)
				yOff += r.layout.LineHeight
			}

		case TypeImage:
			targetBounds := image.Rect(0, y, r.layout.CanvasWidth, y+item.Height)
			draw.Draw(canvas, targetBounds, item.Img, image.Point{}, draw.Over)

		default:
			return nil, errors.New("image.Render: invalid type")
		}
		y += item.Height
	}

	return canvas, nil
}

// Text calculates line wrapping lengths, height, and sets the item text color.
func (r *Renderer) Text(text string, textColor color.Color) Item {
	maxWidth := r.layout.CanvasWidth - (r.layout.PaddingX * 2)

	var lines []string
	var current string
	for word := range strings.FieldsSeq(text) {
		line := word
		if current != "" {
			line = current + " " + word
		}

		if r.font.MeasureString(line).Round() > maxWidth {
			if current != "" {
				lines = append(lines, current)
			}
			current = word
		} else {
			current = line
		}
	}
	if current != "" {
		lines = append(lines, current)
	}

	totalHeight := (len(lines) * r.layout.LineHeight)
	item := Item{
		Type:   TypeText,
		Lines:  lines,
		Height: totalHeight,
		Color:  textColor,
	}
	return item
}

// Image downloads and scales remote assets to fit the layout width parameters.
func (r *Renderer) Image(img image.Image) (Item, error) {
	bounds := img.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()

	if srcW == r.layout.CanvasWidth {
		return Item{Type: TypeImage, Img: img, Height: srcH}, nil
	}

	scaledHeight := int(float64(srcH) / float64(srcW) * float64(r.layout.CanvasWidth))
	finalImg := image.NewRGBA(image.Rect(0, 0, r.layout.CanvasWidth, scaledHeight))

	xdraw.BiLinear.Scale(finalImg, finalImg.Bounds(), img, bounds, draw.Over, nil)

	item := Item{
		Type:   TypeImage,
		Img:    finalImg,
		Height: scaledHeight,
	}
	return item, nil
}
