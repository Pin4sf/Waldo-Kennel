package devicebridge

import (
	"crypto/ed25519"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestHTTPGoldenVectors(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	var nonce [NonceBytes]byte
	for i := range seed {
		seed[i] = byte(i)
	}
	for i := range nonce {
		nonce[i] = byte(i)
	}
	key := ed25519.NewKeyFromSeed(seed)
	for _, tc := range []struct{ name, body, digest, signature string }{
		{"redeem", `{"code":"AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8","contract_version":"0.2.3","declared_capabilities":["machine_state_query","notify_local"],"device_pubkey":"A6EHv_POEL4dcN0Y50vAmWfk1jCbpQ1fHdyGZBJVMbg","label":"Test Mac"}`, "90d6daa8d89edfae3a13f3584eb36cf04155539bf5c2c2ccf2f73de122ba6175", "VlVfMNC0TTbHKXsDsjGjWyBHnaVAMSud0b_DlEukL78UzWBdJhs_knA09uH0NSzsuFbWpLSU-xYhqKEsFGPbDw"},
		{"unicode", `{"code":"AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8","contract_version":"0.2.3","declared_capabilities":["machine_state_query"],"device_pubkey":"A6EHv_POEL4dcN0Y50vAmWfk1jCbpQ1fHdyGZBJVMbg","label":"Café <Go> 𐀀"}`, "cc34011527d0ae10bad46dcc1977f79898a14b2b4b71ce25f3d7b1280bb6e00a", "LN1dFNztIWTdRKIPPqWtUolXKHhmGygGapStEuvsxIftaLrwwGfo9V_cED3D5T1fQufFL4cAYsKjoBTBe4PhAA"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base, err := HTTPBaseString(time.Unix(1790200800, 0), nonce, "POST", RedeemPath, []byte(tc.body))
			want := "1790200800\nAAECAwQFBgcICQoLDA0ODw\nPOST\n/devices/redeem\n" + tc.digest
			if err != nil || base != want {
				t.Fatalf("base = %q, %v", base, err)
			}
			headers, err := SignHTTPHeaders(key, time.Unix(1790200800, 0), "POST", RedeemPath, []byte(tc.body), nonce, "")
			if err != nil || headers.Get("X-Waldo-Signature") != tc.signature || headers.Get("X-Waldo-Device-Id") != "" {
				t.Fatalf("headers = %v, %v", headers, err)
			}
			sig, err := base64.RawURLEncoding.DecodeString(tc.signature)
			if err != nil || !ed25519.Verify(key.Public().(ed25519.PublicKey), []byte(base), sig) {
				t.Fatal("golden signature does not verify")
			}
		})
	}
}

func TestConnectSigningExactTargetAndEmptyBody(t *testing.T) {
	var nonce [NonceBytes]byte
	target, err := ConnectRequestTarget([]string{ClassMachineStateQuery, ClassNotifyLocal})
	if err != nil {
		t.Fatal(err)
	}
	base, err := HTTPBaseString(time.Unix(1790200800, 0), nonce, "GET", target, nil)
	if err != nil || !strings.HasSuffix(base, "\nGET\n"+target+"\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855") {
		t.Fatalf("base = %q, %v", base, err)
	}
	for _, bad := range []string{"/devices/connect", target + "&x=1", "/devices/connect?declared_capabilities=machine_state_query&contract_version=0.2.3"} {
		if _, err := HTTPBaseString(time.Unix(1790200800, 0), nonce, "GET", bad, nil); err != ErrInvalidSigningInput {
			t.Fatalf("accepted %q: %v", bad, err)
		}
	}
}
