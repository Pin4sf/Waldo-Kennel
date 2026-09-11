package artifactstore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

func TestExportStagesAtomicallyAndRefusesDestinationSymlink(t *testing.T) {
	store, err := New(Config{Root: filepath.Join(t.TempDir(), "artifacts")})
	if err != nil {
		t.Fatal(err)
	}
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "run.sh"), []byte("#!/bin/sh\necho ok\n"), 0o750); err != nil {
		t.Fatal(err)
	}
	result, err := store.Retain(context.Background(), Input{
		AttemptID: "delivery-atomic", OutcomeID: "outcome-1", PlanRevisionID: "plan-1", WorkUnitID: "unit-1",
		ContractRevisionNumber: 1, WorkspaceKind: domain.WorkspaceStagedFolder, WorkspacePath: workspace,
	})
	if err != nil {
		t.Fatal(err)
	}

	destination := filepath.Join(t.TempDir(), "bundle")
	if _, err := store.Export(context.Background(), ExportRequest{Receipt: result.Receipt, Draft: true, Destination: destination}); err != nil {
		t.Fatalf("atomic export: %v", err)
	}
	info, err := os.Stat(filepath.Join(destination, "run.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o750 {
		t.Fatalf("mode = %o, want 750", info.Mode().Perm())
	}

	symlink := filepath.Join(t.TempDir(), "symlink")
	if err := os.Symlink(destination, symlink); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Export(context.Background(), ExportRequest{Receipt: result.Receipt, Draft: true, Destination: symlink}); !errors.Is(err, ErrExportDestinationUnsafe) {
		t.Fatalf("symlink destination = %v, want ErrExportDestinationUnsafe", err)
	}
}
