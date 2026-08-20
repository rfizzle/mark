package parser

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var colwidthRegexp = regexp.MustCompile(`(?i)^\s*<!--\s*colwidth:\s*(.*?)\s*-->\s*$`)

// cloudContentWidth is the width in pixels of Confluence Cloud's default
// (fixed-width) content area, used to convert percentage widths to pixels.
const cloudContentWidth = 760

type ColWidthTransformer struct {
	// cloud enables conversion of percentage widths to pixels. The Confluence
	// Cloud editor (fabric) stores column widths only as pixel values in its
	// document model; percentage widths render on a freshly published page but
	// are dropped the first time the page round-trips through the editor.
	cloud bool
}

func NewColWidthTransformer(cloud bool) parser.ASTTransformer {
	return &ColWidthTransformer{cloud: cloud}
}

func (t *ColWidthTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	var toRemove []ast.Node

	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || n.Kind() != ast.KindHTMLBlock {
			return ast.WalkContinue, nil
		}

		lines := n.Lines()
		if lines.Len() == 0 {
			return ast.WalkContinue, nil
		}

		var raw []byte
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			raw = append(raw, seg.Value(reader.Source())...)
		}

		matches := colwidthRegexp.FindSubmatch(raw)
		if matches == nil {
			return ast.WalkContinue, nil
		}

		widths := parseWidths(string(matches[1]))
		if t.cloud {
			widths = convertPercentsToPixels(widths)
		}

		// Always remove the directive comment from output
		toRemove = append(toRemove, n)

		if len(widths) == 0 {
			return ast.WalkContinue, nil
		}

		next := n.NextSibling()
		if next == nil || next.Kind() != east.KindTable {
			return ast.WalkContinue, nil
		}

		// Validate column count matches the table
		table := next.(*east.Table)
		if len(widths) != len(table.Alignments) {
			return ast.WalkContinue, nil
		}

		next.SetAttributeString("data-colwidths", widths)

		return ast.WalkContinue, nil
	})

	for _, n := range toRemove {
		n.Parent().RemoveChild(n.Parent(), n)
	}
}

func parseWidths(s string) []string {
	parts := strings.Split(s, ",")
	var widths []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		p = normalizeWidth(p)
		if !isValidCSSWidth(p) {
			continue
		}
		widths = append(widths, p)
	}
	return widths
}

// convertPercentsToPixels rewrites percentage widths as pixel widths relative
// to Confluence Cloud's default content width. Non-percentage values are kept
// as-is.
func convertPercentsToPixels(widths []string) []string {
	converted := make([]string, len(widths))
	for i, w := range widths {
		pct, ok := strings.CutSuffix(w, "%")
		if !ok {
			converted[i] = w
			continue
		}
		value, err := strconv.ParseFloat(pct, 64)
		if err != nil {
			converted[i] = w
			continue
		}
		converted[i] = fmt.Sprintf("%dpx", int(math.Round(value/100*cloudContentWidth)))
	}
	return converted
}

// normalizeWidth appends "px" to bare numeric values.
func normalizeWidth(s string) string {
	for _, c := range s {
		if !unicode.IsDigit(c) && c != '.' {
			return s
		}
	}
	return s + "px"
}

// isValidCSSWidth validates that a width value is a safe CSS length.
// Allows: digits, dots, percent signs, and known unit suffixes.
var validWidthRegexp = regexp.MustCompile(`^(\d+(\.\d+)?(px|%|em|rem|ch|vw|vh|ex|cm|mm|in|pt|pc)?|auto)$`)

func isValidCSSWidth(s string) bool {
	return validWidthRegexp.MatchString(s)
}
