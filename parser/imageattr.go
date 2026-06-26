package parser

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type ImageAttrTransformer struct{}

func NewImageAttrTransformer() parser.ASTTransformer {
	return &ImageAttrTransformer{}
}

func (t *ImageAttrTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	source := reader.Source()
	var toRemove []ast.Node

	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || n.Kind() != ast.KindImage {
			return ast.WalkContinue, nil
		}

		next := n.NextSibling()
		if next == nil || next.Kind() != ast.KindText {
			return ast.WalkContinue, nil
		}

		textNode := next.(*ast.Text)
		seg := textNode.Segment
		raw := seg.Value(source)

		attrs, consumed := parseInlineAttrs(raw)
		if consumed == 0 {
			return ast.WalkContinue, nil
		}

		for _, attr := range attrs {
			n.SetAttribute(attr.Name, attr.Value)
		}

		if consumed >= len(raw) {
			toRemove = append(toRemove, next)
		} else {
			textNode.Segment = text.NewSegment(seg.Start+consumed, seg.Stop)
		}

		return ast.WalkContinue, nil
	})

	for _, n := range toRemove {
		n.Parent().RemoveChild(n.Parent(), n)
	}
}

// parseInlineAttrs parses a leading {key=value, ...} from raw bytes.
// Returns the parsed attributes and the number of bytes consumed.
func parseInlineAttrs(raw []byte) ([]parser.Attribute, int) {
	if len(raw) == 0 || raw[0] != '{' {
		return nil, 0
	}

	end := -1
	for i := 1; i < len(raw); i++ {
		if raw[i] == '}' {
			end = i
			break
		}
	}
	if end < 0 {
		return nil, 0
	}

	inner := raw[1:end]
	attrs := parseAttrPairs(inner)
	if len(attrs) == 0 {
		return nil, 0
	}

	return attrs, end + 1
}

// parseAttrPairs parses "key=value" pairs separated by spaces or commas.
func parseAttrPairs(data []byte) []parser.Attribute {
	var attrs []parser.Attribute
	i := 0
	for i < len(data) {
		// Skip whitespace and commas
		for i < len(data) && (data[i] == ' ' || data[i] == ',') {
			i++
		}
		if i >= len(data) {
			break
		}

		// Read key
		keyStart := i
		for i < len(data) && data[i] != '=' && data[i] != ' ' && data[i] != ',' {
			i++
		}
		if i >= len(data) || data[i] != '=' {
			return nil
		}
		key := data[keyStart:i]
		i++ // skip '='

		// Read value
		valStart := i
		for i < len(data) && data[i] != ' ' && data[i] != ',' && data[i] != '}' {
			i++
		}
		val := data[valStart:i]
		if len(key) == 0 || len(val) == 0 {
			return nil
		}

		attrs = append(attrs, parser.Attribute{
			Name:  key,
			Value: val,
		})
	}
	return attrs
}
