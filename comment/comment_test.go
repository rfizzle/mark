package comment

import (
	"testing"
)

func TestExtractMarkersEmpty(t *testing.T) {
	markers := ExtractMarkers("<p>No markers here</p>")
	if len(markers) != 0 {
		t.Fatalf("expected 0 markers, got %d", len(markers))
	}
}

func TestExtractMarkersSingle(t *testing.T) {
	html := `<p>Some text <ac:inline-comment-marker ac:ref="abc-123">highlighted</ac:inline-comment-marker> more text</p>`
	markers := ExtractMarkers(html)
	if len(markers) != 1 {
		t.Fatalf("expected 1 marker, got %d", len(markers))
	}
	if markers[0].Ref != "abc-123" {
		t.Errorf("expected ref %q, got %q", "abc-123", markers[0].Ref)
	}
	if markers[0].MarkedText != "highlighted" {
		t.Errorf("expected marked text %q, got %q", "highlighted", markers[0].MarkedText)
	}
}

func TestExtractMarkersMultiple(t *testing.T) {
	html := `<p><ac:inline-comment-marker ac:ref="id-1">first</ac:inline-comment-marker> and <ac:inline-comment-marker ac:ref="id-2">second</ac:inline-comment-marker></p>`
	markers := ExtractMarkers(html)
	if len(markers) != 2 {
		t.Fatalf("expected 2 markers, got %d", len(markers))
	}
	if markers[0].Ref != "id-1" || markers[1].Ref != "id-2" {
		t.Errorf("unexpected refs: %q, %q", markers[0].Ref, markers[1].Ref)
	}
}

func TestExtractMarkersNestedHTML(t *testing.T) {
	html := `<ac:inline-comment-marker ac:ref="rich-1"><strong>bold text</strong> and <a href="http://example.com">a link</a></ac:inline-comment-marker>`
	markers := ExtractMarkers(html)
	if len(markers) != 1 {
		t.Fatalf("expected 1 marker, got %d", len(markers))
	}
	expected := `<strong>bold text</strong> and <a href="http://example.com">a link</a>`
	if markers[0].MarkedText != expected {
		t.Errorf("expected marked text %q, got %q", expected, markers[0].MarkedText)
	}
}

func TestMergeMarkersTextPresent(t *testing.T) {
	oldHTML := `<p><ac:inline-comment-marker ac:ref="abc-123">important text</ac:inline-comment-marker></p>`
	newHTML := `<p>important text</p>`

	markers := ExtractMarkers(oldHTML)
	result := MergeMarkers(newHTML, markers)

	expected := `<p><ac:inline-comment-marker ac:ref="abc-123">important text</ac:inline-comment-marker></p>`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestMergeMarkersTextChanged(t *testing.T) {
	oldHTML := `<p><ac:inline-comment-marker ac:ref="abc-123">old text</ac:inline-comment-marker></p>`
	newHTML := `<p>completely different text</p>`

	markers := ExtractMarkers(oldHTML)
	result := MergeMarkers(newHTML, markers)

	// Marker should be dropped since "old text" is not in newHTML.
	if result != newHTML {
		t.Errorf("expected unchanged HTML:\n%s\ngot:\n%s", newHTML, result)
	}
}

func TestMergeMarkersMultiple(t *testing.T) {
	oldHTML := `<p><ac:inline-comment-marker ac:ref="id-1">alpha</ac:inline-comment-marker> and <ac:inline-comment-marker ac:ref="id-2">beta</ac:inline-comment-marker></p>`
	newHTML := `<p>alpha and beta</p>`

	markers := ExtractMarkers(oldHTML)
	result := MergeMarkers(newHTML, markers)

	expected := `<p><ac:inline-comment-marker ac:ref="id-1">alpha</ac:inline-comment-marker> and <ac:inline-comment-marker ac:ref="id-2">beta</ac:inline-comment-marker></p>`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestMergeMarkersPartialMatch(t *testing.T) {
	oldHTML := `<p><ac:inline-comment-marker ac:ref="id-1">kept</ac:inline-comment-marker> and <ac:inline-comment-marker ac:ref="id-2">removed</ac:inline-comment-marker></p>`
	newHTML := `<p>kept and replaced</p>`

	markers := ExtractMarkers(oldHTML)
	result := MergeMarkers(newHTML, markers)

	expected := `<p><ac:inline-comment-marker ac:ref="id-1">kept</ac:inline-comment-marker> and replaced</p>`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestMergeMarkersOverlapping(t *testing.T) {
	// Two markers whose text overlaps in the new HTML.
	oldHTML := `<ac:inline-comment-marker ac:ref="id-1">AB</ac:inline-comment-marker><ac:inline-comment-marker ac:ref="id-2">BC</ac:inline-comment-marker>`
	newHTML := `ABC`

	markers := ExtractMarkers(oldHTML)
	result := MergeMarkers(newHTML, markers)

	// "AB" is found first and wrapped; "BC" now spans across the marker boundary
	// and should not be found for wrapping.
	expected := `<ac:inline-comment-marker ac:ref="id-1">AB</ac:inline-comment-marker>C`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestMergeMarkersNestedHTML(t *testing.T) {
	oldHTML := `<ac:inline-comment-marker ac:ref="rich-1"><strong>bold</strong></ac:inline-comment-marker>`
	newHTML := `<p><strong>bold</strong> paragraph</p>`

	markers := ExtractMarkers(oldHTML)
	result := MergeMarkers(newHTML, markers)

	expected := `<p><ac:inline-comment-marker ac:ref="rich-1"><strong>bold</strong></ac:inline-comment-marker> paragraph</p>`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestStripMarkers(t *testing.T) {
	html := `<p><ac:inline-comment-marker ac:ref="abc-123">highlighted text</ac:inline-comment-marker> and <ac:inline-comment-marker ac:ref="def-456"><strong>bold</strong></ac:inline-comment-marker></p>`
	result := StripMarkers(html)
	expected := `<p>highlighted text and <strong>bold</strong></p>`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestStripMarkersNoMarkers(t *testing.T) {
	html := `<p>plain text</p>`
	result := StripMarkers(html)
	if result != html {
		t.Errorf("expected unchanged HTML, got:\n%s", result)
	}
}

func TestMergeMarkersDuplicateText(t *testing.T) {
	// Same text appears twice; only the first occurrence should be wrapped.
	oldHTML := `<ac:inline-comment-marker ac:ref="id-1">hello</ac:inline-comment-marker>`
	newHTML := `<p>hello world hello</p>`

	markers := ExtractMarkers(oldHTML)
	result := MergeMarkers(newHTML, markers)

	expected := `<p><ac:inline-comment-marker ac:ref="id-1">hello</ac:inline-comment-marker> world hello</p>`
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}
