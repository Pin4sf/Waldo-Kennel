package httpd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	telemetryadapter "github.com/Pin4sf/Waldo-Kennel/backend/internal/adapters/telemetry"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/config"
)

type telemetryRoundTripper func(*http.Request) (*http.Response, error)

func (f telemetryRoundTripper) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCLIInvokedRouteEmitsTelemetryForUserCommands(t *testing.T) {
	sink := &captureSink{}
	r := NewRouterWithControl(config.Config{DataDir: t.TempDir()}, discardLogger(), nil, APIDeps{Telemetry: sink}, ControlDeps{})

	postInvoked := func(command, commandPath string) {
		t.Helper()
		body := `{"command":"` + command + `","commandPath":"` + commandPath + `"}`
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/internal/telemetry/cli-invoked", strings.NewReader(body))
		req.Host = "127.0.0.1:3001"
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want 202", rec.Code)
		}
	}

	postInvoked("spawn", "kennel spawn")
	if len(sink.events) != 2 {
		t.Fatalf("events = %d, want 2", len(sink.events))
	}
	if sink.events[0].Name != "kennel.cli.invoked" {
		t.Fatalf("event name = %q, want kennel.cli.invoked", sink.events[0].Name)
	}
	if got := sink.events[0].Payload["command_path"]; got != "kennel spawn" {
		t.Fatalf("command_path = %#v, want kennel spawn", got)
	}
	if got := sink.events[0].Payload["actor_type"]; got != "user" {
		t.Fatalf("actor_type = %#v, want user", got)
	}
	if sink.events[1].Name != "kennel.app.active" {
		t.Fatalf("second event name = %q, want kennel.app.active", sink.events[1].Name)
	}
	if got := sink.events[1].Payload["channel"]; got != "cli" {
		t.Fatalf("channel = %#v, want cli", got)
	}

	// Repeat invocations of the same command the same day are polling noise:
	// both the per-command invocation event and the daily activity heartbeat
	// stay silent.
	postInvoked("spawn", "kennel spawn")
	if len(sink.events) != 2 {
		t.Fatalf("events after repeat invocation = %d, want 2", len(sink.events))
	}

	// A different command the same day still reports its first invocation, but
	// no additional heartbeat.
	postInvoked("send", "kennel send")
	if len(sink.events) != 3 {
		t.Fatalf("events after new command = %d, want 3", len(sink.events))
	}
	if sink.events[2].Name != "kennel.cli.invoked" {
		t.Fatalf("third event name = %q, want kennel.cli.invoked", sink.events[2].Name)
	}
	if got := sink.events[2].Payload["command_path"]; got != "kennel send" {
		t.Fatalf("command_path = %#v, want kennel send", got)
	}
}

