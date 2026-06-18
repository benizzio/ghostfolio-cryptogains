// Package fixture provides empirical fixture loading and validation helpers.
// Authored by: OpenCode
package fixture

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// collectOracleOutputPaths returns every JSON fixture path below one root in
// stable lexical order.
// Authored by: OpenCode
func collectOracleOutputPaths(rootPath string) ([]string, error) {
	var paths = make([]string, 0)

	var err = filepath.WalkDir(rootPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			return nil
		}

		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk oracle output fixtures %s: %w", rootPath, err)
	}

	sort.Strings(paths)
	if len(paths) != 0 {
		return paths, nil
	}

	return nil, fmt.Errorf("walk oracle output fixtures %s: no JSON fixtures found", rootPath)
}
