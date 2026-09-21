package mcpstdio

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

const protocolVersion = "2025-11-25"

type ProcessConfig struct {
	Command string
	Args    []string
	Dir     string
	Env     []string
}

type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema"`
}

type CallResult struct {
	Content           []map[string]any `json:"content,omitempty"`
	StructuredContent any              `json:"structuredContent,omitempty"`
	IsError           bool             `json:"isError,omitempty"`
}

func (r CallResult) Value() any {
	if r.StructuredContent != nil {
		return r.StructuredContent
	}
	return map[string]any{"content": r.Content, "isError": r.IsError}
}

type Client struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
	stderr  *lockedBuffer
	nextID  int64
	wait    sync.Once
	waitErr error
}

func Start(ctx context.Context, cfg ProcessConfig) (*Client, error) {
	if cfg.Command == "" {
		return nil, fmt.Errorf("MCP command is required")
	}
	cmd := exec.CommandContext(ctx, cfg.Command, cfg.Args...)
	cmd.Dir = cfg.Dir
	if len(cfg.Env) > 0 {
		cmd.Env = cfg.Env
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	c := &Client{
		cmd:    cmd,
		stdin:  stdin,
		stderr: &lockedBuffer{},
	}
	c.scanner = bufio.NewScanner(stdout)
	c.scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	go func() {
		_, _ = io.Copy(c.stderr, io.LimitReader(stderr, 1<<20))
	}()

	var initResult struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if err := c.request("initialize", map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "agent-bench",
			"version": "0.1.0",
		},
	}, &initResult); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("MCP initialize: %w", err)
	}
	if initResult.ProtocolVersion == "" {
		_ = c.Close()
		return nil, fmt.Errorf("MCP initialize returned no protocol version")
	}
	if err := c.notify("notifications/initialized", map[string]any{}); err != nil {
		_ = c.Close()
		return nil, err
	}
	return c, nil
}

func (c *Client) ListTools() ([]Tool, error) {
	var result struct {
		Tools []Tool `json:"tools"`
	}
	if err := c.request("tools/list", map[string]any{}, &result); err != nil {
		return nil, err
	}
	return result.Tools, nil
}

func (c *Client) CallTool(name string, args map[string]any) (CallResult, error) {
	var result CallResult
	if err := c.request("tools/call", map[string]any{
		"name":      name,
		"arguments": args,
	}, &result); err != nil {
		return CallResult{}, err
	}
	if result.IsError {
		return result, fmt.Errorf("MCP tool %s returned isError", name)
	}
	return result, nil
}

func (c *Client) request(method string, params any, out any) error {
	c.nextID++
	id := c.nextID
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}
	if err := c.write(payload); err != nil {
		return err
	}

	for c.scanner.Scan() {
		line := c.scanner.Bytes()
		var envelope struct {
			ID     json.RawMessage `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
				Data    any    `json:"data,omitempty"`
			} `json:"error,omitempty"`
		}
		if err := json.Unmarshal(line, &envelope); err != nil {
			return fmt.Errorf("decode MCP response: %w", err)
		}
		if len(envelope.ID) == 0 {
			continue
		}
		var responseID int64
		if err := json.Unmarshal(envelope.ID, &responseID); err != nil || responseID != id {
			continue
		}
		if envelope.Error != nil {
			return fmt.Errorf("MCP error %d: %s", envelope.Error.Code, envelope.Error.Message)
		}
		if out == nil {
			return nil
		}
		if err := json.Unmarshal(envelope.Result, out); err != nil {
			return fmt.Errorf("decode MCP result for %s: %w", method, err)
		}
		return nil
	}
	if err := c.scanner.Err(); err != nil {
		return err
	}
	return fmt.Errorf("MCP server closed stdout: %s", c.stderr.String())
}

func (c *Client) notify(method string, params any) error {
	return c.write(map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	})
}

func (c *Client) write(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = c.stdin.Write(b)
	return err
}

func (c *Client) Close() error {
	if c == nil || c.cmd == nil {
		return nil
	}
	_ = c.stdin.Close()
	c.wait.Do(func() { c.waitErr = c.cmd.Wait() })
	return c.waitErr
}

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

func InheritEnv(extra ...string) []string {
	return append(os.Environ(), extra...)
}
