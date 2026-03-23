package confluence

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

// FolderInfo represents a Confluence Cloud folder (v2 API).
type FolderInfo struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	SpaceID  string `json:"spaceId"`
	ParentID string `json:"parentId,omitempty"`
}

// GetSpaceByKey resolves a space key to its SpaceInfo (including numeric ID).
func (api *API) GetSpaceByKey(spaceKey string) (*SpaceInfo, error) {
	request, err := api.rest.Res("space/"+spaceKey, &SpaceInfo{}).Get(nil)
	if err != nil {
		return nil, err
	}
	if request.Raw.StatusCode != http.StatusOK {
		return nil, newErrorStatusNotOK(request)
	}
	return request.Response.(*SpaceInfo), nil
}

// GetSpaceIDByKey resolves a space key to its string space ID for use with the v2 API.
func (api *API) GetSpaceIDByKey(spaceKey string) (string, error) {
	info, err := api.GetSpaceByKey(spaceKey)
	if err != nil {
		return "", err
	}
	return strconv.Itoa(info.ID), nil
}

// FindFolderByTitle searches for a folder by title within a space.
// If parentID is non-empty, it filters to folders under that parent.
// Returns nil, nil if no matching folder is found.
func (api *API) FindFolderByTitle(spaceID, title, parentID string) (*FolderInfo, error) {
	if !api.IsCloud() {
		return nil, errors.New("folders are only supported on Confluence Cloud")
	}

	result := struct {
		Results []FolderInfo `json:"results"`
	}{}

	params := map[string]string{
		"space-id": spaceID,
		"title":    title,
	}
	if parentID != "" {
		params["parent-id"] = parentID
	}

	request, err := api.restV2.Res("folders", &result).Get(params)
	if err != nil {
		return nil, err
	}

	if request.Raw.StatusCode == http.StatusNotFound {
		_ = request.Raw.Body.Close()
		return nil, nil
	}
	if request.Raw.StatusCode != http.StatusOK {
		return nil, newErrorStatusNotOK(request)
	}

	if len(result.Results) == 0 {
		return nil, nil
	}

	// If parentID was specified, filter client-side to ensure the match is exact.
	if parentID != "" {
		for _, f := range result.Results {
			if f.ParentID == parentID {
				return &f, nil
			}
		}
		return nil, nil
	}

	return &result.Results[0], nil
}

// CreateFolder creates a new folder in the given space.
// If parentID is non-empty, the folder is created as a child of that parent folder.
func (api *API) CreateFolder(spaceID, title, parentID string) (*FolderInfo, error) {
	if !api.IsCloud() {
		return nil, errors.New("folders are only supported on Confluence Cloud")
	}

	payload := map[string]interface{}{
		"spaceId": spaceID,
		"title":   title,
	}
	if parentID != "" {
		payload["parentId"] = parentID
	}

	request, err := api.restV2.Res("folders", &FolderInfo{}).Post(payload)
	if err != nil {
		return nil, err
	}

	if request.Raw.StatusCode != http.StatusOK && request.Raw.StatusCode != http.StatusCreated {
		return nil, newErrorStatusNotOK(request)
	}

	return request.Response.(*FolderInfo), nil
}

// CreatePageV2 creates a page using the Confluence Cloud v2 API.
// This is needed when placing a page under a folder, since the v1 ancestors
// array may not accept folder IDs.
func (api *API) CreatePageV2(spaceID, parentID, title, body string) (*PageInfo, error) {
	if !api.IsCloud() {
		return nil, errors.New("v2 page creation is only supported on Confluence Cloud")
	}

	payload := map[string]interface{}{
		"spaceId":  spaceID,
		"parentId": parentID,
		"status":   "current",
		"title":    title,
		"body": map[string]interface{}{
			"representation": "storage",
			"value":          body,
		},
	}

	result := &struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Version struct {
			Number  int64  `json:"number"`
			Message string `json:"message"`
		} `json:"version"`
	}{}

	request, err := api.restV2.Res("pages", result).Post(payload)
	if err != nil {
		return nil, err
	}

	if request.Raw.StatusCode != http.StatusOK && request.Raw.StatusCode != http.StatusCreated {
		return nil, newErrorStatusNotOK(request)
	}

	// Convert v2 response to PageInfo for compatibility with the rest of the codebase.
	page := &PageInfo{
		ID:    result.ID,
		Title: result.Title,
		Type:  "page",
	}
	page.Version.Number = result.Version.Number

	// Re-fetch via v1 API to get full PageInfo including ancestors and links.
	full, err := api.GetPageByID(page.ID)
	if err != nil {
		return nil, fmt.Errorf("page created but unable to re-fetch via v1 API: %w", err)
	}
	if full != nil {
		return full, nil
	}

	return page, nil
}
