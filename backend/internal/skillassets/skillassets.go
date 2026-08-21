// Package skillassets embeds the using-kennel skill (the ao CLI catalog) and
// installs it into the Kennel data dir at daemon boot. Worker sessions run in a
// worktree of whatever project they were spawned in, so a repo-relative
// skills/ path only resolves when that project happens to be the Kennel repo
// itself. Installing under the data dir gives every session, in any project, a
// stable absolute path to read.
//
// The embedded copy is the single source of truth. Install clobbers the
// on-disk copy on every boot, so a new daemon build always refreshes it and the
// two can never drift; there is no version marker or hash to keep in sync
// because the daemon binary already is the version.
//
// Materialize writes that same embedded tree into an arbitrary destination
// directory (used by the opencode adapter to place the skill where opencode's
// skill tool discovers it under .opencode/skills/).
package skillassets

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"embed"
)

//go:embed using-kennel
var files embed.FS

// SkillName is the installed skill's directory name under <dataDir>/skills.
const SkillName = "using-kennel"

// Dir returns the absolute directory the skill installs into for a given data
// dir. Callers building prompts use this so the path they cite always matches
// where Install writes.
func Dir(dataDir string) string {
	return filepath.Join(dataDir, "skills", SkillName)
}

// Install writes the embedded using-kennel skill into <dataDir>/skills/using-kennel,
// replacing any existing copy. It runs once at daemon boot, before any session
// spawns, so a plain clobber-and-write needs no locking: there are no
// concurrent readers yet. A failure is returned but is non-fatal to boot (the
// skill enhances `kennel --help`, it is not load-bearing).
func Install(dataDir string) error {
	return Materialize(Dir(dataDir))
}

// Materialize writes the embedded using-kennel skill into destDir (the skill root
// itself, e.g. <dataDir>/skills/using-kennel or <workspace>/.opencode/skills/using-kennel),
// replacing any existing copy. Callers that need Kennel-ownership guards must apply
// them before calling Materialize.
func Materialize(destDir string) error {
	if strings.TrimSpace(destDir) == "" {
		return fmt.Errorf("skillassets.Materialize: destDir is required")
	}
	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("clear skill dir %q: %w", destDir, err)
	}
	// embed.FS always uses forward-slash paths rooted at "using-kennel"; strip that
	// prefix and map each entry onto destDir with the platform separator.
	return fs.WalkDir(files, SkillName, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(p, SkillName)
		rel = strings.TrimPrefix(rel, "/")
		target := destDir
		if rel != "" {
			target = filepath.Join(destDir, filepath.FromSlash(rel))
		}
		if d.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		b, err := files.ReadFile(p)
		if err != nil {
			return fmt.Errorf("read embedded %q: %w", p, err)
		}
		if err := os.WriteFile(target, b, 0o600); err != nil {
			return fmt.Errorf("write %q: %w", target, err)
		}
		return nil
	})
}
