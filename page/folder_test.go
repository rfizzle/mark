package page

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFolderPath(t *testing.T) {
	t.Run("simple path", func(t *testing.T) {
		segments, err := ParseFolderPath("/a/b/c")
		assert.NoError(t, err)
		assert.Equal(t, []string{"a", "b", "c"}, segments)
	})

	t.Run("no leading slash", func(t *testing.T) {
		segments, err := ParseFolderPath("a/b")
		assert.NoError(t, err)
		assert.Equal(t, []string{"a", "b"}, segments)
	})

	t.Run("trailing slash", func(t *testing.T) {
		segments, err := ParseFolderPath("/single/")
		assert.NoError(t, err)
		assert.Equal(t, []string{"single"}, segments)
	})

	t.Run("single segment", func(t *testing.T) {
		segments, err := ParseFolderPath("docs")
		assert.NoError(t, err)
		assert.Equal(t, []string{"docs"}, segments)
	})

	t.Run("empty path", func(t *testing.T) {
		_, err := ParseFolderPath("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("only slashes", func(t *testing.T) {
		_, err := ParseFolderPath("///")
		assert.Error(t, err)
	})

	t.Run("double slash produces empty segment", func(t *testing.T) {
		_, err := ParseFolderPath("a//b")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty segment")
	})

	t.Run("spaces around segments are trimmed", func(t *testing.T) {
		segments, err := ParseFolderPath("/ project / docs /")
		assert.NoError(t, err)
		assert.Equal(t, []string{"project", "docs"}, segments)
	})
}
