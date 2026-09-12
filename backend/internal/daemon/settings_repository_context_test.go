package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/config"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/httpd"
	settingssvc "github.com/Pin4sf/Waldo-Kennel/backend/internal/service/settings"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/storage/sqlite"
)

type repositoryContextResponse struct {
	MaxFiles            *int64 `json:"maxFiles"`
	MaxBytes            *int64 `json:"maxBytes"`
	MaxVisited          *int64 `json:"maxVisited"`
	EffectiveMaxFiles   int64  `json:"effectiveMaxFiles"`
	EffectiveMaxBytes   int64  `json:"effectiveMaxBytes"`
	EffectiveMaxVisited int64  `json:"effectiveMaxVisited"`
}

func patchRepositoryContext(t *testing.T, handler http.Handler, body string) (int, repositoryContextResponse) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/settings/repository-context", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var decoded repositoryContextResponse
	if rec.Code == http.StatusOK {
		if err := json.NewDecoder(rec.Body).Decode(&decoded); err != nil {
			t.Fatalf("decode repository-context response: %v", err)
		}
	}
	return rec.Code, decoded
}

func TestRepositoryContextPatchHTTPPreservesClearsUncapsAndRejectsWithoutMutation(t *testing.T) {
	db, err := sqlite.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	svc := settingssvc.New(settingsStore{store: db}, nil, nil).WithReasoningEnvLookup(func(string) string { return "" })
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := httpd.NewRouterWithControl(config.Config{}, log, nil, httpd.APIDeps{Settings: svc}, httpd.ControlDeps{})

	status, got := patchRepositoryContext(t, router, `{"maxFiles":12,"maxBytes":2048,"maxVisited":300}`)
	if status != http.StatusOK || got.MaxFiles == nil || *got.MaxFiles != 12 || got.MaxBytes == nil || *got.MaxBytes != 2048 || got.MaxVisited == nil || *got.MaxVisited != 300 {
		t.Fatalf("initial full patch: status=%d response=%+v", status, got)
	}
	status, got = patchRepositoryContext(t, router, `{"maxFiles":64}`)
	if status != http.StatusOK || got.MaxFiles == nil || *got.MaxFiles != 64 || got.MaxBytes == nil || *got.MaxBytes != 2048 || got.MaxVisited == nil || *got.MaxVisited != 300 {
		t.Fatalf("omitted fields were not preserved: status=%d response=%+v", status, got)
	}
	status, got = patchRepositoryContext(t, router, `{"maxBytes":null,"maxVisited":0}`)
	if status != http.StatusOK || got.MaxBytes != nil || got.EffectiveMaxBytes != settingssvc.DefaultRepositoryContextMaxBytes || got.MaxVisited == nil || *got.MaxVisited != 0 || got.EffectiveMaxVisited != 0 {
		t.Fatalf("null/default or zero/uncapped semantics lost: status=%d response=%+v", status, got)
	}
	status, empty := patchRepositoryContext(t, router, `{}`)
	if status != http.StatusOK || empty.MaxFiles == nil || *empty.MaxFiles != 64 || empty.MaxBytes != nil || empty.MaxVisited == nil || *empty.MaxVisited != 0 {
		t.Fatalf("empty object was not a no-op: status=%d response=%+v", status, empty)
	}

	for name, body := range map[string]string{
		"negative":       `{"maxFiles":-1}`,
		"wrong type":     `{"maxFiles":"many"}`,
		"top-level null": `null`,
		"empty body":     ``,
		"unknown field":  `{"maxFilez":12}`,
	} {
		t.Run(name, func(t *testing.T) {
			status, _ := patchRepositoryContext(t, router, body)
			if status != http.StatusBadRequest {
				t.Fatalf("status=%d, want 400", status)
			}
			stored, err := db.GetAppSettings(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if stored.RepositoryContextMaxFiles == nil || *stored.RepositoryContextMaxFiles != 64 || stored.RepositoryContextMaxBytes != nil || stored.RepositoryContextMaxVisited == nil || *stored.RepositoryContextMaxVisited != 0 {
				t.Fatalf("rejected patch mutated persisted settings: %+v", stored)
			}
		})
	}
}

func TestRepositoryContextPatchSerializesConcurrentFieldUpdates(t *testing.T) {
	db, err := sqlite.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	svc := settingssvc.New(settingsStore{store: db}, nil, nil)
	files, byteLimit := int64(77), int64(8192)
	patches := []settingssvc.RepositoryContextLimitsPatch{
		{MaxFiles: settingssvc.RepositoryContextLimitPatch{Present: true, Value: &files}},
		{MaxBytes: settingssvc.RepositoryContextLimitPatch{Present: true, Value: &byteLimit}},
	}
	var wg sync.WaitGroup
	errs := make(chan error, len(patches))
	for _, patch := range patches {
		wg.Add(1)
		go func(patch settingssvc.RepositoryContextLimitsPatch) {
			defer wg.Done()
			_, err := svc.PatchRepositoryContextLimits(context.Background(), patch)
			errs <- err
		}(patch)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent patch: %v", err)
		}
	}
	stored, err := db.GetAppSettings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stored.RepositoryContextMaxFiles == nil || *stored.RepositoryContextMaxFiles != files || stored.RepositoryContextMaxBytes == nil || *stored.RepositoryContextMaxBytes != byteLimit {
		t.Fatalf("concurrent field updates lost: %+v", stored)
	}
}
