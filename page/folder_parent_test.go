package page

import (
	"testing"

	"github.com/kovetskiy/mark/v16/confluence"
	"github.com/kovetskiy/mark/v16/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveFolderParent_NilMeta(t *testing.T) {
	_, err := ResolveFolderParent(false, nil, nil)
	require.Error(t, err)
}

func TestResolveFolderParent_NoFolderIsNoop(t *testing.T) {
	// No Folder header: must return nil without touching the API. Passing a
	// nil API proves no call is attempted.
	parent, err := ResolveFolderParent(false, nil, &metadata.Meta{Parents: []string{"A", "B"}})
	require.NoError(t, err)
	assert.Nil(t, parent)
}

func TestResolveFolderParent_RequiresCloud(t *testing.T) {
	api := confluence.NewAPI("http://confluence.example.internal", "", "", false)

	_, err := ResolveFolderParent(false, api, &metadata.Meta{Space: "DOC", Folder: "docs"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires Confluence Cloud")
}
