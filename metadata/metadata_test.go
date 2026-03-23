package metadata

import (
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractDocumentLeadingH1(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "..")
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}

	filename = "testdata/header.md"

	markdown, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	actual := ExtractDocumentLeadingH1(markdown)

	assert.Equal(t, "a", actual)
}

func TestSetTitleFromFilename(t *testing.T) {
	t.Run("set title from filename", func(t *testing.T) {
		meta := &Meta{Title: ""}
		setTitleFromFilename(meta, "/path/to/test.md")
		assert.Equal(t, "Test", meta.Title)
	})

	t.Run("replace underscores with spaces", func(t *testing.T) {
		meta := &Meta{Title: ""}
		setTitleFromFilename(meta, "/path/to/test_with_underscores.md")
		assert.Equal(t, "Test With Underscores", meta.Title)
	})

	t.Run("replace dashes with spaces", func(t *testing.T) {
		meta := &Meta{Title: ""}
		setTitleFromFilename(meta, "/path/to/test-with-dashes.md")
		assert.Equal(t, "Test With Dashes", meta.Title)
	})

	t.Run("mixed underscores and dashes", func(t *testing.T) {
		meta := &Meta{Title: ""}
		setTitleFromFilename(meta, "/path/to/test_with-mixed_separators.md")
		assert.Equal(t, "Test With Mixed Separators", meta.Title)
	})

	t.Run("already title cased", func(t *testing.T) {
		meta := &Meta{Title: ""}
		setTitleFromFilename(meta, "/path/to/Already-Title-Cased.md")
		assert.Equal(t, "Already Title Cased", meta.Title)
	})
}

func TestExtractMetaContentAppearance(t *testing.T) {
	t.Run("default fills missing content appearance", func(t *testing.T) {
		data := []byte("<!-- Space: DOC -->\n<!-- Title: Example -->\n\nbody\n")

		meta, _, err := ExtractMeta(data, "", false, false, "", nil, false, FixedContentAppearance, "")
		assert.NoError(t, err)
		assert.NotNil(t, meta)
		assert.Equal(t, FixedContentAppearance, meta.ContentAppearance)
	})

	t.Run("header takes precedence over default", func(t *testing.T) {
		data := []byte("<!-- Space: DOC -->\n<!-- Title: Example -->\n<!-- Content-Appearance: full-width -->\n\nbody\n")

		meta, _, err := ExtractMeta(data, "", false, false, "", nil, false, FixedContentAppearance, "")
		assert.NoError(t, err)
		assert.NotNil(t, meta)
		assert.Equal(t, FullWidthContentAppearance, meta.ContentAppearance)
	})

	t.Run("falls back to full-width when default isn't set", func(t *testing.T) {
		data := []byte("<!-- Space: DOC -->\n<!-- Title: Example -->\n\nbody\n")

		meta, _, err := ExtractMeta(data, "", false, false, "", nil, false, "", "")
		assert.NoError(t, err)
		assert.NotNil(t, meta)
		assert.Equal(t, FullWidthContentAppearance, meta.ContentAppearance)
	})
}

func TestExtractMetaFolder(t *testing.T) {
	t.Run("parses folder header", func(t *testing.T) {
		data := []byte("<!-- Space: DOC -->\n<!-- Title: Example -->\n<!-- Folder: /project/docs/api -->\n\nbody\n")

		meta, _, err := ExtractMeta(data, "", false, false, "", nil, false, "", "")
		assert.NoError(t, err)
		assert.NotNil(t, meta)
		assert.Equal(t, "/project/docs/api", meta.Folder)
	})

	t.Run("CLI folder used as default when document has none", func(t *testing.T) {
		data := []byte("<!-- Space: DOC -->\n<!-- Title: Example -->\n\nbody\n")

		meta, _, err := ExtractMeta(data, "", false, false, "", nil, false, "", "/cli/folder")
		assert.NoError(t, err)
		assert.NotNil(t, meta)
		assert.Equal(t, "/cli/folder", meta.Folder)
	})

	t.Run("document folder takes precedence over CLI", func(t *testing.T) {
		data := []byte("<!-- Space: DOC -->\n<!-- Title: Example -->\n<!-- Folder: /doc/folder -->\n\nbody\n")

		meta, _, err := ExtractMeta(data, "", false, false, "", nil, false, "", "/cli/folder")
		assert.NoError(t, err)
		assert.NotNil(t, meta)
		assert.Equal(t, "/doc/folder", meta.Folder)
	})

	t.Run("folder combined with parent", func(t *testing.T) {
		data := []byte("<!-- Space: DOC -->\n<!-- Folder: /project/docs -->\n<!-- Parent: API Reference -->\n<!-- Title: Auth -->\n\nbody\n")

		meta, _, err := ExtractMeta(data, "", false, false, "", nil, false, "", "")
		assert.NoError(t, err)
		assert.NotNil(t, meta)
		assert.Equal(t, "/project/docs", meta.Folder)
		assert.Equal(t, []string{"API Reference"}, meta.Parents)
		assert.Equal(t, "Auth", meta.Title)
	})

	t.Run("empty folder is not set", func(t *testing.T) {
		data := []byte("<!-- Space: DOC -->\n<!-- Title: Example -->\n\nbody\n")

		meta, _, err := ExtractMeta(data, "", false, false, "", nil, false, "", "")
		assert.NoError(t, err)
		assert.NotNil(t, meta)
		assert.Equal(t, "", meta.Folder)
	})
}

func TestExtractMetaPageID(t *testing.T) {
	t.Run("parses PageID header", func(t *testing.T) {
		data := []byte("<!-- Space: DOC -->\n<!-- Title: Example -->\n<!-- PageID: 12345678 -->\n\nbody\n")

		meta, _, err := ExtractMeta(data, "", false, false, "", nil, false, "", "")
		assert.NoError(t, err)
		assert.NotNil(t, meta)
		assert.Equal(t, "12345678", meta.PageID)
	})

	t.Run("parses lowercase pageid header", func(t *testing.T) {
		data := []byte("<!-- Space: DOC -->\n<!-- pageid: 87654321 -->\n<!-- Title: Example -->\n\nbody\n")

		meta, _, err := ExtractMeta(data, "", false, false, "", nil, false, "", "")
		assert.NoError(t, err)
		assert.NotNil(t, meta)
		assert.Equal(t, "87654321", meta.PageID)
	})

	t.Run("PageID with whitespace is trimmed", func(t *testing.T) {
		data := []byte("<!-- Space: DOC -->\n<!-- Title: Example -->\n<!-- PageID:  123  -->\n\nbody\n")

		meta, _, err := ExtractMeta(data, "", false, false, "", nil, false, "", "")
		assert.NoError(t, err)
		assert.NotNil(t, meta)
		assert.Equal(t, "123", meta.PageID)
	})

	t.Run("empty when PageID not specified", func(t *testing.T) {
		data := []byte("<!-- Space: DOC -->\n<!-- Title: Example -->\n\nbody\n")

		meta, _, err := ExtractMeta(data, "", false, false, "", nil, false, "", "")
		assert.NoError(t, err)
		assert.NotNil(t, meta)
		assert.Equal(t, "", meta.PageID)
	})
}
