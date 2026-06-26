package parser

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var colwidthRegexp = regexp.MustCompile(`(?i)<!--\s*colwidth:\s*(.+?)\s*-->`)

type ColWidthTransformer struct{}

func NewColWidthTransformer() parser.ASTTransformer {
	return &ColWidthTransformer{}
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
		if len(widths) == 0 {
			return ast.WalkContinue, nil
		}

		next := n.NextSibling()
		if next == nil || next.Kind() != east.KindTable {
			toRemove = append(toRemove, n)
			return ast.WalkContinue, nil
		}

		next.SetAttributeString("data-colwidths", widths)
		toRemove = append(toRemove, n)

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
		widths = append(widths, p)
	}
	return widths
}
