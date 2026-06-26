package renderer

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/kovetskiy/mark/v16/ascii"
	"github.com/kovetskiy/mark/v16/attachment"
	"github.com/kovetskiy/mark/v16/d2"
	"github.com/kovetskiy/mark/v16/mermaid"
	"github.com/kovetskiy/mark/v16/stdlib"
	"github.com/kovetskiy/mark/v16/types"
	"github.com/reconquest/pkg/log"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

type ConfluenceFencedCodeBlockRenderer struct {
	html.Config
	Stdlib      *stdlib.Lib
	MarkConfig  types.MarkConfig
	Attachments attachment.Attacher
}

var reBlockDetails = regexp.MustCompile(
	// (<Lang>|-) (collapse|<theme>|\d)* (title <title>)?

	`^(?:(\w*)|-)\s*\b(\S.*?\S?)??\s*(?:\btitle\s+(\S.*\S?))?$`,
)

// NewConfluenceRenderer creates a new instance of the ConfluenceRenderer
func NewConfluenceFencedCodeBlockRenderer(stdlib *stdlib.Lib, attachments attachment.Attacher, cfg types.MarkConfig, opts ...html.Option) renderer.NodeRenderer {
	return &ConfluenceFencedCodeBlockRenderer{
		Config:      html.NewConfig(),
		Stdlib:      stdlib,
		MarkConfig:  cfg,
		Attachments: attachments,
	}
}

// RegisterFuncs implements NodeRenderer.RegisterFuncs .
func (r *ConfluenceFencedCodeBlockRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
}

func ParseLanguage(lang string) string {
	// lang takes the following form: language? "collapse"? ("title"? <any string>*)?
	// let's split it by spaces
	paramlist := strings.Fields(lang)

	// get the word in question, aka the first one
	first := lang
	if len(paramlist) > 0 {
		first = paramlist[0]
	}

	if first == "collapse" || first == "title" {
		// collapsing or including a title without a language
		return ""
	}
	// the default case with language being the first one
	return first
}

func ParseTitle(lang string) string {
	index := strings.Index(lang, "title")
	if index >= 0 {
		// it's found, check if title is given and return it
		start := index + 6
		if len(lang) > start {
			return strings.TrimSpace(lang[start:])
		}
	}
	return ""
}

// parseKeyValueInt extracts a "key=N" integer value from an option string.
// Returns (value, true) if found and valid, (0, false) otherwise.
func parseKeyValueInt(option, key string) (int, bool) {
	prefix := key + "="
	if strings.HasPrefix(option, prefix) {
		if v, err := strconv.Atoi(option[len(prefix):]); err == nil && v > 0 {
			return v, true
		}
	}
	return 0, false
}

// parseKeyValueFloat extracts a "key=N" float value from an option string.
// Returns (value, true) if found and valid, (0, false) otherwise.
func parseKeyValueFloat(option, key string) (float64, bool) {
	prefix := key + "="
	if strings.HasPrefix(option, prefix) {
		if v, err := strconv.ParseFloat(option[len(prefix):], 64); err == nil && v > 0 {
			return v, true
		}
	}
	return 0, false
}

