// Package governedtools exposes the frozen repository affordances of one
// admitted WorkUnit over a private stdio MCP connection.
package governedtools

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/governedcheck"
)

const maxTextBytes = 1 << 20

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// Server serves only the repository tools represented by one frozen policy.
type Server struct {
	Policy        domain.AttemptExecutionPolicy
	WorkspaceRoot string
	In            io.Reader
	Out           io.Writer
	RunCheck      func(context.Context, governedcheck.Request) (governedcheck.Result, error)
	root          *os.Root
}

// Serve runs the bounded MCP server until its stdio input closes.
func (s Server) Serve(ctx context.Context) error {
	if err := s.Policy.Validate(); err != nil {
		return err
	}
	root, err := filepath.EvalSymlinks(filepath.Clean(s.WorkspaceRoot))
	if err != nil || !filepath.IsAbs(root) {
		return errors.New("governed tools require an existing absolute workspace")
	}
	s.WorkspaceRoot = root
	if err := s.Policy.ValidateWorkspaceRoot(root); err != nil {
		return err
	}
	s.root, err = os.OpenRoot(root)
	if err != nil {
		return fmt.Errorf("open governed workspace: %w", err)
	}
	defer func() { _ = s.root.Close() }()
	if s.In == nil {
		s.In = os.Stdin
	}
	if s.Out == nil {
		s.Out = os.Stdout
	}
	scanner := bufio.NewScanner(s.In)
	scanner.Buffer(make([]byte, 64*1024), 2*maxTextBytes)
	enc := json.NewEncoder(s.Out)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		var req request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}
		if len(req.ID) == 0 {
			continue
		}
		var id interface{}
		_ = json.Unmarshal(req.ID, &id)
		result, callErr := s.handle(ctx, req)
		res := response{JSONRPC: "2.0", ID: id, Result: result}
		if callErr != nil {
			res.Result = nil
			res.Error = map[string]interface{}{"code": -32000, "message": callErr.Error()}
		}
		if err := enc.Encode(res); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (s Server) handle(ctx context.Context, req request) (interface{}, error) {
	switch req.Method {
	case "initialize":
		return map[string]interface{}{"protocolVersion": "2025-06-18", "capabilities": map[string]interface{}{"tools": map[string]interface{}{}}, "serverInfo": map[string]interface{}{"name": "kennel-governed-repository", "version": "1"}}, nil
	case "ping":
		return map[string]interface{}{}, nil
	case "tools/list":
		return map[string]interface{}{"tools": s.tools()}, nil
	case "tools/call":
		var p struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return nil, err
		}
		text, err := s.call(ctx, p.Name, p.Arguments)
		if err != nil {
			message := err.Error()
			if text != "" {
				message = text + "\nerror=" + message
			}
			return map[string]interface{}{"content": []interface{}{map[string]interface{}{"type": "text", "text": message}}, "isError": true}, nil
		}
		return map[string]interface{}{"content": []interface{}{map[string]interface{}{"type": "text", "text": text}}}, nil
	default:
		return nil, fmt.Errorf("unsupported method %q", req.Method)
	}
}

func tool(name, description string, properties map[string]interface{}, required ...string) map[string]interface{} {
	return map[string]interface{}{"name": name, "description": description, "inputSchema": map[string]interface{}{"type": "object", "properties": properties, "required": required, "additionalProperties": false}}
}

func (s Server) tools() []map[string]interface{} {
	path := map[string]interface{}{"path": map[string]interface{}{"type": "string", "description": "Workspace-relative path"}}
	var tools []map[string]interface{}
	if s.Policy.Has(domain.CapabilityWorktreeRead) {
		tools = append(tools,
			tool("list_repository", "List repository files beneath an optional workspace-relative path.", path),
			tool("read_text_file", "Read one UTF-8 text file inside the leased workspace.", path, "path"),
		)
	}
	if s.Policy.Has(domain.CapabilityWorktreeWrite) {
		tools = append(tools, tool("write_text_file", "Write one UTF-8 text file inside the leased workspace.", map[string]interface{}{"path": path["path"], "content": map[string]interface{}{"type": "string"}}, "path", "content"))
	}
	if s.Policy.Has(domain.CapabilityWorktreeExec) && len(s.Policy.ApprovedChecks) > 0 {
		ids := make([]string, 0, len(s.Policy.ApprovedChecks))
		for _, c := range s.Policy.ApprovedChecks {
			ids = append(ids, c.ID.String())
		}
		tools = append(tools, tool("run_approved_check", "Run one exact check frozen into the approved Plan. No arbitrary command is accepted.", map[string]interface{}{"check_id": map[string]interface{}{"type": "string", "enum": ids}}, "check_id"))
	}
	return tools
}

