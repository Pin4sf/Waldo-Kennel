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
const developerToolPATH = "/Applications/Xcode.app/Contents/Developer/usr/bin:/Library/Developer/CommandLineTools/usr/bin:"

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
	path := checkPATH
	// /usr/bin/python3 is an xcrun shim that tries to create a cache outside
	// the authorized workspace before the approved script starts. Prefer the
	// installed developer runtime for this one executable name so the same
	// Python runs without widening write authority for the check.
	if req.Argv[0] == "python3" {
		path = developerToolPATH + path
	}
	resolved, err := lookPathIn(req.Argv[0], path)
	if err != nil {
		return nil, fmt.Errorf("%w: executable %q is not on the check path", ErrInvalidCommand, req.Argv[0])
	}
	argv := append([]string{"-p", s.profile(), "-D", "WORKSPACE=" + root, resolved}, req.Argv[1:]...)
	cmd := exec.CommandContext(ctx, sandboxExecPath, argv...)
	cmd.Dir = root
	cmd.Env = safeEnvironment(req.Environment)
	cmd.Env[0] = "PATH=" + path
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
		// Apple's /usr/bin/python3 is an xcrun shim on current macOS and loads
		// its signed runtime from the selected system developer installation.
		// These roots are executable runtime resources, not owner workspace
		// data; without them an approved Python check is reported as a false
		// work failure before its script starts.
		`(allow file-read* (subpath "/System") (subpath "/usr/lib") (subpath "/usr/bin") (subpath "/bin") (subpath "/usr/share") (subpath "/dev") (subpath "/private/preboot") (subpath "/private/var/db/dyld") (subpath "/Library/Apple") (subpath "/Library/Developer/CommandLineTools") (subpath "/Applications/Xcode.app/Contents/Developer"))`,
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
