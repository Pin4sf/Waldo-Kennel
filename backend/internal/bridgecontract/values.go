// Package bridgecontract owns the v0.2.3 wire values and limits.
// Changes to these normative values require a reviewed contract version.
package bridgecontract

const (
	Version                   = "0.2.3"
	RedeemPath                = "/devices/redeem"
	ConnectPath               = "/devices/connect"
	ConnectVersionQueryPrefix = "?contract_version="
	ConnectCapabilitiesQuery  = "&declared_capabilities="
	TypeCommand               = "command"
	TypeAck                   = "ack"
	TypeResult                = "result"
	TypeReceipt               = "receipt"
	TypeHeartbeat             = "heartbeat"
	ClassMachineStateQuery    = "machine_state_query"
	ClassNotifyLocal          = "notify_local"
	AckAccepted               = "accepted"
	AckRejected               = "rejected"
	AckExpired                = "expired"
	ResultAnswered            = "answered"
	ResultDelivered           = "delivered"
	ResultFailed              = "failed"
	ReasonInvalidShape        = "invalid_shape"
	ReasonVersionMismatch     = "version_mismatch"
	ReasonIdempotencyConflict = "idempotency_conflict"
	ReasonUnknownCommand      = "unknown_command"
	ReasonUnknownMessage      = "unknown_message"
	ReasonExpired             = "expired"
	ReasonDeliveryUnknown     = "delivery_unknown"
	ReasonProcessingFailed    = "processing_failed"
	PairingCodeBytes          = 32
	PairingCodeExpirySeconds  = 600
	QuerySessionStatus        = "session_status"
	QueryAttemptStatus        = "attempt_status"
	QueryWorktreeWatch        = "worktree_watch"
	StateIdle                 = "idle"
	StateRunning              = "running"
	StateDone                 = "done"
	StateFailed               = "failed"
	StateUnknown              = "unknown"
	StateWatching             = "watching"
	StateStopped              = "stopped"
	SeverityInfo              = "info"
	SeverityWarning           = "warning"
	SeverityError             = "error"
	NonceBytes                = 16
	NonceEncodedLength        = 22
	SignatureEncodedLength    = 86
	ULIDLength                = 26
	ReplayWindowSeconds       = 300
	HeartbeatSeconds          = 30
	RedeemBodyMaxBytes        = 2048
	// Proposed local response resource bound; not a normative wire limit.
	RedeemResponseMaxBytes   = 4096
	FrameMaxBytes            = 8192
	IdentifierMaxBytes       = 128
	LabelMaxBytes            = 120
	TitleMaxBytes            = 120
	NotificationBodyMaxBytes = 1024
	OutboxDepthMax           = 1000000
	TombstonesPerDeviceMax   = 100000
	CommandTTLMaxSeconds     = 86400
	PairingCodeMaxBytes      = 43
	MaxTimestampDigits       = 10
	CommandRevision          = 1
)

// ValidCapabilities admits only the nonempty ordered K0 command subset.
func ValidCapabilities(c []string) bool {
	if len(c) == 0 || len(c) > 2 {
		return false
	}
	if c[0] == ClassMachineStateQuery {
		return len(c) == 1 || c[1] == ClassNotifyLocal
	}
	return len(c) == 1 && c[0] == ClassNotifyLocal
}
