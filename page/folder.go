package page

import (
	"errors"
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

// formatFolderPath joins folder path segments as "/A/B/"; no segments is "/".
func formatFolderPath(segments []string) string {
	if len(segments) == 0 {
		return "/"
	}
	return "/" + strings.Join(segments, "/") + "/"
}

// ResolveFolder ensures the entire folder path exists in Confluence Cloud,
// creating intermediate folders as needed (mkdir -p semantics).
// Returns the deepest FolderInfo.
//
// In dry-run mode nothing is created: the path is walked read-only and the
// result has no ID if any folder on it would have to be created.
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

	if dryRun {
		return resolveFolderDryRun(api, spaceKey, spaceID, segments)
	}

	var parentID string
	var current *confluence.FolderInfo

	for i, segment := range segments {
		folder, created, err := api.FindOrCreateFolder(spaceID, spaceKey, segment, parentID)
		if err != nil {
			var conflict *confluence.FolderTitleConflictError
			if errors.As(err, &conflict) {
				conflict.ParentPath = formatFolderPath(segments[:i])
				return nil, conflict
			}
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

// resolveFolderDryRun walks the folder path without creating anything. It
// reports which folders exist and which would be created, and fails with a
// *confluence.FolderTitleConflictError if a folder that would be created has
// a title already used elsewhere in the space.
func resolveFolderDryRun(
	api *confluence.API,
	spaceKey string,
	spaceID string,
	segments []string,
) (*confluence.FolderInfo, error) {
	var current *confluence.FolderInfo
	// Once one folder is missing, everything below it is missing too, but
	// each title still has to be checked against the rest of the space.
	missing := false

	for i, segment := range segments {
		parentPath := formatFolderPath(segments[:i])

		if !missing {
			var parentID string
			if current != nil {
				parentID = current.ID
			}
			folder, err := api.FindFolder(spaceKey, segment, parentID)
			if err != nil {
				return nil, fmt.Errorf("error looking up folder %q under %q: %w", segment, parentPath, err)
			}
			if folder != nil {
				log.Infof(nil, "[dry-run] folder %q exists (id=%s)", formatFolderPath(segments[:i+1]), folder.ID)
				current = folder
				continue
			}
		}

		existing, err := api.FindFolderInSpace(spaceKey, segment)
		if err != nil {
			return nil, fmt.Errorf("error searching space %s for folder %q: %w", spaceKey, segment, err)
		}
		if existing != nil {
			return nil, &confluence.FolderTitleConflictError{
				Title:      segment,
				ParentPath: parentPath,
				Existing:   existing,
			}
		}

		log.Infof(nil, "[dry-run] would create folder %q under %q", segment, parentPath)
		missing = true
		current = &confluence.FolderInfo{Title: segment, SpaceID: spaceID}
	}

	return current, nil
}
