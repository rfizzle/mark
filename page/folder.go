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
		if dryRun {
			log.Infof(nil,
				"[dry-run] would ensure folder %q exists (parent=%q, space=%s)",
				segment, parentID, spaceKey)
			// Return a synthetic FolderInfo; we can't continue walking
			// because we don't have a real ID for subsequent lookups.
			return &confluence.FolderInfo{
				Title:   segment,
				SpaceID: spaceID,
			}, nil
		}

		folder, created, err := api.FindOrCreateFolder(spaceID, spaceKey, segment, parentID)
		if err != nil {
			return nil, fmt.Errorf("error ensuring folder %q: %w", segment, err)
		}

		if created {
			log.Infof(nil, "created folder %q in space %s (id=%s)", segment, spaceKey, folder.ID)
		} else {
			log.Debugf(nil, "folder %q already exists (id=%s)", segment, folder.ID)
		}

		current = folder
		parentID = folder.ID
	}

	return current, nil
}
