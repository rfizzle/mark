package page

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPageID(t *testing.T) {
	t.Run("pure numeric string is a page ID", func(t *testing.T) {
		assert.True(t, isPageID("123456789"))
	})

	t.Run("zero is a page ID", func(t *testing.T) {
		assert.True(t, isPageID("0"))
	})

	t.Run("large number is a page ID", func(t *testing.T) {
		assert.True(t, isPageID("18446744073709551615")) // max uint64
	})

	t.Run("title string is not a page ID", func(t *testing.T) {
		assert.False(t, isPageID("API Reference"))
	})

	t.Run("mixed alphanumeric is not a page ID", func(t *testing.T) {
		assert.False(t, isPageID("123abc"))
	})

	t.Run("empty string is not a page ID", func(t *testing.T) {
		assert.False(t, isPageID(""))
	})

	t.Run("negative number is not a page ID", func(t *testing.T) {
		assert.False(t, isPageID("-1"))
	})

	t.Run("decimal is not a page ID", func(t *testing.T) {
		assert.False(t, isPageID("123.456"))
	})

	t.Run("string with spaces is not a page ID", func(t *testing.T) {
		assert.False(t, isPageID("123 456"))
	})
}
