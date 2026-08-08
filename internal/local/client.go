package local

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"wireguardhub/internal/ssh"
)

// Client executes commands on the local machine via os/exec.
// It implements ssh.Executor so that wireguard.Service can use it
// interchangeably with the SSH client.
type Client struct {
	sudoPassword string
	ServerID     string
	OnExec       func(ssh.ExecEvent)
}

// NewClient creates a local execution client. The sudoPassword is used
// to handle sudo commands the same way the SSH client does (sudo -S with
// password piped via stdin).
func NewClient(sudoPassword string) *Client {
	return &Client{sudoPassword: sudoPassword}
}

func (c *Client) emit(e ssh.ExecEvent) {
	if c.OnExec != nil {
		e.ServerID = c.ServerID
		c.OnExec(e)
	}
}

func (c *Client) Exec(cmd string) (string, string, error) {
	return c.ExecWithInput(cmd, "")
}

func (c *Client) ExecWithInput(cmd string, input string) (string, string, error) {
	c.emit(ssh.ExecEvent{Kind: "command", Command: cmd})
	stdout, stderr, err := c.execInternal(cmd, input)
	for _, line := range strings.Split(strings.TrimRight(stdout, "\n"), "\n") {
		if line != "" {
			c.emit(ssh.ExecEvent{Kind: "output", Line: line})
		}
	}
	if stderr != "" {
		for _, line := range strings.Split(strings.TrimRight(stderr, "\n"), "\n") {
			if line != "" {
				c.emit(ssh.ExecEvent{Kind: "output", Line: line})
			}
		}
	}
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	c.emit(ssh.ExecEvent{Kind: "done", Error: errMsg})
	return stdout, stderr, err
}

// ExecSilent runs a command without emitting terminal events.
func (c *Client) ExecSilent(cmd string) (string, string, error) {
	return c.ExecWithInputSilent(cmd, "")
}

// ExecWithInputSilent runs a command with stdin input without emitting
// terminal events. Use for commands that handle sensitive data.
func (c *Client) ExecWithInputSilent(cmd string, input string) (string, string, error) {
	return c.execInternal(cmd, input)
}

// execInternal runs a command locally and returns stdout/stderr/error
// without emitting any terminal events.
func (c *Client) execInternal(cmd string, input string) (string, string, error) {
	finalCmd := cmd
	finalInput := input
	if strings.HasPrefix(cmd, "sudo ") && !strings.HasPrefix(cmd, "sudo -S") {
		if c.sudoPassword != "" {
			finalCmd = "sudo -S -p '' " + strings.TrimPrefix(cmd, "sudo ")
			finalInput = c.sudoPassword + "\n" + input
		} else {
			return "", "", fmt.Errorf("no sudo password configured")
		}
	}

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
	c.emit(ssh.ExecEvent{Kind: "command", Command: cmd})

	finalCmd := cmd
	finalInput := ""
	if strings.HasPrefix(cmd, "sudo ") && !strings.HasPrefix(cmd, "sudo -S") {
		if c.sudoPassword != "" {
			finalCmd = "sudo -S -p '' " + strings.TrimPrefix(cmd, "sudo ")
			finalInput = c.sudoPassword + "\n"
		} else {
			return nil, fmt.Errorf("no sudo password configured")
		}
	}

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
				c.emit(ssh.ExecEvent{Kind: "output", Line: line})
			}
		}
	}()

	err := ec.Start()
	if err != nil {
		pw.Close()
		c.emit(ssh.ExecEvent{Kind: "done", Error: err.Error()})
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
		c.emit(ssh.ExecEvent{Kind: "done", Error: errMsg})
	}()

	return &processCloser{cmd: ec}, nil
}

func (c *Client) IsConnected() bool {
	return true
}

func (c *Client) Close() error {
	return nil
}
