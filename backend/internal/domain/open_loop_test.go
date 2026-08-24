package domain

import (
	"strings"
	"testing"
	"time"
)

func validConfirmOpenLoopInput() ConfirmOpenLoopInput {
	return ConfirmOpenLoopInput{
		ID:               OpenLoopID("loop-1"),
		SpaceID:          ResponsibilitySpaceID("home-1"),
		Owner:            "owner",
		Trigger:          "Passport expires within six months",
		RecheckCondition: "Recheck after the renewal appointment",
		SourceRef:        "quick-capture:cap-1",
		Confirmation:     OpenLoopConfirmationExplicit,
		ConfirmedAt:      time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC),
	}
}

func TestOpenLoopCandidateRequiresExplicitConfirmation(t *testing.T) {
	candidate := NewOpenLoopCandidate("Renew passport")
	if candidate.IsCanonical() {
		t.Fatal("candidate became canonical responsibility")
	}

	input := validConfirmOpenLoopInput()
	input.Confirmation = ""
	if _, err := candidate.Confirm(input); err == nil || !strings.Contains(err.Error(), "confirmation is required") {
		t.Fatalf("Confirm() = %v, want explicit-confirmation error", err)
	}

	input.Confirmation = OpenLoopConfirmationExplicit
	loop, err := candidate.Confirm(input)
	if err != nil {
		t.Fatalf("Confirm() = %v, want nil", err)
	}
	if !loop.IsCanonical() {
		t.Fatal("explicitly confirmed Open Loop must be canonical")
	}
	if loop.Meaning != "Renew passport" {
		t.Fatalf("Meaning = %q, want candidate meaning", loop.Meaning)
	}
}

func TestOpenLoopCandidateAcceptsUnambiguousDirectCommand(t *testing.T) {
	candidate := NewOpenLoopCandidate("Remember to renew the passport")
	input := validConfirmOpenLoopInput()
	input.Confirmation = OpenLoopConfirmationDirectCommand

	if _, err := candidate.Confirm(input); err != nil {
		t.Fatalf("Confirm() = %v, want nil for direct command", err)
	}
}

func TestOpenLoopValidation(t *testing.T) {
	candidate := NewOpenLoopCandidate("Renew passport")
	valid, err := candidate.Confirm(validConfirmOpenLoopInput())
	if err != nil {
		t.Fatalf("build valid Open Loop: %v", err)
	}

	tests := []struct {
		name    string
		mutate  func(*OpenLoop)
		wantErr string
	}{
		{name: "missing id", mutate: func(loop *OpenLoop) { loop.ID = "" }, wantErr: "open loop id is required"},
		{name: "missing space", mutate: func(loop *OpenLoop) { loop.SpaceID = "" }, wantErr: "responsibility space id is required"},
		{name: "blank meaning", mutate: func(loop *OpenLoop) { loop.Meaning = "  " }, wantErr: "meaning is required"},
		{name: "blank owner", mutate: func(loop *OpenLoop) { loop.Owner = "" }, wantErr: "owner is required"},
		{name: "blank trigger", mutate: func(loop *OpenLoop) { loop.Trigger = "" }, wantErr: "trigger is required"},
		{name: "blank recheck", mutate: func(loop *OpenLoop) { loop.RecheckCondition = "" }, wantErr: "recheck condition is required"},
		{name: "blank source", mutate: func(loop *OpenLoop) { loop.SourceRef = "" }, wantErr: "source reference is required"},
		{name: "missing confirmation", mutate: func(loop *OpenLoop) { loop.Confirmation = "" }, wantErr: "unsupported open loop confirmation"},
		{name: "missing confirmation time", mutate: func(loop *OpenLoop) { loop.ConfirmedAt = time.Time{} }, wantErr: "confirmation time is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loop := valid
			tt.mutate(&loop)
			err := loop.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoopDispositionKindSpecificFields(t *testing.T) {
	base := LoopDisposition{
		ID:         LoopDispositionID("disp-1"),
		OpenLoopID: OpenLoopID("loop-1"),
		Actor:      LoopDispositionActorOwner,
		Reason:     "Owner decided",
		OccurredAt: time.Date(2026, 8, 24, 11, 0, 0, 0, time.UTC),
	}

	transfer := base
	transfer.Kind = LoopDispositionTransfer
	if err := transfer.Validate(); err == nil || !strings.Contains(err.Error(), "target space") {
		t.Fatalf("transfer Validate() = %v, want target-space error", err)
	}
	transfer.TargetSpaceID = ResponsibilitySpaceID("home-2")
	transfer.TargetOwner = "new-owner"
	if err := transfer.Validate(); err != nil {
		t.Fatalf("complete transfer Validate() = %v, want nil", err)
	}

	supersede := base
	supersede.Kind = LoopDispositionSupersede
	if err := supersede.Validate(); err == nil || !strings.Contains(err.Error(), "successor reference") {
		t.Fatalf("supersede Validate() = %v, want successor-reference error", err)
	}
	supersede.SuccessorRef = "open-loop:loop-2"
	if err := supersede.Validate(); err != nil {
		t.Fatalf("complete supersede Validate() = %v, want nil", err)
	}
}

func TestLoopDispositionRejectsProviderCompletion(t *testing.T) {
	disposition := LoopDisposition{
		ID:         LoopDispositionID("disp-1"),
		OpenLoopID: OpenLoopID("loop-1"),
		Kind:       LoopDispositionClose,
		Actor:      LoopDispositionActorProvider,
		Reason:     "provider session completed",
		OccurredAt: time.Date(2026, 8, 24, 11, 0, 0, 0, time.UTC),
	}

	err := disposition.Validate()
	if err == nil || !strings.Contains(err.Error(), "owner decision") {
		t.Fatalf("Validate() = %v, want owner-decision error", err)
	}
}

func TestLoopDispositionValidation(t *testing.T) {
	valid := LoopDisposition{
		ID:         LoopDispositionID("disp-1"),
		OpenLoopID: OpenLoopID("loop-1"),
		Kind:       LoopDispositionRelease,
		Actor:      LoopDispositionActorOwner,
		Reason:     "No longer mine to carry",
		OccurredAt: time.Date(2026, 8, 24, 11, 0, 0, 0, time.UTC),
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}

	tests := []struct {
		name    string
		mutate  func(*LoopDisposition)
		wantErr string
	}{
		{name: "missing id", mutate: func(d *LoopDisposition) { d.ID = "" }, wantErr: "disposition id is required"},
		{name: "missing loop", mutate: func(d *LoopDisposition) { d.OpenLoopID = "" }, wantErr: "open loop id is required"},
		{name: "unknown kind", mutate: func(d *LoopDisposition) { d.Kind = "done" }, wantErr: "unsupported loop disposition"},
		{name: "missing actor", mutate: func(d *LoopDisposition) { d.Actor = "" }, wantErr: "owner decision"},
		{name: "blank reason", mutate: func(d *LoopDisposition) { d.Reason = " " }, wantErr: "reason is required"},
		{name: "missing time", mutate: func(d *LoopDisposition) { d.OccurredAt = time.Time{} }, wantErr: "occurred time is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			disposition := valid
			tt.mutate(&disposition)
			err := disposition.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}