// renderFencedCodeBlock renders a FencedCodeBlock
func (r *ConfluenceFencedCodeBlockRenderer) renderFencedCodeBlock(writer util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	var info []byte
	nodeFencedCodeBlock := node.(*ast.FencedCodeBlock)
	if nodeFencedCodeBlock.Info != nil {
		segment := nodeFencedCodeBlock.Info.Segment
		info = segment.Value(source)
	}
	groups := reBlockDetails.FindStringSubmatch(string(info))
	linenumbers := false
	firstline := 0
	theme := ""
	collapse := false
	lang := ""
	var options []string
	title := ""
	widthOverride := 0
	scaleOverride := 0.0
	if len(groups) > 0 {
		lang, options, title = groups[1], strings.Fields(groups[2]), groups[3]
		for _, option := range options {
			if option == "collapse" {
				collapse = true
				continue
			}
			if option == "nocollapse" {
				collapse = false
				continue
			}
			if option == "linenumbers" {
				linenumbers = true
				continue
			}
			if v, ok := parseKeyValueInt(option, "width"); ok {
				widthOverride = v
				continue
			}
			if v, ok := parseKeyValueFloat(option, "scale"); ok {
				scaleOverride = v
				continue
			}

			var i int
			if _, err := fmt.Sscanf(option, "%d", &i); err == nil {
				linenumbers = i > 0
				firstline = i
				continue
			}
			theme = option
		}

	}

	var lval []byte

	lines := node.Lines().Len()
	for i := 0; i < lines; i++ {
		line := node.Lines().At(i)
		lval = append(lval, line.Value(source)...)
	}

	if lang == "d2" && slices.Contains(r.MarkConfig.Features, "d2") {
		scale := r.MarkConfig.D2Scale
		if scaleOverride > 0 {
			scale = scaleOverride
		}
		attachment, err := d2.ProcessD2(title, lval, scale)
		if err != nil {
			log.Debugf(nil, "error: %v", err)
			return ast.WalkStop, err
		}
		r.Attachments.Attach(attachment)

		effectiveAlign := calculateAlign(r.MarkConfig.ImageAlign, attachment.Width)
		effectiveLayout := calculateLayout(effectiveAlign, attachment.Width)
		displayWidth := calculateDisplayWidth(attachment.Width, effectiveLayout, widthOverride)

		err = r.Stdlib.Templates.ExecuteTemplate(
			writer,
			"ac:image",
			struct {
				Align          string
				Layout         string
				OriginalWidth  string
				OriginalHeight string
				Width          string
				Height         string
				Title          string
				Alt            string
				Attachment     string
				Url            string
			}{
				effectiveAlign,
				effectiveLayout,
				attachment.Width,
				attachment.Height,
				displayWidth,
				"",
				attachment.Name,
				"",
				attachment.Filename,
				"",
			},
		)

		if err != nil {
			return ast.WalkStop, err
		}

	} else if lang == "mermaid" && slices.Contains(r.MarkConfig.Features, "mermaid") {
		scale := r.MarkConfig.MermaidScale
		if scaleOverride > 0 {
			scale = scaleOverride
		}
		attachment, err := mermaid.ProcessMermaidLocally(title, lval, scale)
		if err != nil {
			log.Debugf(nil, "error: %v", err)
			return ast.WalkStop, err
		}
		r.Attachments.Attach(attachment)

		effectiveAlign := calculateAlign(r.MarkConfig.ImageAlign, attachment.Width)
		effectiveLayout := calculateLayout(effectiveAlign, attachment.Width)
		displayWidth := calculateDisplayWidth(attachment.Width, effectiveLayout, widthOverride)

		err = r.Stdlib.Templates.ExecuteTemplate(
			writer,
			"ac:image",
			struct {
				Align          string
				Layout         string
				OriginalWidth  string
				OriginalHeight string
				Width          string
				Height         string
				Title          string
				Alt            string
				Attachment     string
				Url            string
			}{
				effectiveAlign,
				effectiveLayout,
				attachment.Width,
				attachment.Height,
				displayWidth,
				"",
				attachment.Name,
				"",
				attachment.Filename,
				"",
			},
		)

		if err != nil {
			return ast.WalkStop, err
		}

	} else if lang == "ascii" && slices.Contains(r.MarkConfig.Features, "ascii") {
		scale := r.MarkConfig.ASCIIScale
		if scaleOverride > 0 {
			scale = scaleOverride
		}
		attachment, err := ascii.ProcessASCII(title, lval, scale)
		if err != nil {
			log.Debugf(nil, "error: %v", err)
			return ast.WalkStop, err
		}
		r.Attachments.Attach(attachment)

		effectiveAlign := calculateAlign(r.MarkConfig.ImageAlign, attachment.Width)
		effectiveLayout := calculateLayout(effectiveAlign, attachment.Width)
		displayWidth := calculateDisplayWidth(attachment.Width, effectiveLayout, widthOverride)

		err = r.Stdlib.Templates.ExecuteTemplate(
			writer,
			"ac:image",
			struct {
				Align          string
				Layout         string
				OriginalWidth  string
				OriginalHeight string
				Width          string
				Height         string
				Title          string
				Alt            string
				Attachment     string
				Url            string
			}{
				effectiveAlign,
				effectiveLayout,
				attachment.Width,
				attachment.Height,
				displayWidth,
				"",
				attachment.Name,
				"",
				attachment.Filename,
				"",
			},
		)

		if err != nil {
			return ast.WalkStop, err
		}

	} else {
		err := r.Stdlib.Templates.ExecuteTemplate(
			writer,
			"ac:code",
			struct {
				Language    string
				Collapse    bool
				Title       string
				Theme       string
				Linenumbers bool
				Firstline   int
				Text        string
			}{
				lang,
				collapse,
				title,
				theme,
				linenumbers,
				firstline,
				strings.TrimSuffix(string(lval), "\n"),
			},
		)

		if err != nil {
			return ast.WalkStop, err
		}
	}

	return ast.WalkContinue, nil
}
