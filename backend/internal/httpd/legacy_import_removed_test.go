package httpd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/config"
)

func TestLegacyImportRoutesAreNotExposed(t *testing.T) {
	router := NewRouterWithControl(config.Config{}, discardLogger(), nil, APIDeps{}, ControlDeps{})

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/import", nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Fatalf("%s /api/v1/import status = %d, want %d; body=%s", method, rec.Code, http.StatusNotFound, rec.Body.String())
			}
		})
	}
}
