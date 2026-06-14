package fetch

import (
	"bytes"
	"context"
	"fmt"
	"image/color"
	"image/png"
	"path/filepath"
	"time"

	"dbot/pkg/htmlp"
	"dbot/pkg/render"

	"github.com/bwmarrin/discordgo"
	"github.com/fr-str/log"
)

var layout = render.Layout{
	CanvasWidth: 600,
	FontSize:    18,
	LineHeight:  30,
	PaddingX:    15,
	PaddingY:    10,
	BgColor:     color.Black,
}

func jbzd(ctx context.Context, url string) ([]*discordgo.File, error) {
	log.Trace("jbzd: extracting")

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	body, err := get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("jbzd: failed to get html body: %w", err)
	}
	defer body.Close()

	doc, err := htmlp.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("jbzd: failed to parse html body: %w", err)
	}

	article := doc.Find(htmlp.ByTag("article"))
	if article == nil {
		return nil, fmt.Errorf("jbzd: article not found")
	}

	elements := article.
		Find(htmlp.And(htmlp.ByTag("div"), htmlp.ByAttr("class", "article-elements"))).
		FindAll(htmlp.ByTag("div"))

	if len(elements) > 0 {
		return jbzdElements(ctx, elements)
	}

	single := article.Find(htmlp.And(htmlp.ByTag("div"), htmlp.ByAttrValues("class", "article-image")))
	if single != nil {
		return jbzdSingle(ctx, single)
	}
	return nil, fmt.Errorf("jbzd: no article elements found")
}

func jbzdSingle(ctx context.Context, single *htmlp.Node) ([]*discordgo.File, error) {
	// It's a single element article
	video := single.Find(htmlp.ByTag("videoplyr")).GetAttr("video_url")
	if video != "" {
		file, err := downloadToRam(ctx, video)
		if err != nil {
			return nil, fmt.Errorf("jbzd: download video: %w", err)
		}
		return []*discordgo.File{file}, nil
	}

	// Image
	src := single.Find(htmlp.ByTag("img")).GetAttr("src")
	img, err := get(ctx, src)
	if err != nil {
		return nil, fmt.Errorf("jbzd: fetch image: %w", err)
	}

	return []*discordgo.File{{Name: "dupa." + filepath.Ext(src), Reader: readerToRam(img)}}, nil
}

func jbzdElements(ctx context.Context, elements []*htmlp.Node) ([]*discordgo.File, error) {
	r, err := render.New(layout)
	if err != nil {
		return nil, fmt.Errorf("jbzd: create renderer: %w", err)
	}

	height := 0
	queue := []render.Item{}
	files := []*discordgo.File{}

	flush := func() error {
		if len(queue) == 0 {
			return nil
		}
		var err error
		files, err = appendImage(files, r, queue)
		if err != nil {
			return err
		}
		height = 0
		queue = queue[:0]
		return nil
	}

	for _, e := range elements {
		video := e.Find(htmlp.ByTag("videoplyr")).GetAttr("video_url")
		desc := e.Find(htmlp.ByAttrValues("class", "article-description"))

		switch {
		case video != "":
			if err := flush(); err != nil {
				return nil, fmt.Errorf("jbzd: flushing: %w", err)
			}

			file, err := downloadToRam(ctx, video)
			if err != nil {
				return nil, fmt.Errorf("jbzd: download video: %w", err)
			}
			files = append(files, file)

		case desc != nil:
			var txtHeight int
			queue, txtHeight = appendDescription(queue, desc, r)
			height += txtHeight

			if height > 1280 {
				if err := flush(); err != nil {
					return nil, fmt.Errorf("jbzd: flushing: %w", err)
				}
			}

		default:
			src := e.Find(htmlp.ByTag("img")).GetAttr("src")
			img, err := getImage(ctx, src)
			if err != nil {
				return nil, fmt.Errorf("jbzd: get image: %w", err)
			}

			item, err := r.Image(img)
			if err != nil {
				return nil, fmt.Errorf("jbzd: scale image: %w", err)
			}
			queue = append(queue, item)
			height += item.Height

			if height > 1280 {
				if err := flush(); err != nil {
					return nil, fmt.Errorf("jbzd: flushing: %w", err)
				}
			}
		}
	}

	err = flush()
	if err != nil {
		return nil, fmt.Errorf("jbzd: flushing: %w", err)
	}
	return files, nil
}

func appendImage(files []*discordgo.File, r *render.Renderer, q []render.Item) ([]*discordgo.File, error) {
	if len(q) == 0 {
		return files, nil
	}

	canvas, err := r.Render(q)
	if err != nil {
		return files, err
	}

	buf := &bytes.Buffer{}
	err = png.Encode(buf, canvas)
	if err != nil {
		return files, err
	}
	return append(files, &discordgo.File{Name: "dupa.png", Reader: buf}), nil
}

func appendDescription(queue []render.Item, desc *htmlp.Node, r *render.Renderer) ([]render.Item, int) {
	var height int

	spans := desc.FindAll(htmlp.ByTag("span"))
	if len(spans) == 0 {
		txt := r.Text(desc.Text(), color.White)
		return append(queue, txt), txt.Height
	}

	for _, span := range spans {
		col, _ := render.ParseCSSColor(span.GetAttr("style"))
		txt := r.Text(span.Text(), col)

		queue = append(queue, txt)
		height += txt.Height
	}

	return queue, height
}
