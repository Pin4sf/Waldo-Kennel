package devicebridge

// This file-backed fake exercises the SessionStore transaction boundary across
// fresh processes. It is test-only, not a database, migration or Gate B proof.
import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type fakeCommandRecord struct {
	Raw, Result []byte
	Fingerprint string
}
type fakeReceiptRecord struct {
	Result      []byte
	RetainUntil int64
}
type fakeStoreState struct {
	Commands    map[string]fakeCommandRecord
	Messages    map[string]string
	Idempotency map[string]string
	Outbox      map[string]PendingResult
	Tombstones  map[string]fakeReceiptRecord
	Seq         int64
}
type fileSessionStore struct {
	mu         sync.Mutex
	path       string
	state      fakeStoreState
	failCommit bool
}

func openFileSessionStore(path string) (*fileSessionStore, error) {
	s := &fileSessionStore{path: path}
	raw, err := os.ReadFile(path)
	if err == nil {
		err = json.Unmarshal(raw, &s.state)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if s.state.Commands == nil {
		s.state = fakeStoreState{Commands: map[string]fakeCommandRecord{}, Messages: map[string]string{}, Idempotency: map[string]string{}, Outbox: map[string]PendingResult{}, Tombstones: map[string]fakeReceiptRecord{}}
	}
	return s, nil
}
func fakeKey(scope Scope, values ...any) string {
	fields := []any{scope.DeviceID, scope.OwnerID}
	fields = append(fields, values...)
	raw, _ := json.Marshal(fields)
	return string(raw)
}
func fakeCommandKey(scope Scope, frame map[string]any) string {
	return fakeKey(scope, frame["command_id"], frame["revision"])
}
func fakeFingerprint(frame map[string]any, command bool) string {
	cp := make(map[string]any, len(frame))
	for k, v := range frame {
		if command && k == "message_id" {
			continue
		}
		if k != "timestamp" && k != "nonce" && k != "signature" {
			cp[k] = v
		}
	}
	raw, _ := marshalCanonical(cp)
	sum := sha256Bytes(raw)
	return hex.EncodeToString(sum[:])
}
func sha256Bytes(raw []byte) [32]byte { return sha256.Sum256(raw) }
func (s *fileSessionStore) transaction(fn func(*fakeStoreState) error) error {
	raw, err := json.Marshal(s.state)
	if err != nil {
		return err
	}
	var next fakeStoreState
	if err = json.Unmarshal(raw, &next); err != nil {
		return err
	}
	if err = fn(&next); err != nil {
		return err
	}
	if s.failCommit {
		return errors.New("injected fake transaction failure")
	}
	raw, err = json.Marshal(next)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), "bridge-fake-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(raw); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(name, s.path); err != nil {
		return err
	}
	s.state = next
	return nil
}
func fakeParse(scope Scope, raw []byte) (map[string]any, error) {
	f, err := ParseBackendFrame(raw)
	if err != nil {
		return nil, err
	}
	if !scope.valid() || !scope.matches(f) {
		return nil, ErrTerminalFrame
	}
	return f, nil
}
func (s *fileSessionStore) Admit(ctx context.Context, scope Scope, raw []byte, now time.Time) (Admission, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := fakeParse(scope, raw)
	if err != nil {
		return Admission{}, err
	}
	if f["type"] != TypeCommand {
		return Admission{}, ErrInvalidFrameShape
	}
	key := fakeCommandKey(scope, f)
	msg := fakeKey(scope, f["message_id"])
	idem := fakeKey(scope, f["idempotency_key"])
	fp := fakeFingerprint(f, true)
	logical := fakeFingerprint(f, false)
	if old, ok := s.state.Messages[msg]; ok && old != logical {
		return Admission{State: AckRejected, Reason: ReasonIdempotencyConflict}, nil
	}
	if old, ok := s.state.Idempotency[idem]; ok && old != key {
		return Admission{State: AckRejected, Reason: ReasonIdempotencyConflict}, nil
	}
	old, exists := s.state.Commands[key]
	if exists && old.Fingerprint != fp {
		return Admission{State: AckRejected, Reason: ReasonIdempotencyConflict}, nil
	}
	if !exists {
		count := 0
		for _, t := range s.state.Tombstones {
			result, _ := parseJSONObject(t.Result)
			if scope.matches(result) {
				count++
			}
		}
		if count >= TombstonesPerDeviceMax {
			return Admission{State: AckRejected, Reason: ReasonInvalidShape}, nil
		}
	}
	err = s.transaction(func(next *fakeStoreState) error {
		next.Messages[msg] = logical
		next.Idempotency[idem] = key
		if !exists {
			next.Commands[key] = fakeCommandRecord{Raw: raw, Fingerprint: fp}
		}
		return nil
	})
	return Admission{State: AckAccepted, New: !exists}, err
}
func (s *fileSessionStore) ResultForCommand(ctx context.Context, scope Scope, raw []byte) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := fakeParse(scope, raw)
	if err != nil {
		return nil, false, err
	}
	r, ok := s.state.Commands[fakeCommandKey(scope, f)]
	if !ok || r.Fingerprint != fakeFingerprint(f, true) {
		return nil, false, ErrInvalidFrameShape
	}
	return append([]byte(nil), r.Result...), len(r.Result) > 0, nil
}
func (s *fileSessionStore) CommitResult(ctx context.Context, scope Scope, command, result []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, err := fakeParse(scope, command)
	if err != nil {
		return err
	}
	r, err := parseJSONObject(result)
	if err != nil || ValidateResultForCommand(r, c) != nil {
		return ErrInvalidFrameShape
	}
	key := fakeCommandKey(scope, c)
	old, ok := s.state.Commands[key]
	if !ok || old.Fingerprint != fakeFingerprint(c, true) {
		return ErrInvalidFrameShape
	}
	if len(old.Result) > 0 {
		prior, _ := parseJSONObject(old.Result)
		if fakeFingerprint(prior, false) == fakeFingerprint(r, false) {
			return nil
		}
		return ErrInvalidFrameShape
	}
	for _, p := range s.state.Outbox {
		prior, _ := parseJSONObject(p.Frame)
		if prior["device_id"] == scope.DeviceID && prior["message_id"] == r["message_id"] {
			return ErrInvalidFrameShape
		}
	}
	return s.transaction(func(next *fakeStoreState) error {
		old.Result = append([]byte(nil), result...)
		next.Commands[key] = old
		next.Seq++
		next.Outbox[key] = PendingResult{next.Seq, result}
		return nil
	})
}
func (s *fileSessionStore) Pending(ctx context.Context, scope Scope) ([]PendingResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var rows []PendingResult
	for _, row := range s.state.Outbox {
		f, _ := parseJSONObject(row.Frame)
		if scope.matches(f) {
			rows = append(rows, PendingResult{row.Seq, append([]byte(nil), row.Frame...)})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Seq < rows[j].Seq })
	return rows, nil
}
func (s *fileSessionStore) Depth(ctx context.Context, scope Scope) (int64, error) {
	rows, err := s.Pending(ctx, scope)
	return int64(len(rows)), err
}
func (s *fileSessionStore) AcceptedUnresulted(ctx context.Context, scope Scope) ([][]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var rows [][]byte
	for _, r := range s.state.Commands {
		c, _ := ParseBackendFrame(r.Raw)
		if scope.matches(c) && len(r.Result) == 0 {
			rows = append(rows, append([]byte(nil), r.Raw...))
		}
	}
	sort.Slice(rows, func(i, j int) bool { return string(rows[i]) < string(rows[j]) })
	return rows, nil
}
func (s *fileSessionStore) ApplyReceipt(ctx context.Context, scope Scope, raw []byte, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	receipt, err := fakeParse(scope, raw)
	if err != nil {
		return err
	}
	if receipt["type"] != TypeReceipt {
		return ErrInvalidFrameShape
	}
	id := receipt["payload"].(map[string]any)["result_message_id"]
	tkey := fakeKey(scope, id)
	if tomb, ok := s.state.Tombstones[tkey]; ok {
		result, _ := parseJSONObject(tomb.Result)
		return ValidateReceiptForResult(receipt, result)
	}
	for key, row := range s.state.Outbox {
		result, _ := parseJSONObject(row.Frame)
		if !scope.matches(result) || result["message_id"] != id {
			continue
		}
		if err := ValidateReceiptForResult(receipt, result); err != nil {
			return err
		}
		command, _ := ParseBackendFrame(s.state.Commands[key].Raw)
		expires, _ := parseInteger(command["expires_at"])
		return s.transaction(func(next *fakeStoreState) error {
			delete(next.Outbox, key)
			next.Tombstones[tkey] = fakeReceiptRecord{Result: row.Frame, RetainUntil: expires + ReplayWindowSeconds}
			return nil
		})
	}
	return ErrUnknownReceiptMessage
}
func (s *fileSessionStore) ClearPaired(ctx context.Context, scope Scope) error { return nil }

var _ SessionStore = (*fileSessionStore)(nil)
