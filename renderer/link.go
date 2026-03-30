package renderer

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

type ConfluenceLinkRenderer struct {
	html.Config
}

// NewConfluenceRenderer creates a new instance of the ConfluenceRenderer
func NewConfluenceLinkRenderer(opts ...html.Option) renderer.NodeRenderer {
	return &ConfluenceLinkRenderer{
		Config: html.NewConfig(),
	}
}

// RegisterFuncs implements NodeRenderer.RegisterFuncs .
func (r *ConfluenceLinkRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindLink, r.renderLink)
}

// renderLink renders links specifically for confluence
func (r *ConfluenceLinkRenderer) renderLink(writer util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Link)
	if len(n.Destination) >= 3 && string(n.Destination[0:3]) == "ac:" {
		if entering {
			if len(n.Destination) >= 6 && string(n.Destination[0:6]) == "ac:id:" {
				// PageID-based link: ac:id:12345 or ac:id:12345#anchor
				pageRef := string(n.Destination[6:])
				var pageID, anchor string
				if idx := strings.IndexByte(pageRef, '#'); idx >= 0 {
					pageID = pageRef[:idx]
					anchor = pageRef[idx+1:]
				} else {
					pageID = pageRef
				}

				if pageID == "" {
					// Empty page ID: fall back to using link text as page title
					_, err := writer.Write([]byte("<ac:link><ri:page ri:content-title=\""))
					if err != nil {
						return ast.WalkStop, err
					}
					//nolint:staticcheck
					_, err = writer.Write(node.Text(source))
					if err != nil {
						return ast.WalkStop, err
					}
					_, err = writer.Write([]byte("\"/>"))
					if err != nil {
						return ast.WalkStop, err
					}
				} else {
					_, err := writer.Write([]byte("<ac:link"))
					if err != nil {
						return ast.WalkStop, err
					}
					if anchor != "" {
						_, err = writer.Write([]byte(" ac:anchor=\"" + anchor + "\""))
						if err != nil {
							return ast.WalkStop, err
						}
					}
					_, err = writer.Write([]byte("><ri:content-entity ri:content-id=\"" + pageID + "\"/>"))
					if err != nil {
						return ast.WalkStop, err
					}
				}

				_, err := writer.Write([]byte("<ac:plain-text-link-body><![CDATA["))
				if err != nil {
					return ast.WalkStop, err
				}
				//nolint:staticcheck
				_, err = writer.Write(node.Text(source))
				if err != nil {
					return ast.WalkStop, err
				}
				_, err = writer.Write([]byte("]]></ac:plain-text-link-body></ac:link>"))
				if err != nil {
					return ast.WalkStop, err
				}
			} else {
				// Title-based link: ac:PageTitle
				_, err := writer.Write([]byte("<ac:link><ri:page ri:content-title=\""))
				if err != nil {
					return ast.WalkStop, err
				}

				if len(string(n.Destination)) < 4 {
					//nolint:staticcheck
					_, err := writer.Write(node.Text(source))
					if err != nil {
						return ast.WalkStop, err
					}
				} else {
					_, err := writer.Write(n.Destination[3:])
					if err != nil {
						return ast.WalkStop, err
					}
				}
				_, err = writer.Write([]byte("\"/><ac:plain-text-link-body><![CDATA["))
				if err != nil {
					return ast.WalkStop, err
				}

				//nolint:staticcheck
				_, err = writer.Write(node.Text(source))
				if err != nil {
					return ast.WalkStop, err
				}

				_, err = writer.Write([]byte("]]></ac:plain-text-link-body></ac:link>"))
				if err != nil {
					return ast.WalkStop, err
				}
			}
		}
		return ast.WalkSkipChildren, nil
	}
	return r.goldmarkRenderLink(writer, source, node, entering)
}

// goldmarkRenderLink is the default renderLink implementation from https://github.com/yuin/goldmark/blob/9d6f314b99ca23037c93d76f248be7b37de6220a/renderer/html/html.go#L552
func (r *ConfluenceLinkRenderer) goldmarkRenderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Link)
	if entering {
		_, _ = w.WriteString("<a href=\"")
		if r.Unsafe || !html.IsDangerousURL(n.Destination) {
			_, _ = w.Write(util.EscapeHTML(util.URLEscape(n.Destination, true)))
		}
		_ = w.WriteByte('"')
		if n.Title != nil {
			_, _ = w.WriteString(` title="`)
			r.Writer.Write(w, n.Title)
			_ = w.WriteByte('"')
		}
		if n.Attributes() != nil {
			html.RenderAttributes(w, n, html.LinkAttributeFilter)
		}
		_ = w.WriteByte('>')
	} else {
		_, _ = w.WriteString("</a>")
	}
	return ast.WalkContinue, nil
}