func stringArg(args map[string]interface{}, key string) (string, error) {
	v, ok := args[key].(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", key)
	}
	return v, nil
}

func (s Server) call(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	switch name {
	case "list_repository":
		if !s.Policy.Has(domain.CapabilityWorktreeRead) {
			return "", errors.New("repository read capability denied")
		}
		raw, _ := args["path"].(string)
		path, err := repositoryPath(raw, true)
		if err != nil {
			return "", err
		}
		root, err := s.root.OpenRoot(path)
		if err != nil {
			return "", err
		}
		defer func() { _ = root.Close() }()
		var files []string
		err = fs.WalkDir(root.FS(), ".", func(child string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if child == "." {
				return nil
			}
			if entry.Name() == ".git" || entry.Name() == "node_modules" {
				if entry.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			rel := child
			if path != "." {
				rel = filepath.Join(path, child)
			}
			files = append(files, filepath.ToSlash(rel))
			if len(files) > 5000 {
				return errors.New("repository listing exceeds 5000 files")
			}
			return nil
		})
		sort.Strings(files)
		return strings.Join(files, "\n"), err
	case "read_text_file":
		if !s.Policy.Has(domain.CapabilityWorktreeRead) {
			return "", errors.New("repository read capability denied")
		}
		raw, err := stringArg(args, "path")
		if err != nil {
			return "", err
		}
		path, err := repositoryPath(raw, false)
		if err != nil {
			return "", err
		}
		info, err := s.root.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			return "", errors.New("path is not a regular file")
		}
		if info.Size() > maxTextBytes {
			return "", errors.New("file exceeds 1 MiB")
		}
		b, err := s.root.ReadFile(path)
		if err != nil {
			return "", err
		}
		if strings.IndexByte(string(b), 0) >= 0 {
			return "", errors.New("binary files are not readable")
		}
		return string(b), nil
	case "write_text_file":
		if !s.Policy.Has(domain.CapabilityWorktreeWrite) {
			return "", errors.New("worktree.write was not granted")
		}
		raw, err := stringArg(args, "path")
		if err != nil {
			return "", err
		}
		content, err := stringArg(args, "content")
		if err != nil {
			return "", err
		}
		if len(content) > maxTextBytes {
			return "", errors.New("content exceeds 1 MiB")
		}
		path, err := repositoryPath(raw, false)
		if err != nil {
			return "", err
		}
		parent := filepath.Dir(path)
		if err := s.root.MkdirAll(parent, 0o750); err != nil {
			return "", err
		}
		tmpPath := filepath.Join(parent, ".kennel-write-"+uuid.NewString())
		tmp, err := s.root.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return "", err
		}
		defer func() { _ = s.root.Remove(tmpPath) }()
		if _, err = tmp.WriteString(content); err == nil {
			err = tmp.Chmod(0o600)
		}
		if closeErr := tmp.Close(); err == nil {
			err = closeErr
		}
		if err == nil {
			err = s.root.Rename(tmpPath, path)
		}
		if err != nil {
			return "", err
		}
		return "wrote " + filepath.ToSlash(raw), nil
	case "run_approved_check":
		id, err := stringArg(args, "check_id")
		if err != nil {
			return "", err
		}
		for _, check := range s.Policy.ApprovedChecks {
			if check.ID.String() != id {
				continue
			}
			run := s.RunCheck
			if run == nil {
				run = governedcheck.Run
			}
			result, runErr := run(ctx, governedcheck.Request{Policy: s.Policy, WorkspaceRoot: s.WorkspaceRoot, Argv: check.Argv, Timeout: time.Duration(check.TimeoutSeconds) * time.Second})
			text := fmt.Sprintf("exit_code=%d enforced_by=%s\n%s", result.ExitCode, result.EnforcedBy, result.Output)
			return text, runErr
		}
		return "", fmt.Errorf("check %q is not approved", id)
	default:
		return "", fmt.Errorf("tool %q is not granted", name)
	}
}

func repositoryPath(raw string, allowRoot bool) (string, error) {
	if raw == "" {
		raw = "."
	}
	if filepath.IsAbs(raw) {
		return "", errors.New("path must be workspace-relative")
	}
	path := filepath.Clean(raw)
	if path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes leased workspace")
	}
	if path == "." && !allowRoot {
		return "", errors.New("path must name a repository file")
	}
	for _, component := range strings.Split(filepath.ToSlash(path), "/") {
		if strings.EqualFold(component, ".git") {
			return "", errors.New("git custody metadata is not part of worktree file authority")
		}
	}
	return path, nil
}
