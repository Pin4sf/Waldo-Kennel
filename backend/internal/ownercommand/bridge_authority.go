package ownercommand

import "strings"

const BridgeAuthorizationScheme = "KennelBridge"

type BridgeAuthority struct{ token string }

func NewBridgeAuthority(token string) *BridgeAuthority {
	if !validToken(token) || len(token) != 43 {
		return nil
	}
	return &BridgeAuthority{token: token}
}
func (a *BridgeAuthority) Authenticate(header string) bool {
	if a == nil {
		return false
	}
	scheme, token, ok := strings.Cut(strings.TrimSpace(header), " ")
	return ok && scheme == BridgeAuthorizationScheme && TokenMatches(token, a.token)
}
