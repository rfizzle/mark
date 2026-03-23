package page

import (
	"fmt"
	"strings"

	"github.com/kovetskiy/mark/v16/confluence"
	"github.com/reconquest/pkg/log"
)

// ParseFolderPath splits a slash-separated folder path into segments,
// stripping leading/trailing slashes and rejecting empty segments.
func ParseFolderPath(path string) ([]string, error) {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil, fmt.Errorf("folder path is empty")
	}

	segments := strings.Split(path, "/")
	var result []string
	for _, s := range segments {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, fmt.Errorf("folder path contains an empty segment")
		}
		result = append(result, s)
	}
	return result, nil
}

// ResolveFolder ensures the entire folder path exists in Confluence Cloud,
// creating intermediate folders as needed (mkdir -p semantics).
// Returns the deepest FolderInfo.
func ResolveFolder(
	dryRun bool,
	api *confluence.API,
	spaceKey string,
	folderPath string,
) (*confluence.FolderInfo, error) {
	if !api.IsCloud() {
		return nil, fmt.Errorf(
			"the --folder / <!-- Folder: --> option requires Confluence Cloud; "+
				"the connected instance %q does not appear to be a Cloud site",
			api.BaseURL,
		)
	}

	segments, err := ParseFolderPath(folderPath)
	if err != nil {
		return nil, err
	}

	spaceID, err := api.GetSpaceIDByKey(spaceKey)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve space key %q to ID: %w", spaceKey, err)
	}

	var parentID string
	var current *confluence.FolderInfo

	for _, segment := range segments {
		folder, err := api.FindFolderByTitle(spaceID, segment, parentID)
		if err != nil {
			return nil, fmt.Errorf("error searching for folder %q: %w", segment, err)
		}

		if folder != nil {
			log.Debugf(nil, "folder %q already exists (id=%s)", segment, folder.ID)
			current = folder
			parentID = folder.ID
			continue
		}

		if dryRun {
			log.Infof(nil,
				"[dry-run] would create folder %q (parent=%q, space=%s)",
				segment, parentID, spaceID)
			// Return a synthetic FolderInfo; we can't continue walking
			// because we don't have a real ID for subsequent lookups.
			return &confluence.FolderInfo{
				Title:   segment,
				SpaceID: spaceID,
			}, nil
		}

		log.Infof(nil, "creating folder %q in space %s", segment, spaceID)
		folder, err = api.CreateFolder(spaceID, segment, parentID)
		if err != nil {
			return nil, fmt.Errorf("unable to create folder %q: %w", segment, err)
		}

		log.Debugf(nil, "created folder %q (id=%s)", segment, folder.ID)
		current = folder
		parentID = folder.ID
	}

	return current, nil
}
