package mark

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/kovetskiy/mark/v16/confluence"
	"github.com/kovetskiy/mark/v16/confluence/confluencetest"
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

func TestCheckPageNotArchived(t *testing.T) {
	assert.NoError(t, checkPageNotArchived(nil, "1"))
	assert.NoError(t, checkPageNotArchived(&confluence.PageInfo{ID: "1", Status: "current"}, "1"))
	assert.NoError(t, checkPageNotArchived(&confluence.PageInfo{ID: "1"}, "1"), "missing status is not archived")

	err := checkPageNotArchived(&confluence.PageInfo{ID: "1", Title: "Old", Status: "archived"}, "1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `PageID "1"`)
	assert.Contains(t, err.Error(), "restore the page in Confluence, or stop syncing this file")
}

// archivedPageAPI serves page 42 as archived and fails the test on any other
// request, so a sync must stop right after looking the page up.
func archivedPageAPI(t *testing.T) *confluence.API {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /wiki/rest/api/content/42", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"id":"42","type":"page","status":"archived","title":"Old Page","version":{"number":3}}`)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request after finding an archived page: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	})
	return confluencetest.NewAPI(t, mux)
}

func writeMarkdown(t *testing.T, content string) string {
	path := filepath.Join(t.TempDir(), "page.md")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

const archivedPageMarkdown = `<!-- Space: DOC -->
<!-- Title: Old Page -->
<!-- PageID: 42 -->
<!-- Folder: /Archive/ -->

# Old Page
`

func TestProcessFile_ArchivedPageFails(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		t.Run(fmt.Sprintf("dryRun=%v", dryRun), func(t *testing.T) {
			file := writeMarkdown(t, archivedPageMarkdown)

			_, err := ProcessFile(file, archivedPageAPI(t), Config{DryRun: dryRun})
			require.Error(t, err)
			assert.Contains(t, err.Error(), `page with PageID "42" ("Old Page") is archived`)
		})
	}
}

func TestProcessFile_ArchivedTargetURLPageFails(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		t.Run(fmt.Sprintf("dryRun=%v", dryRun), func(t *testing.T) {
			file := writeMarkdown(t, "# Old Page\n")

			_, err := ProcessFile(file, archivedPageAPI(t), Config{PageID: "42", DryRun: dryRun})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "is archived")
		})
	}
}
