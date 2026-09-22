package mark

import (
	"testing"

	"github.com/kovetskiy/mark/v16/confluence"
	"github.com/kovetskiy/mark/v16/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ancestor = struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func pageWithAncestors(id string, ancestors ...ancestor) *confluence.PageInfo {
	pg := &confluence.PageInfo{ID: id, Title: "Page " + id, Type: "page"}
	pg.Ancestors = ancestors
	return pg
}

func TestCurrentParentID(t *testing.T) {
	assert.Equal(t, "", currentParentID(pageWithAncestors("1")))

	pg := pageWithAncestors("1", ancestor{ID: "10", Title: "Root"}, ancestor{ID: "20", Title: "Parent"})
	assert.Equal(t, "20", currentParentID(pg), "should use the last (immediate) ancestor")
}

// A non-Cloud API: IsCloud() is false, so movePageIfNeeded never makes a
// network call and instead rewrites ancestors for the v1 UpdatePage.
func serverAPI() *confluence.API {
	return confluence.NewAPI("http://confluence.example.internal", "", "", false)
}

func TestMovePageIfNeeded_NilParent(t *testing.T) {
	pg := pageWithAncestors("1", ancestor{ID: "20", Title: "Parent"})

	got, err := movePageIfNeeded(serverAPI(), pg, nil, &metadata.Meta{})
	require.NoError(t, err)
	assert.Same(t, pg, got)
	assert.Equal(t, "20", currentParentID(got), "ancestors must be untouched")
}

func TestMovePageIfNeeded_AlreadyUnderParent(t *testing.T) {
	pg := pageWithAncestors("1", ancestor{ID: "10", Title: "Root"}, ancestor{ID: "20", Title: "Parent"})
	parent := &confluence.PageInfo{ID: "20", Title: "Parent"}

	got, err := movePageIfNeeded(serverAPI(), pg, parent, &metadata.Meta{Folder: "docs"})
	require.NoError(t, err)
	assert.Same(t, pg, got)
	assert.Len(t, got.Ancestors, 2, "no-op move must not rewrite ancestors")
}

func TestMovePageIfNeeded_RewritesAncestorsWhenNotCloud(t *testing.T) {
	pg := pageWithAncestors("1", ancestor{ID: "10", Title: "Root"}, ancestor{ID: "20", Title: "Old"})
	parent := &confluence.PageInfo{ID: "30", Title: "New"}

	got, err := movePageIfNeeded(serverAPI(), pg, parent, &metadata.Meta{})
	require.NoError(t, err)
	require.Len(t, got.Ancestors, 1)
	assert.Equal(t, "30", got.Ancestors[0].ID)
	assert.Equal(t, "New", got.Ancestors[0].Title)
}

func TestMovePageIfNeeded_NoAncestorsGetsParent(t *testing.T) {
	pg := pageWithAncestors("1")
	parent := &confluence.PageInfo{ID: "30", Title: "New"}

	got, err := movePageIfNeeded(serverAPI(), pg, parent, &metadata.Meta{})
	require.NoError(t, err)
	assert.Equal(t, "30", currentParentID(got))
}
