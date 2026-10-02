package bridgeruntime

import (
	"context"
	"encoding/json"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/devicebridge"
	"io"
	"net/http"
	"testing"
	"time"
)

type commandSocket struct {
	commands [][]byte
	index    int
	writes   [][]byte
}

func (s *commandSocket) Read(context.Context) ([]byte, error) {
	if s.index == len(s.commands) {
		return nil, io.EOF
	}
	raw := s.commands[s.index]
	s.index++
	return raw, nil
}
func (s *commandSocket) Write(_ context.Context, raw []byte) error {
	s.writes = append(s.writes, append([]byte(nil), raw...))
	return nil
}
func (s *commandSocket) Close() error { return nil }
func TestHandlersShipUnknownAndDeliveryUnknown(t *testing.T) {
	repo, store, d, keys, dir := realIdentity(t)
	socket := &commandSocket{}
	for i, class := range []string{"machine_state_query", "notify_local"} {
		payload := map[string]any{"query_id": "q", "query_kind": "session_status"}
		id := "query"
		message := "01ARZ3NDEKTSV4RRFFQ69G5FAV"
		if i == 1 {
			payload = map[string]any{"notification_id": "n", "title": "Title", "body": "Body", "severity": "info"}
			id = "notification"
			message = "01ARZ3NDEKTSV4RRFFQ69G5FAW"
		}
		raw, e := json.Marshal(map[string]any{"contract_version": devicebridge.ContractVersion, "type": "command", "message_id": message, "device_id": string(d.DeviceID), "owner_id": d.OwnerID, "command_id": id, "idempotency_key": id, "revision": 1, "class": class, "expires_at": time.Now().Unix() + 600, "payload": payload})
		if e != nil {
			t.Fatal(e)
		}
		socket.commands = append(socket.commands, raw)
	}
	f := NewFactory(Dependencies{Fence: repo, Keys: keys, SessionStore: store, KeyDirectory: dir, ReadyProbe: func(context.Context) error { return nil }, Dialer: dialFunc(func(context.Context, string, http.Header) (devicebridge.Socket, error) { return socket, nil })})
	raw, e := f.New(context.Background(), "https://example.test", d, nil)
	if e != nil {
		t.Fatal(e)
	}
	s := raw.(*Session)
	if e = s.client.Connect(context.Background()); e != io.EOF {
		t.Fatal(e)
	}
	var unknown, delivery bool
	for _, raw := range socket.writes {
		var frame map[string]any
		if e = json.Unmarshal(raw, &frame); e != nil {
			t.Fatal(e)
		}
		if frame["type"] != "result" {
			continue
		}
		p := frame["payload"].(map[string]any)
		if frame["command_id"] == "query" {
			unknown = p["status"] == "answered" && p["answer"].(map[string]any)["state"] == "unknown"
		}
		if frame["command_id"] == "notification" {
			delivery = p["status"] == "failed" && p["reason"] == "delivery_unknown"
		}
	}
	if !unknown || !delivery {
		t.Fatal("dishonest adapters")
	}
	handlers := s.client.Executor.(devicebridge.Handlers)
	if handlers.Display != nil {
		t.Fatal("notification adapter invented")
	}
}