func TestCLIInvokedRouteDropsRoutineInternalSuccessTelemetry(t *testing.T) {
	sink := &captureSink{}
	r := NewRouterWithControl(config.Config{DataDir: t.TempDir()}, discardLogger(), nil, APIDeps{Telemetry: sink}, ControlDeps{})

	for _, body := range []string{
		`{"command":"status","commandPath":"kennel status","actorType":"user"}`,
		`{"command":"ls","commandPath":"kennel session ls","actorType":"user"}`,
		`{"command":"get","commandPath":"kennel session get","actorType":"user"}`,
		`{"command":"ls","commandPath":"kennel project ls","actorType":"user"}`,
		`{"command":"get","commandPath":"kennel project get","actorType":"user"}`,
		`{"command":"ls","commandPath":"kennel orchestrator ls","actorType":"user"}`,
		`{"command":"hooks","commandPath":"kennel hooks","actorType":"agent"}`,
		`{"command":"hooks","commandPath":"kennel  hooks","actorType":"user"}`,
		`{"command":"hooks","commandPath":"KENNEL HOOKS","actorType":"user"}`,
		`{"command":"hooks","commandPath":"kennel hooks claude-code post-tool-use","actorType":"user"}`,
		`{"command":"pty-host","commandPath":"kennel pty-host","actorType":"system"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/internal/telemetry/cli-invoked", strings.NewReader(body))
		req.Host = "127.0.0.1:3001"
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("status for %s = %d, want 202", body, rec.Code)
		}
	}

	if len(sink.events) != 0 {
		t.Fatalf("events = %#v, want none for routine internal successes", sink.events)
	}
}

func TestCLIInvokedRouteDropsUnknownLegacyCommandPaths(t *testing.T) {
	sink := &captureSink{}
	r := NewRouterWithControl(config.Config{DataDir: t.TempDir()}, discardLogger(), nil, APIDeps{Telemetry: sink}, ControlDeps{})

	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/internal/telemetry/cli-invoked", strings.NewReader(`{"command":"surprise","commandPath":"kennel surprise"}`))
	req.Host = "127.0.0.1:3001"
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}
	if len(sink.events) != 0 {
		t.Fatalf("events = %#v, want none for unknown legacy command", sink.events)
	}
}

func TestCLIInvokedRouteSeparatesAgentAndSystemInvocationsFromActiveUsers(t *testing.T) {
	sink := &captureSink{}
	r := NewRouterWithControl(config.Config{DataDir: t.TempDir()}, discardLogger(), nil, APIDeps{Telemetry: sink}, ControlDeps{})

	postInvoked := func(body string) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/internal/telemetry/cli-invoked", strings.NewReader(body))
		req.Host = "127.0.0.1:3001"
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want 202", rec.Code)
		}
	}

	// Older CLIs do not send actorType, so the daemon infers kennel hooks as a
	// routine internal agent path and drops successful invocation telemetry.
	postInvoked(`{"command":"hooks","commandPath":"kennel hooks"}`)
	if len(sink.events) != 0 {
		t.Fatalf("events after hooks = %d, want 0", len(sink.events))
	}

	// Newer CLIs mark any command run inside an KENNEL-managed agent session as
	// agent-context, even if it is not the hooks subcommand.
	postInvoked(`{"command":"send","commandPath":"kennel send","actorType":"agent"}`)
	if len(sink.events) != 1 {
		t.Fatalf("events after agent send = %d, want 1", len(sink.events))
	}
	if sink.events[0].Payload["actor_type"] != "agent" {
		t.Fatalf("agent send actor_type = %#v, want agent", sink.events[0].Payload["actor_type"])
	}

	// Internal runtime hosts are system background processes and should not
	// emit CLI usage or active-user telemetry at all.
	postInvoked(`{"command":"pty-host","commandPath":"kennel pty-host"}`)
	if len(sink.events) != 1 {
		t.Fatalf("events after pty-host = %d, want 1", len(sink.events))
	}
}

func TestCLIInvokedRouteDedupeIncludesActorType(t *testing.T) {
	sink := &captureSink{}
	r := NewRouterWithControl(config.Config{DataDir: t.TempDir()}, discardLogger(), nil, APIDeps{Telemetry: sink}, ControlDeps{})

	postInvoked := func(body string) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/internal/telemetry/cli-invoked", strings.NewReader(body))
		req.Host = "127.0.0.1:3001"
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want 202", rec.Code)
		}
	}

	postInvoked(`{"command":"send","commandPath":"kennel send","actorType":"agent"}`)
	postInvoked(`{"command":"send","commandPath":"kennel send","actorType":"agent"}`)
	if len(sink.events) != 1 {
		t.Fatalf("events after repeated agent command = %d, want 1", len(sink.events))
	}

	postInvoked(`{"command":"send","commandPath":"kennel send","actorType":"user"}`)
	if len(sink.events) != 3 {
		t.Fatalf("events after same user command = %d, want 3", len(sink.events))
	}
	if sink.events[1].Name != "kennel.cli.invoked" || sink.events[1].Payload["actor_type"] != "user" {
		t.Fatalf("second emitted event = %#v, want user kennel.cli.invoked", sink.events[1])
	}
	if sink.events[2].Name != "kennel.app.active" {
		t.Fatalf("third emitted event = %#v, want kennel.app.active", sink.events[2])
	}
}

func TestCLIInvokedRouteRequiresLoopback(t *testing.T) {
	sink := &captureSink{}
	r := NewRouterWithControl(config.Config{DataDir: t.TempDir()}, discardLogger(), nil, APIDeps{Telemetry: sink}, ControlDeps{})

	req := httptest.NewRequest(http.MethodPost, "http://evil.example/internal/telemetry/cli-invoked", strings.NewReader(`{"command":"status","commandPath":"kennel status"}`))
	req.Host = "evil.example"
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if len(sink.events) != 0 {
		t.Fatalf("events = %d, want 0", len(sink.events))
	}
}

func TestCLIUsageErrorRouteEmitsTelemetry(t *testing.T) {
	sink := &captureSink{}
	r := chi.NewRouter()
	mountTelemetry(r, config.Config{DataDir: t.TempDir()}, sink)

	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/internal/telemetry/cli-usage-error", strings.NewReader(`{"command":"status","commandPath":"kennel status","error":"too many args"}`))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}
	if len(sink.events) != 1 || sink.events[0].Name != "kennel.cli.usage_errors" {
		t.Fatalf("events = %#v, want one kennel.cli.usage_errors event", sink.events)
	}
	payload := sink.events[0].Payload
	if got := payload["component"]; got != "cli" {
		t.Fatalf("payload.component = %#v, want cli", got)
	}
	if got := payload["operation"]; got != "command_parse" {
		t.Fatalf("payload.operation = %#v, want command_parse", got)
	}
	if got := payload["command_path"]; got != "kennel status" {
		t.Fatalf("payload.command_path = %#v, want kennel status", got)
	}
	if got := payload["error_kind"]; got != "usage" {
		t.Fatalf("payload.error_kind = %#v, want usage", got)
	}
	if got := payload["fingerprint"]; got == "" {
		t.Fatalf("payload.fingerprint = %#v, want non-empty", got)
	}
	if _, ok := payload["error"]; ok {
		t.Fatalf("payload leaked raw error text: %#v", payload)
	}
}

func TestCLIUsageErrorRouteHashesInvalidCommandsBeforeRemoteExport(t *testing.T) {
	requests := make(chan map[string]any, 1)
	sink, err := telemetryadapter.NewPostHogSink(
		t.TempDir(),
		"phc_test",
		"https://us.i.posthog.com",
		"",
		"",
		telemetryRoundTripper(func(req *http.Request) (*http.Response, error) {
			defer req.Body.Close()
			var body map[string]any
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				return nil, err
			}
			requests <- body
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       http.NoBody,
			}, nil
		}),
		discardLogger(),
	)
	if err != nil {
		t.Fatalf("NewPostHogSink: %v", err)
	}

	r := chi.NewRouter()
	mountTelemetry(r, config.Config{DataDir: t.TempDir()}, sink)

	const rawCommand = "Review https://example.com/private; ping @security"
	req := httptest.NewRequest(
		http.MethodPost,
		"http://127.0.0.1/internal/telemetry/cli-usage-error",
		strings.NewReader(`{"command":"`+rawCommand+`","commandPath":"kennel `+rawCommand+`","error":"too many args"}`),
	)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}
	if err := sink.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}

	select {
	case request := <-requests:
		properties, ok := request["properties"].(map[string]any)
		if !ok {
			t.Fatalf("properties type = %T, want map[string]any", request["properties"])
		}
		for _, key := range []string{"command", "command_path"} {
			value, ok := properties[key].(string)
			if !ok || len(value) != len("sha256:")+16 || !strings.HasPrefix(value, "sha256:") {
				t.Fatalf("properties.%s = %#v, want sha256:<16 hex>", key, properties[key])
			}
			if strings.Contains(value, rawCommand) {
				t.Fatalf("properties.%s leaked raw command: %q", key, value)
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("PostHog sink did not send request")
	}
}

func TestCLIInvokedRoutePersistsDailyReservationsAcrossRouterRestart(t *testing.T) {
	dataDir := t.TempDir()
	sink := &captureSink{}
	cfg := config.Config{DataDir: dataDir}

	postInvoked := func(r http.Handler, command, commandPath string) {
		t.Helper()
		body := `{"command":"` + command + `","commandPath":"` + commandPath + `"}`
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/internal/telemetry/cli-invoked", strings.NewReader(body))
		req.Host = "127.0.0.1:3001"
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want 202", rec.Code)
		}
	}

	r1 := NewRouterWithControl(cfg, discardLogger(), nil, APIDeps{Telemetry: sink}, ControlDeps{})
	postInvoked(r1, "spawn", "kennel spawn")
	if len(sink.events) != 2 {
		t.Fatalf("events after first invocation = %d, want 2", len(sink.events))
	}

	r2 := NewRouterWithControl(cfg, discardLogger(), nil, APIDeps{Telemetry: sink}, ControlDeps{})
	postInvoked(r2, "spawn", "kennel spawn")
	if len(sink.events) != 2 {
		t.Fatalf("events after router restart repeat = %d, want 2", len(sink.events))
	}

	postInvoked(r2, "send", "kennel send")
	if len(sink.events) != 3 {
		t.Fatalf("events after router restart new command = %d, want 3", len(sink.events))
	}
	if sink.events[2].Name != "kennel.cli.invoked" {
		t.Fatalf("third event name = %q, want kennel.cli.invoked", sink.events[2].Name)
	}
}

func TestRecoverTelemetryEmitsPanicEvent(t *testing.T) {
	sink := &captureSink{}
	r := NewRouterWithControl(config.Config{}, discardLogger(), nil, APIDeps{Telemetry: sink}, ControlDeps{})
	r.Get("/panic", func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/panic", nil)
	req.Host = "127.0.0.1:3001"
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var panicPayload, fiveXXPayload map[string]any
	for _, ev := range sink.events {
		switch ev.Name {
		case "kennel.daemon.panic":
			panicPayload = ev.Payload
		case "kennel.http.5xx":
			fiveXXPayload = ev.Payload
		}
	}
	if panicPayload == nil {
		t.Fatalf("events = %#v, want kennel.daemon.panic", sink.events)
	}
	if fiveXXPayload == nil {
		t.Fatalf("events = %#v, want kennel.http.5xx after recovery", sink.events)
	}
	if got := panicPayload["component"]; got != "httpd" {
		t.Fatalf("panic payload.component = %#v, want httpd", got)
	}
	if got := panicPayload["operation"]; got != "http_request_panic" {
		t.Fatalf("panic payload.operation = %#v, want http_request_panic", got)
	}
	if got := panicPayload["path"]; got != "/panic" {
		t.Fatalf("panic payload.path = %#v, want /panic", got)
	}
	if got := panicPayload["panic_kind"]; got != "string" {
		t.Fatalf("panic payload.panic_kind = %#v, want string", got)
	}
	if got := panicPayload["fingerprint"]; got == "" {
		t.Fatalf("panic payload.fingerprint = %#v, want non-empty", got)
	}
	if got := panicPayload["stack_fingerprint"]; got == "" {
		t.Fatalf("panic payload.stack_fingerprint = %#v, want non-empty", got)
	}
	if got := fiveXXPayload["path"]; got != "/panic" {
		t.Fatalf("5xx payload.path = %#v, want /panic", got)
	}
	if got := fiveXXPayload["status_family"]; got != "5xx" {
		t.Fatalf("5xx payload.status_family = %#v, want 5xx", got)
	}
}
