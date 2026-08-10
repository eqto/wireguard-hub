package local

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"

	"wireguardhub/internal/ssh"
)

// Client executes commands on the local machine via os/exec.
// It implements ssh.Executor so that wireguard.Service can use it
// interchangeably with the SSH client.
type Client struct {
	*ssh.BaseExecutor
}

// NewClient creates a local execution client. The sudoPassword is used
// to handle sudo commands the same way the SSH client does (sudo -S with
// password piped via stdin).
func NewClient(sudoPassword string) *Client {
	return &Client{BaseExecutor: &ssh.BaseExecutor{SudoPassword: sudoPassword}}
}

func (c *Client) Exec(cmd string) (string, string, error) {
	return c.ExecWithInput(cmd, "")
}

// ExecF is a printf-style wrapper around Exec.
func (c *Client) ExecF(format string, args ...any) (string, string, error) {
	return c.Exec(fmt.Sprintf(format, args...))
}

// ExecSilentF is a printf-style wrapper around ExecSilent.
func (c *Client) ExecSilentF(format string, args ...any) (string, string, error) {
	return c.ExecSilent(fmt.Sprintf(format, args...))
}

// ExecWithInputF is a printf-style wrapper around ExecWithInput.
// input is passed first so the variadic args slice can be last.
func (c *Client) ExecWithInputF(input, format string, args ...any) (string, string, error) {
	return c.ExecWithInput(fmt.Sprintf(format, args...), input)
}

// ExecWithInputSilentF is a printf-style wrapper around ExecWithInputSilent.
// input is passed first so the variadic args slice can be last.
func (c *Client) ExecWithInputSilentF(input, format string, args ...any) (string, string, error) {
	return c.ExecWithInputSilent(fmt.Sprintf(format, args...), input)
}

func (c *Client) ExecWithInput(cmd string, input string) (string, string, error) {
	return c.ExecWithEmit(c.execInternal, cmd, input)
}

// ExecSilent runs a command without emitting terminal events.
func (c *Client) ExecSilent(cmd string) (string, string, error) {
	return c.ExecWithInputSilent(cmd, "")
}

// CommandExists checks whether a binary is available on the server's PATH.
func (c *Client) CommandExists(name string) bool {
	_, _, err := c.ExecSilent("command -v " + name)
	return err == nil
}

// ExecWithInputSilent runs a command with stdin input without emitting
// terminal events. Use for commands that handle sensitive data.
func (c *Client) ExecWithInputSilent(cmd string, input string) (string, string, error) {
	return c.execInternal(cmd, input)
}

// execInternal runs a command locally and returns stdout/stderr/error
// without emitting any terminal events.
func (c *Client) execInternal(cmd string, input string) (string, string, error) {
	if ssh.NeedsSudoPassword(cmd) && c.SudoPassword == "" {
		return "", "", fmt.Errorf("no sudo password configured")
	}
	finalCmd, finalInput := c.RewriteSudo(cmd, input)

	ec := exec.Command("bash", "-c", finalCmd)

	var stdout, stderr bytes.Buffer
	ec.Stdout = &stdout
	ec.Stderr = &stderr
	if finalInput != "" {
		ec.Stdin = bytes.NewBufferString(finalInput)
	}

	err := ec.Run()
	return stdout.String(), stderr.String(), err
}

// processCloser wraps an os/exec.Cmd process so it can be used as io.Closer
// for cancellation (sends SIGTERM to the process).
type processCloser struct {
	cmd *exec.Cmd
}

func (p *processCloser) Close() error {
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Kill()
	}
	return nil
}

func (c *Client) ExecStreaming(cmd string, onLine func(string)) (io.Closer, error) {
	c.Emit(ssh.ExecEvent{Kind: "command", Command: cmd})

	if ssh.NeedsSudoPassword(cmd) && c.SudoPassword == "" {
		return nil, fmt.Errorf("no sudo password configured")
	}
	finalCmd, finalInput := c.RewriteSudo(cmd, "")

	ec := exec.Command("bash", "-c", finalCmd)

	if finalInput != "" {
		ec.Stdin = bytes.NewBufferString(finalInput)
	}

	pr, pw := io.Pipe()
	ec.Stdout = pw
	ec.Stderr = pw

	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(pr)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if onLine != nil {
				onLine(line)
			} else {
				c.Emit(ssh.ExecEvent{Kind: "output", Line: line})
			}
		}
	}()

	err := ec.Start()
	if err != nil {
		pw.Close()
		c.Emit(ssh.ExecEvent{Kind: "done", Error: err.Error()})
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	go func() {
		ec.Wait()
		pw.Close()
		<-done
		errMsg := ""
		if ec.ProcessState != nil && !ec.ProcessState.Success() {
			errMsg = fmt.Sprintf("exit status %d", ec.ProcessState.ExitCode())
		}
		c.Emit(ssh.ExecEvent{Kind: "done", Error: errMsg})
	}()

	return &processCloser{cmd: ec}, nil
}

func (c *Client) IsConnected() bool {
	return true
}

func (c *Client) Close() error {
	return nil
}
