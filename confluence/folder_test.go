package confluence_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/kovetskiy/mark/v16/confluence"
	"github.com/kovetskiy/mark/v16/confluence/confluencetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// duplicateFolderBody stands in for Confluence Cloud's reply when POST /folders
// is given a title already used by a folder anywhere in the space. mark only
// relies on the "exists with the same title" phrase in it.
const duplicateFolderBody = `{"errors":[{"status":400,"code":"BAD_REQUEST",` +
	`"title":"A folder exists with the same title in this space","detail":null}]}`

// space serves the space lookups every folder operation needs: space DOC has
// ID 7 and homepage 100.
func space(mux *http.ServeMux) {
	mux.HandleFunc("GET /wiki/rest/api/space/DOC", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"id":7,"key":"DOC","homepage":{"id":"100","title":"Home"}}`)
	})
}

func duplicateOnCreate(mux *http.ServeMux) {
	mux.HandleFunc("POST /wiki/api/v2/folders", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, duplicateFolderBody)
	})
}

// serve answers GET path with body.
func serve(mux *http.ServeMux, path, body string) {
	mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, body)
	})
}

func search(mux *http.ServeMux, body string) {
	mux.HandleFunc("GET /wiki/rest/api/search", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, body)
	})
}

const noResults = `{"results":[]}`

func TestFindOrCreateFolder_Created(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /wiki/api/v2/folders", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"id":"500","title":"Guides","spaceId":"7","parentId":"200"}`)
	})
	api := confluencetest.NewAPI(t, mux)

	folder, created, err := api.FindOrCreateFolder("7", "DOC", "Guides", "200")
	require.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, "500", folder.ID)
}

func TestFindOrCreateFolder_ExistsUnderParent(t *testing.T) {
	mux := http.NewServeMux()
	duplicateOnCreate(mux)
	serve(mux, "/wiki/api/v2/folders/200/direct-children",
		`{"results":[{"id":"500","type":"folder","title":"Guides"}]}`)
	api := confluencetest.NewAPI(t, mux)

	folder, created, err := api.FindOrCreateFolder("7", "DOC", "Guides", "200")
	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, "500", folder.ID)
}

func TestFindOrCreateFolder_TitleUsedElsewhere(t *testing.T) {
	mux := http.NewServeMux()
	space(mux)
	duplicateOnCreate(mux)
	serve(mux, "/wiki/api/v2/folders/200/direct-children", noResults)
	search(mux, `{"results":[{"content":{"id":"900","type":"folder","title":"Guides"}}]}`)
	serve(mux, "/wiki/api/v2/folders/900", `{"id":"900","title":"Guides","parentId":"800","parentType":"folder"}`)
	serve(mux, "/wiki/api/v2/folders/800", `{"id":"800","title":"Team","parentId":"100","parentType":"page"}`)
	api := confluencetest.NewAPI(t, mux)

	_, _, err := api.FindOrCreateFolder("7", "DOC", "Guides", "200")

	var conflict *confluence.FolderTitleConflictError
	require.True(t, errors.As(err, &conflict), "want FolderTitleConflictError, got %v", err)
	assert.Equal(t, "Guides", conflict.Title)
	require.NotNil(t, conflict.Existing)
	assert.Equal(t, confluence.FolderLocation{ID: "900", Title: "Guides", Path: "/Team/Guides/"}, *conflict.Existing)
	assert.Contains(t, err.Error(), `"Guides" already exists at "/Team/Guides/" (id 900)`)
	assert.Contains(t, err.Error(), "rename this Folder segment")
}

func TestFindOrCreateFolder_TitleUsedElsewhereAtRoot(t *testing.T) {
	mux := http.NewServeMux()
	space(mux)
	duplicateOnCreate(mux)
	serve(mux, "/wiki/api/v2/pages/100/direct-children", noResults)
	search(mux, `{"results":[{"content":{"id":"900","type":"folder","title":"Guides"}}]}`)
	serve(mux, "/wiki/api/v2/folders/900", `{"id":"900","title":"Guides","parentId":"300","parentType":"folder"}`)
	serve(mux, "/wiki/api/v2/folders/300", `{"id":"300","title":"Archive","parentId":"100","parentType":"page"}`)
	api := confluencetest.NewAPI(t, mux)

	_, _, err := api.FindOrCreateFolder("7", "DOC", "Guides", "")

	var conflict *confluence.FolderTitleConflictError
	require.True(t, errors.As(err, &conflict), "want FolderTitleConflictError, got %v", err)
	require.NotNil(t, conflict.Existing)
	assert.Equal(t, "/Archive/Guides/", conflict.Existing.Path)
}

