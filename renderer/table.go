package renderer

import (
	"fmt"

	east "github.com/yuin/goldmark/extension/ast"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

var TableAttributeFilter = html.GlobalAttributeFilter.ExtendString(`align,bgcolor,border,cellpadding,cellspacing,frame,rules,summary,width`)

type ConfluenceTableRenderer struct {
	html.Config
}

func NewConfluenceTableRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &ConfluenceTableRenderer{
		Config: html.NewConfig(),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

func (r *ConfluenceTableRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(east.KindTable, r.renderTable)
	reg.Register(east.KindTableHeader, r.renderTableHeader)
	reg.Register(east.KindTableRow, r.renderTableRow)
	reg.Register(east.KindTableCell, r.renderTableCell)
}

func (r *ConfluenceTableRenderer) renderTable(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		var widths []string
		if v, ok := n.AttributeString("data-colwidths"); ok {
			widths, _ = v.([]string)
			n.RemoveAttributes()
		}

		_, _ = w.WriteString("<table")
		if n.Attributes() != nil {
			html.RenderAttributes(w, n, TableAttributeFilter)
		}
		_, _ = w.WriteString(">\n")

		if len(widths) > 0 {
			_, _ = w.WriteString("<colgroup>\n")
			for _, width := range widths {
				_, _ = fmt.Fprintf(w, "<col style=\"width: %s\" />\n", width)
			}
			_, _ = w.WriteString("</colgroup>\n")
		}
	} else {
		_, _ = w.WriteString("</table>\n")
	}
	return ast.WalkContinue, nil
}

func (r *ConfluenceTableRenderer) renderTableHeader(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("<thead>\n")
		_, _ = w.WriteString("<tr>\n")
	} else {
		_, _ = w.WriteString("</tr>\n")
		_, _ = w.WriteString("</thead>\n")
		if n.NextSibling() != nil {
			_, _ = w.WriteString("<tbody>\n")
		}
	}
	return ast.WalkContinue, nil
}

func (r *ConfluenceTableRenderer) renderTableRow(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("<tr>\n")
	} else {
		_, _ = w.WriteString("</tr>\n")
		if n.Parent().LastChild() == n {
			_, _ = w.WriteString("</tbody>\n")
		}
	}
	return ast.WalkContinue, nil
}

func (r *ConfluenceTableRenderer) renderTableCell(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*east.TableCell)
	tag := "td"
	if n.Parent().Kind() == east.KindTableHeader {
		tag = "th"
	}
	if entering {
		_, _ = fmt.Fprintf(w, "<%s", tag)
		if n.Alignment != east.AlignNone {
			_, _ = fmt.Fprintf(w, " style=\"text-align:%s\"", n.Alignment.String())
		}
		_ = w.WriteByte('>')
	} else {
		_, _ = fmt.Fprintf(w, "</%s>\n", tag)
	}
	return ast.WalkContinue, nil
}
