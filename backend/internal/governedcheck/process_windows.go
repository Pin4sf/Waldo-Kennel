//go:build windows

package governedcheck

import (
	"os/exec"
	"time"
)

func configureProcessGroup(*exec.Cmd) {}

// terminateTree is best-effort on Windows: no job object is configured here, so
// only the launched process is stopped. A check whose descendants outlive it is
// reported through Result.TerminationUnknown rather than assumed finished.
func terminateTree(cmd *exec.Cmd, _ time.Duration) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