func TestFindOrCreateFolder_PrefersExactTitleMatch(t *testing.T) {
	mux := http.NewServeMux()
	space(mux)
	duplicateOnCreate(mux)
	serve(mux, "/wiki/api/v2/folders/200/direct-children", noResults)
	search(mux, `{"results":[`+
		`{"content":{"id":"901","type":"folder","title":"guides"}},`+
		`{"content":{"id":"900","type":"folder","title":"Guides"}}]}`)
	serve(mux, "/wiki/api/v2/folders/900", `{"id":"900","title":"Guides","parentId":"100","parentType":"page"}`)
	api := confluencetest.NewAPI(t, mux)

	_, _, err := api.FindOrCreateFolder("7", "DOC", "Guides", "200")

	var conflict *confluence.FolderTitleConflictError
	require.True(t, errors.As(err, &conflict), "want FolderTitleConflictError, got %v", err)
	require.NotNil(t, conflict.Existing)
	assert.Equal(t, "900", conflict.Existing.ID)
}

func TestFindOrCreateFolder_TitleUsedButNotFound(t *testing.T) {
	mux := http.NewServeMux()
	duplicateOnCreate(mux)
	serve(mux, "/wiki/api/v2/folders/200/direct-children", noResults)
	search(mux, noResults)
	api := confluencetest.NewAPI(t, mux)

	_, _, err := api.FindOrCreateFolder("7", "DOC", "Guides", "200")

	var conflict *confluence.FolderTitleConflictError
	require.True(t, errors.As(err, &conflict), "want FolderTitleConflictError, got %v", err)
	assert.Nil(t, conflict.Existing)
	assert.Contains(t, err.Error(), "no folder with that title could be found")
}

func TestFindOrCreateFolder_PathUnknown(t *testing.T) {
	mux := http.NewServeMux()
	space(mux)
	duplicateOnCreate(mux)
	serve(mux, "/wiki/api/v2/folders/200/direct-children", noResults)
	search(mux, `{"results":[{"content":{"id":"900","type":"folder","title":"Guides"}}]}`)
	mux.HandleFunc("GET /wiki/api/v2/folders/900", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	api := confluencetest.NewAPI(t, mux)

	_, _, err := api.FindOrCreateFolder("7", "DOC", "Guides", "200")

	var conflict *confluence.FolderTitleConflictError
	require.True(t, errors.As(err, &conflict), "want FolderTitleConflictError, got %v", err)
	require.NotNil(t, conflict.Existing)
	assert.Empty(t, conflict.Existing.Path)
	assert.Contains(t, err.Error(), `"Guides" already exists (id 900)`)
}

func TestFindOrCreateFolder_OtherErrorsPassThrough(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /wiki/api/v2/folders", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = fmt.Fprint(w, `{"errors":[{"status":403,"title":"Not permitted"}]}`)
	})
	api := confluencetest.NewAPI(t, mux)

	_, _, err := api.FindOrCreateFolder("7", "DOC", "Guides", "200")
	require.Error(t, err)

	var conflict *confluence.FolderTitleConflictError
	assert.False(t, errors.As(err, &conflict))
	assert.Contains(t, err.Error(), "403")
}

func TestFolderTitleConflictError(t *testing.T) {
	err := &confluence.FolderTitleConflictError{
		Title:      "Guides",
		ParentPath: "/Archive/",
		Existing:   &confluence.FolderLocation{ID: "900", Title: "Guides", Path: "/Guides/"},
	}
	assert.Equal(t,
		`cannot create folder "Guides" under "/Archive/": Confluence folder titles are unique `+
			`within a space and "Guides" already exists at "/Guides/" (id 900); `+
			`rename this Folder segment`,
		err.Error())
}

func TestPageInfoStatus(t *testing.T) {
	mux := http.NewServeMux()
	serve(mux, "/wiki/rest/api/content/42", `{"id":"42","type":"page","status":"archived","title":"Old"}`)
	api := confluencetest.NewAPI(t, confluencetest.ReadOnly(t, mux))

	pg, err := api.GetPageByID("42")
	require.NoError(t, err)
	assert.Equal(t, "archived", pg.Status)
}

func TestIsCloud(t *testing.T) {
	assert.True(t, confluencetest.NewAPI(t, http.NewServeMux()).IsCloud())
}
