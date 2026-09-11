//go:build !windows

package supervisor

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
)

// maxUnixSocketPathBytes is the portable limit for sockaddr_un.sun_path on
// Darwin and Linux. Keep the address below it even when the user's state path
// is unusually long.
const maxUnixSocketPathBytes = 103

// Listen creates the Unix-domain listener for the supervisor watchdog. The
// endpoint is deliberately a short, per-daemon path under /tmp rather than a
// sibling of running.json: macOS rejects long sockaddr_un paths. The address
// is published in running.json for the Electron client to consume.
func Listen(runFilePath string) (net.Listener, string, error) {
	_ = runFilePath // retained in the cross-platform listener contract
	root := "/tmp"
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		root = os.TempDir()
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return nil, "", fmt.Errorf("generate supervisor socket identity: %w", err)
	}
	sockPath := root + "/kennel-supervise-" + hex.EncodeToString(random) + ".sock"
	if len([]byte(sockPath)) > maxUnixSocketPathBytes {
		return nil, "", fmt.Errorf("supervisor socket path is %d bytes; maximum is %d", len([]byte(sockPath)), maxUnixSocketPathBytes)
	}
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, "", err
	}
	// net.UnixListener normally unlinks by pathname on Close. Disable that
	// behavior so cleanup can verify the inode first and never remove an
	// unrelated object if the pathname is replaced while we are shutting down.
	unixLn, ok := ln.(*net.UnixListener)
	if !ok {
		_ = ln.Close()
		return nil, "", fmt.Errorf("supervisor listener has unexpected type %T", ln)
	}
	unixLn.SetUnlinkOnClose(false)
	fileInfo, err := os.Lstat(sockPath)
	if err != nil {
		_ = ln.Close()
		return nil, "", fmt.Errorf("inspect supervisor socket: %w", err)
	}
	if err := os.Chmod(sockPath, 0o600); err != nil {
		_ = ln.Close()
		if current, statErr := os.Lstat(sockPath); statErr == nil && os.SameFile(fileInfo, current) {
			_ = os.Remove(sockPath)
		}
		return nil, "", fmt.Errorf("restrict supervisor socket: %w", err)
	}
	return &cleanupListener{UnixListener: unixLn, path: sockPath, fileInfo: fileInfo}, sockPath, nil
}

type cleanupListener struct {
	*net.UnixListener
	path     string
	fileInfo os.FileInfo
}

func (l *cleanupListener) Close() error {
	err := l.UnixListener.Close()
	if current, statErr := os.Lstat(l.path); statErr == nil && os.SameFile(l.fileInfo, current) {
		if removeErr := os.Remove(l.path); err == nil {
			err = removeErr
		}
	}
	return err
}
