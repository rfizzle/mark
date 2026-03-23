package confluence

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// FolderInfo represents a Confluence Cloud folder (v2 API).
type FolderInfo struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	SpaceID  string `json:"spaceId"`
	ParentID string `json:"parentId,omitempty"`
}

// folderChild is a child entry returned by the v2 direct-children endpoint.
type folderChild struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

// GetSpaceByKey resolves a space key to its SpaceInfo (including numeric ID).
func (api *API) GetSpaceByKey(spaceKey string) (*SpaceInfo, error) {
	request, err := api.rest.Res("space/"+spaceKey, &SpaceInfo{}).Get()
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

// CreateFolder creates a new folder in the given space via POST /api/v2/folders.
// If parentID is non-empty, the folder is created as a child of that parent.
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

// FindOrCreateFolder tries to create a folder, and if it already exists,
// looks it up via the v2 direct-children endpoint.
// spaceKey is needed to resolve the homepage for root-level folder lookups.
// Returns the folder (existing or newly created) and whether it was created.
func (api *API) FindOrCreateFolder(spaceID, spaceKey, title, parentID string) (*FolderInfo, bool, error) {
	// Try creating first — this is the fast path and also handles
	// the common case where the folder doesn't exist yet.
	folder, err := api.CreateFolder(spaceID, title, parentID)
	if err == nil {
		return folder, true, nil // created=true
	}

	// Check if the error is "already exists" (400 with specific message).
	if !isDuplicateFolderError(err) {
		return nil, false, err
	}

	// Folder already exists. Find it via the direct-children endpoint.
	if parentID != "" {
		folder, err := api.findFolderInChildren(parentID, title)
		if err != nil {
			return nil, false, fmt.Errorf("folder %q exists but unable to find it: %w", title, err)
		}
		if folder != nil {
			return folder, false, nil
		}
	} else {
		// Root-level folder: list children of the space homepage.
		folder, err := api.findRootFolder(spaceKey, title)
		if err != nil {
			return nil, false, fmt.Errorf("folder %q exists but unable to find it at space root: %w", title, err)
		}
		if folder != nil {
			return folder, false, nil
		}
	}

	return nil, false, fmt.Errorf("folder %q already exists in the space but could not be located via API", title)
}

// findFolderInChildren scans a parent folder's direct children for a folder
// with the given title. Uses GET /api/v2/folders/{id}/direct-children with
// pagination.
func (api *API) findFolderInChildren(parentID, title string) (*FolderInfo, error) {
	var cursor string
	for {
		result := struct {
			Results []folderChild `json:"results"`
			Links   struct {
				Next string `json:"next"`
			} `json:"_links"`
		}{}

		params := map[string]string{"limit": "250"}
		if cursor != "" {
			params["cursor"] = cursor
		}

		request, err := api.restV2.Res(
			fmt.Sprintf("folders/%s/direct-children", parentID), &result,
		).Get(params)
		if err != nil {
			return nil, err
		}
		if request.Raw.StatusCode != http.StatusOK {
			return nil, newErrorStatusNotOK(request)
		}

		for _, child := range result.Results {
			if child.Type == "folder" && child.Title == title {
				return &FolderInfo{
					ID:       child.ID,
					Title:    child.Title,
					ParentID: parentID,
				}, nil
			}
		}

		if result.Links.Next == "" {
			break
		}
		cursor = extractCursor(result.Links.Next)
		if cursor == "" {
			break
		}
	}
	return nil, nil
}

// findRootFolder finds a root-level folder in a space by listing the direct
// children of the space homepage via GET /api/v2/pages/{homepage-id}/direct-children.
// Root-level folders are children of the homepage in Confluence Cloud.
func (api *API) findRootFolder(spaceKey, title string) (*FolderInfo, error) {
	homepage, err := api.FindHomePage(spaceKey)
	if err != nil {
		return nil, fmt.Errorf("unable to get homepage for space %q: %w", spaceKey, err)
	}

	return api.findFolderInPageChildren(homepage.ID, title)
}

// findFolderInPageChildren lists direct children of a PAGE via
// GET /api/v2/pages/{id}/direct-children and returns the first folder
// matching the given title.
func (api *API) findFolderInPageChildren(pageID, title string) (*FolderInfo, error) {
	var cursor string
	for {
		result := struct {
			Results []folderChild `json:"results"`
			Links   struct {
				Next string `json:"next"`
			} `json:"_links"`
		}{}

		params := map[string]string{"limit": "250"}
		if cursor != "" {
			params["cursor"] = cursor
		}

		request, err := api.restV2.Res(
			fmt.Sprintf("pages/%s/direct-children", pageID), &result,
		).Get(params)
		if err != nil {
			return nil, err
		}
		if request.Raw.StatusCode != http.StatusOK {
			return nil, newErrorStatusNotOK(request)
		}

		for _, child := range result.Results {
			if child.Type == "folder" && child.Title == title {
				return &FolderInfo{
					ID:    child.ID,
					Title: child.Title,
				}, nil
			}
		}

		if result.Links.Next == "" {
			break
		}
		cursor = extractCursor(result.Links.Next)
		if cursor == "" {
			break
		}
	}
	return nil, nil
}

// isDuplicateFolderError checks if an error from CreateFolder indicates the
// folder already exists.
func isDuplicateFolderError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "exists with the same title")
}

// extractCursor extracts the cursor parameter from a next-page URL.
func extractCursor(nextURL string) string {
	if i := strings.Index(nextURL, "cursor="); i >= 0 {
		c := nextURL[i+len("cursor="):]
		if j := strings.IndexByte(c, '&'); j >= 0 {
			c = c[:j]
		}
		return c
	}
	return ""
}

// GetFolderByID retrieves a folder by its ID via GET /api/v2/folders/{id}.
func (api *API) GetFolderByID(folderID string) (*FolderInfo, error) {
	request, err := api.restV2.Res("folders/"+folderID, &FolderInfo{}).Get()
	if err != nil {
		return nil, err
	}
	if request.Raw.StatusCode != http.StatusOK {
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

// MovePageV2 moves an existing page under a new parent using the v2 API.
// PUT /api/v2/pages/{id} with the new parentId, title, and version.
func (api *API) MovePageV2(page *PageInfo, newParentID string) error {
	if !api.IsCloud() {
		return errors.New("v2 page move is only supported on Confluence Cloud")
	}

	payload := map[string]interface{}{
		"id":       page.ID,
		"status":   "current",
		"title":    page.Title,
		"parentId": newParentID,
		"version": map[string]interface{}{
			"number":  page.Version.Number + 1,
			"message": "moved to folder",
		},
	}

	request, err := api.restV2.Res("pages/"+page.ID, &map[string]interface{}{}).Put(payload)
	if err != nil {
		return err
	}

	if request.Raw.StatusCode != http.StatusOK {
		defer func() { _ = request.Raw.Body.Close() }()
		output, _ := io.ReadAll(request.Raw.Body)
		return fmt.Errorf("unable to move page %s: status %s, output: %s",
			page.ID, request.Raw.Status, output)
	}

	return nil
}
