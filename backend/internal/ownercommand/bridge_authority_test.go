package ownercommand

import "testing"

func TestBridgeAuthorityConstantTimeAndNilSafe(t *testing.T) {
	var absent *BridgeAuthority
	if absent.Authenticate("KennelBridge " + browserToken) {
		t.Fatal("nil authenticated")
	}
	bridge := NewBridgeAuthority(browserToken)
	owner := NewAuthority(ownerToken, "apprun-test")
	for _, header := range []string{"", "KennelOwner " + ownerToken, "KennelOwner " + browserToken, "KennelBridge " + ownerToken, "KennelBridge wrong"} {
		if bridge.Authenticate(header) {
			t.Fatal("cross capability accepted")
		}
	}
	if !bridge.Authenticate("KennelBridge " + browserToken) {
		t.Fatal("correct token rejected")
	}
	for _, header := range []string{"KennelOwner " + browserToken, "KennelBridge " + browserToken} {
		if _, ok := owner.Authenticate(header); ok {
			t.Fatal("bridge token accepted for owner")
		}
	}
	if NewBridgeAuthority("short") != nil {
		t.Fatal("invalid credential created")
	}
}
