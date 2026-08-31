// Command backend is a compatibility wrapper for the Agent Orchestrator daemon.
// The user-facing CLI lives at cmd/kennel; keep this wrapper so existing `go run .`
// development workflows continue to start the daemon while scripts migrate.
package main

import (
	"fmt"
	"os"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/daemon"
)

func main() {
	if err := daemon.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "kennel backend daemon: "+err.Error())
		os.Exit(1)
	}
}
