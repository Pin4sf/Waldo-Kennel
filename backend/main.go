// Command backend starts the Kennel daemon for `go run .` development flows.
// Packaged and CLI builds use cmd/kennel.
package main

import (
	"fmt"
	"os"

	"github.com/aoagents/agent-orchestrator/backend/internal/daemon"
)

func main() {
	if err := daemon.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "kennel backend daemon: "+err.Error())
		os.Exit(1)
	}
}
