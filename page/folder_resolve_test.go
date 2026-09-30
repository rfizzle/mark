package page

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/kovetskiy/mark/v16/confluence"
	"github.com/kovetskiy/mark/v16/confluence/confluencetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// folderSite is a fake Cloud space DOC (ID 7, homepage 100) holding the
// folders below. Each folder is keyed by ID; parent "100" is the space root.
type folderSite struct {
	folders map[string]struct{ title, parent string }
}

func (s folderSite) handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /wiki/rest/api/space/DOC", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"id":7,"key":"DOC","homepage":{"id":"100","title":"Home"}}`)
	})

	mux.HandleFunc("GET /wiki/api/v2/pages/{id}/direct-children", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, s.children(r.PathValue("id")))
	})
	mux.HandleFunc("GET /wiki/api/v2/folders/{id}/direct-children", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, s.children(r.PathValue("id")))
	})

	mux.HandleFunc("GET /wiki/api/v2/folders/{id}", func(w http.ResponseWriter, r *http.Request) {
		f, ok := s.folders[r.PathValue("id")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		parentType := "folder"
		if f.parent == "100" {
			parentType = "page"
		}
		_, _ = fmt.Fprintf(w, `{"id":%q,"title":%q,"parentId":%q,"parentType":%q}`,
			r.PathValue("id"), f.title, f.parent, parentType)
	})

	mux.HandleFunc("GET /wiki/rest/api/search", func(w http.ResponseWriter, r *http.Request) {
		cql := r.URL.Query().Get("cql")
		var results []string
		for id, f := range s.folders {
			if strings.Contains(cql, fmt.Sprintf("title = %q", f.title)) {
				results = append(results, fmt.Sprintf(`{"content":{"id":%q,"type":"folder","title":%q}}`, id, f.title))
			}
		}
		_, _ = fmt.Fprintf(w, `{"results":[%s]}`, strings.Join(results, ","))
	})

	// Every folder title is taken somewhere in the space as far as these
	// tests go, so creation always fails as a duplicate.
	mux.HandleFunc("POST /wiki/api/v2/folders", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, `{"errors":[{"status":400,"title":"A folder exists with the same title in this space"}]}`)
	})

	return mux
}

func (s folderSite) children(parent string) string {
	var results []string
	for id, f := range s.folders {
		if f.parent == parent {
			results = append(results, fmt.Sprintf(`{"id":%q,"type":"folder","title":%q}`, id, f.title))
		}
	}
	return fmt.Sprintf(`{"results":[%s]}`, strings.Join(results, ","))
}

// site has /Team/Guides/ and /Archive/ (empty).
var site = folderSite{folders: map[string]struct{ title, parent string }{
	"800": {"Team", "100"},
	"900": {"Guides", "800"},
	"300": {"Archive", "100"},
}}

func TestResolveFolder_DryRunAllExist(t *testing.T) {
	api := confluencetest.NewAPI(t, confluencetest.ReadOnly(t, site.handler()))

	folder, err := ResolveFolder(true, api, "DOC", "/Team/Guides/")
	require.NoError(t, err)
	assert.Equal(t, "900", folder.ID)
}

func TestResolveFolder_DryRunWouldCreate(t *testing.T) {
	api := confluencetest.NewAPI(t, confluencetest.ReadOnly(t, site.handler()))

	folder, err := ResolveFolder(true, api, "DOC", "/Team/Drafts/Notes/")
	require.NoError(t, err)
	assert.Empty(t, folder.ID, "a folder that would be created has no ID")
	assert.Equal(t, "Notes", folder.Title)
}

func TestResolveFolder_DryRunTitleUsedElsewhere(t *testing.T) {
	api := confluencetest.NewAPI(t, confluencetest.ReadOnly(t, site.handler()))

	_, err := ResolveFolder(true, api, "DOC", "/Archive/Guides/")

	var conflict *confluence.FolderTitleConflictError
	require.True(t, errors.As(err, &conflict), "want FolderTitleConflictError, got %v", err)
	assert.Equal(t, "/Archive/", conflict.ParentPath)
	require.NotNil(t, conflict.Existing)
	assert.Equal(t, "/Team/Guides/", conflict.Existing.Path)
}

func TestResolveFolder_DryRunTitleUsedElsewhereBelowMissingFolder(t *testing.T) {
	api := confluencetest.NewAPI(t, confluencetest.ReadOnly(t, site.handler()))

	_, err := ResolveFolder(true, api, "DOC", "/Drafts/Guides/")

	var conflict *confluence.FolderTitleConflictError
	require.True(t, errors.As(err, &conflict), "want FolderTitleConflictError, got %v", err)
	assert.Equal(t, "/Drafts/", conflict.ParentPath)
	require.NotNil(t, conflict.Existing)
	assert.Equal(t, "900", conflict.Existing.ID)
}

func TestResolveFolder_TitleUsedElsewhereReportsParentPath(t *testing.T) {
	api := confluencetest.NewAPI(t, site.handler())

	_, err := ResolveFolder(false, api, "DOC", "/Archive/Guides/")
	require.Error(t, err)
	assert.Equal(t,
		`cannot create folder "Guides" under "/Archive/": Confluence folder titles are unique `+
			`within a space and "Guides" already exists at "/Team/Guides/" (id 900); `+
			`rename this Folder segment`,
		err.Error())
}

func TestFormatFolderPath(t *testing.T) {
	assert.Equal(t, "/", formatFolderPath(nil))
	assert.Equal(t, "/A/", formatFolderPath([]string{"A"}))
	assert.Equal(t, "/A/B/", formatFolderPath([]string{"A", "B"}))
}
