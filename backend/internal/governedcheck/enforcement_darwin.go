//go:build darwin

package governedcheck

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
)

// sandboxExecPath is Apple's seatbelt front end. It is the same mechanism the
// Codex CLI uses for its own workspace-write sandbox on macOS, which is why the
// policy translation below lines up with the one described in L1b.
const sandboxExecPath = "/usr/bin/sandbox-exec"

// checkPATH is the only search path a check may resolve its executable from.
// It matches the PATH the process is given, so the name that was allowed is the
// binary that runs.
const checkPATH = "/usr/local/bin:/usr/bin:/bin"

func platformEnforcement(policy domain.AttemptExecutionPolicy) (Enforcement, error) {
	info, err := os.Stat(sandboxExecPath)
	if err != nil || info.IsDir() || info.Mode().Perm()&0o111 == 0 {
		return nil, fmt.Errorf("%w: %s is not available on this host", ErrEnforcementUnavailable, sandboxExecPath)
	}
	return seatbelt{writable: policy.Has(domain.CapabilityWorktreeWrite)}, nil
}

// seatbelt confines a check with a generated sandbox profile.
//
// The profile denies by default and then grants back only what the approved
// policy asked for. Notably it never grants network: no capability in this
// codebase can request one, so a check that reaches the network is always
// exceeding its authority.
type seatbelt struct {
	// writable is worktree.write. Without it the check gets no filesystem write
	// at all, not merely a narrower one.
	writable bool
}

func (s seatbelt) Name() string {
	if s.writable {
		return "macos-seatbelt-workspace-write"
	}
	return "macos-seatbelt-read-only"
}

func (s seatbelt) Command(ctx context.Context, req Request, root string) (*exec.Cmd, error) {
	// Resolve the allowed name against the same constrained PATH the check will
	// run with, so the profile confines the binary that actually executes.
	resolved, err := lookPathIn(req.Argv[0], checkPATH)
	if err != nil {
		return nil, fmt.Errorf("%w: executable %q is not on the check path", ErrInvalidCommand, req.Argv[0])
	}
	argv := append([]string{"-p", s.profile(), "-D", "WORKSPACE=" + root, resolved}, req.Argv[1:]...)
	cmd := exec.CommandContext(ctx, sandboxExecPath, argv...)
	cmd.Dir = root
	cmd.Env = safeEnvironment(req.Environment)
	return cmd, nil
}

func (s seatbelt) profile() string {
	rules := []string{
		"(version 1)",
		"(deny default)",
		// Executable runtime resources are readable; owner data is confined
		// to the authorized workspace. Metadata alone exposes no file bytes.
		"(allow file-read-metadata)",
		"(allow file-read-xattr)",
		`(allow file-read* (literal "/"))`,
		`(allow file-read-data (subpath (param "WORKSPACE")))`,
		`(allow file-read* (subpath "/System") (subpath "/usr/lib") (subpath "/usr/bin") (subpath "/bin") (subpath "/usr/share") (subpath "/dev") (subpath "/private/preboot") (subpath "/private/var/db/dyld") (subpath "/Library/Apple"))`,
		"(allow process-exec)",
		"(allow process-fork)",
		"(allow sysctl-read)",
		"(allow mach-lookup)",
		"(allow signal (target self))",
		// Explicit, though (deny default) already covers it: no capability in
		// this codebase can request network, so any network effect is a check
		// exceeding its authority.
		"(deny network*)",
		`(allow file-write-data (literal "/dev/null"))`,
	}
	if s.writable {
		// The workspace subpath is passed as a parameter rather than
		// interpolated, so a path containing profile syntax cannot rewrite the
		// rules around it.
		rules = append(rules, `(allow file-write* (subpath (param "WORKSPACE")))`)
	}
	return strings.Join(rules, "\n") + "\n"
}

// lookPathIn resolves name against an explicit PATH rather than the daemon's
// own environment.
func lookPathIn(name, path string) (string, error) {
	for _, dir := range strings.Split(path, ":") {
		if dir == "" {
			continue
		}
		candidate := dir + "/" + name
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() || info.Mode().Perm()&0o111 == 0 {
			continue
		}
		return candidate, nil
	}
	return "", os.ErrNotExist
}
