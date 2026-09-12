package supervisorcap

import "testing"

func TestAuthorityBindsCapabilityToSessionAndLaunchGeneration(t *testing.T) {
	authority := NewAuthority()
	token, verifier, err := authority.Issue("session-1", "launch-1")
	if err != nil {
		t.Fatal(err)
	}
	if token == "" || verifier == "" {
		t.Fatal("authority returned an empty credential")
	}
	if !authority.Valid("session-1", "launch-1", token, verifier) {
		t.Fatal("owning session generation rejected its capability")
	}
	if authority.Valid("session-2", "launch-1", token, verifier) {
		t.Fatal("capability authorized a different session")
	}
	if authority.Valid("session-1", "launch-2", token, verifier) {
		t.Fatal("capability authorized a different launch generation")
	}
	if authority.Valid("session-1", "launch-1", verifier, verifier) {
		t.Fatal("durable verifier worked as a bearer token")
	}
}

func TestAuthorityVerifierSurvivesDaemonReplacement(t *testing.T) {
	first := NewAuthority()
	token, verifier, err := first.Issue("session-1", "launch-1")
	if err != nil {
		t.Fatal(err)
	}
	if !NewAuthority().Valid("session-1", "launch-1", token, verifier) {
		t.Fatal("replacement daemon rejected a surviving supervisor report")
	}
}
