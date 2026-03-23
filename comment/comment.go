// Package comment provides utilities for preserving Confluence inline comment
// markers across page updates. Confluence anchors inline comments via
// ac:inline-comment-marker elements in the storage format body. When a page
// body is replaced wholesale these markers are lost, orphaning comments. This
// package extracts markers from an existing page body and merges them back
// into freshly compiled HTML where the anchored text still exists.
package comment

import (
	"regexp"
	"strings"
)

// markerRe matches a complete ac:inline-comment-marker element, capturing the
// ac:ref UUID (group 1) and the inner content (group 2). The inner content may
// contain nested HTML (bold, links, etc.) — markers never nest inside each
// other, so a non-greedy match with the explicit closing tag is sufficient.
var markerRe = regexp.MustCompile(
	`<ac:inline-comment-marker\s+ac:ref="([^"]+)">([\s\S]*?)</ac:inline-comment-marker>`,
)

// InlineCommentMarker represents a single inline comment anchor extracted from
// a Confluence page body.
type InlineCommentMarker struct {
	Ref         string // The ac:ref UUID
	MarkedText  string // The inner content wrapped by the marker
	FullElement string // The complete XML element including tags
}

// ExtractMarkers returns all inline comment markers found in the given
// Confluence storage-format HTML.
func ExtractMarkers(html string) []InlineCommentMarker {
	matches := markerRe.FindAllStringSubmatch(html, -1)
	markers := make([]InlineCommentMarker, 0, len(matches))
	for _, m := range matches {
		markers = append(markers, InlineCommentMarker{
			Ref:         m[1],
			MarkedText:  m[2],
			FullElement: m[0],
		})
	}
	return markers
}

// MergeMarkers re-inserts the given markers into newHTML wherever their
// MarkedText still appears verbatim. Markers whose text is no longer present
// are silently dropped (the comment will be orphaned — correct behaviour when
// the anchored text has changed). Markers are processed in order; positions
// already wrapped by a previous marker are not eligible for later ones.
func MergeMarkers(newHTML string, markers []InlineCommentMarker) string {
	// Track byte ranges already wrapped so overlapping markers don't collide.
	type span struct{ start, end int }
	var wrapped []span

	overlaps := func(s, e int) bool {
		for _, w := range wrapped {
			if s < w.end && e > w.start {
				return true
			}
		}
		return false
	}

	for _, m := range markers {
		// Find all occurrences and pick the first non-overlapping one.
		searchFrom := 0
		for {
			idx := strings.Index(newHTML[searchFrom:], m.MarkedText)
			if idx < 0 {
				break
			}
			absIdx := searchFrom + idx
			end := absIdx + len(m.MarkedText)
			if !overlaps(absIdx, end) {
				replacement := m.FullElement
				newHTML = newHTML[:absIdx] + replacement + newHTML[end:]
				wrapped = append(wrapped, span{absIdx, absIdx + len(replacement)})
				// Adjust existing spans that start after this insertion point
				// by the difference in length.
				delta := len(replacement) - len(m.MarkedText)
				for i := range wrapped[:len(wrapped)-1] {
					if wrapped[i].start > absIdx {
						wrapped[i].start += delta
						wrapped[i].end += delta
					}
				}
				break
			}
			searchFrom = absIdx + 1
		}
	}

	return newHTML
}

// StripMarkers removes all ac:inline-comment-marker wrapper elements from the
// given HTML, leaving only their inner content. This is used for clean hash
// computation with --changes-only so that the presence or absence of comment
// markers does not affect change detection.
func StripMarkers(html string) string {
	return markerRe.ReplaceAllString(html, "$2")
}
