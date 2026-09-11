package daemonmeta

// ServiceName identifies the Kennel daemon in loopback health/readiness probes.
// The CLI uses it with the reported PID to avoid signaling an unrelated process
// when a stale run-file's PID has been reused.
const ServiceName = "kennel-daemon"

// BuildIdentity is injected by the desktop daemon build. It is reported by
// the running process rather than inferred from its executable path, because a
// rebuilt binary can replace the same path while an older process keeps serving.
var BuildIdentity = "dev"

// BuildRevision is the source revision used for the daemon build when known.
// It is diagnostic metadata; BuildIdentity remains the admission identity.
var BuildRevision = "unknown"
