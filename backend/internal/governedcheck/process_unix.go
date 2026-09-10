//go:build !windows

package governedcheck

import (
	"os/exec"
	"syscall"
	"time"
)

func configureProcessGroup(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }

// terminateTree stops the whole process group the check created rather than
// only its leader. Setpgid makes the child a group leader, so its pid is also
// the group id and a negative pid addresses every descendant.
func terminateTree(cmd *exec.Cmd, grace time.Duration) error {
	if cmd.Process == nil {
		return nil
	}
	group := -cmd.Process.Pid
	_ = syscall.Kill(group, syscall.SIGTERM)
	// A descendant that ignores SIGTERM must not extend custody indefinitely.
	time.AfterFunc(grace, func() { _ = syscall.Kill(group, syscall.SIGKILL) })
	return nil
}
