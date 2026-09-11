package artifactstore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RemoveOutcomeContent removes only daemon-owned retained snapshots. Source
// repositories and exported destinations are never inputs to this operation.
func (s *Store) RemoveOutcomeContent(ctx context.Context, attemptIDs, contextIDs []string) error {
	valid := func(id string) bool { return id != "" && id != "." && id != ".." && !strings.ContainsAny(id, "/\\\\") }
	for _, ids := range [][]string{attemptIDs, contextIDs} {
		for _, id := range ids {
			if !valid(id) {
				return fmt.Errorf("invalid retained content identity")
			}
		}
	}
	roots := []struct {
		base string
		ids  []string
	}{{s.root, attemptIDs}, {filepath.Join(s.root, documentRoot), contextIDs}}
	for _, group := range roots {
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := os.Lstat(group.base)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("retained content root is not an owned directory")
		}
		for _, id := range group.ids {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := os.RemoveAll(filepath.Join(group.base, id)); err != nil {
				return fmt.Errorf("remove retained content %s: %w", id, err)
			}
		}
	}
	return nil
}
