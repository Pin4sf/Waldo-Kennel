//go:build windows

package governedcheck

import "os/exec"

func configureProcessGroup(*exec.Cmd) {}
