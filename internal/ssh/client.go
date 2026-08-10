package ssh

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"wireguardhub/internal/models"

	"golang.org/x/crypto/ssh"
)

type Client struct {
	client *ssh.Client
	*BaseExecutor
}

type ExecEvent struct {
	ServerID string `json:"serverId"`
	Kind     string `json:"kind"` // "command", "output", "done"
	Command  string `json:"command,omitempty"`
	Line     string `json:"line,omitempty"`
	Error    string `json:"error,omitempty"`
}

func Connect(server models.ServerConfig, jump Executor) (*Client, error) {
	config, addr, err := buildClientConfig(server)
	if err != nil {
		return nil, err
	}

	if jump != nil {
		sshClient, ok := jump.(*Client)
		if !ok {
			return nil, fmt.Errorf("jump host must be an SSH client")
		}
		conn, err := sshClient.client.Dial("tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("failed to dial %s through jump server: %w", addr, err)
		}

		nconn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to handshake with %s via jump server: %w", addr, err)
		}

		return &Client{client: ssh.NewClient(nconn, chans, reqs), BaseExecutor: &BaseExecutor{SudoPassword: server.Password}}, nil
	}

	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	return &Client{client: conn, BaseExecutor: &BaseExecutor{SudoPassword: server.Password}}, nil
}

func buildClientConfig(server models.ServerConfig) (*ssh.ClientConfig, string, error) {
	var authMethods []ssh.AuthMethod

	switch server.AuthMethod {
	case "password":
		authMethods = append(authMethods, ssh.Password(server.Password))
	case "key":
		var signer ssh.Signer
		var err error
		if server.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(server.PrivateKey), []byte(server.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(server.PrivateKey))
		}
		if err != nil {
			return nil, "", fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	default:
		return nil, "", fmt.Errorf("unsupported auth method: %s", server.AuthMethod)
	}

	config := &ssh.ClientConfig{
		User:            server.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", server.Host, server.Port)
	return config, addr, nil
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

// execInternal runs a command and returns its stdout/stderr/error without
// emitting any terminal events. The emitting wrappers (ExecWithInput) call
// this and add emit calls around it.
func (c *Client) execInternal(cmd string, input string) (string, string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	finalCmd, finalInput := c.RewriteSudo(cmd, input)

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	if finalInput != "" {
		session.Stdin = bytes.NewBufferString(finalInput)
	}

	err = session.Run(finalCmd)
	return stdout.String(), stderr.String(), err
}

// ExecStreaming runs a command and calls onLine for each line of stdout/stderr.
// Returns an io.Closer (the underlying session, can be closed to cancel) and error from Run.
func (c *Client) ExecStreaming(cmd string, onLine func(string)) (io.Closer, error) {
	c.Emit(ExecEvent{Kind: "command", Command: cmd})

	session, err := c.client.NewSession()
	if err != nil {
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}
		c.Emit(ExecEvent{Kind: "done", Error: errMsg})
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	finalCmd, finalInput := c.RewriteSudo(cmd, "")

	if finalInput != "" {
		session.Stdin = bytes.NewBufferString(finalInput)
	}

	// Stream stdout and stderr line-by-line
	pr, pw := io.Pipe()
	session.Stdout = pw
	session.Stderr = pw

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
				c.Emit(ExecEvent{Kind: "output", Line: line})
			}
		}
	}()

	err = session.Run(finalCmd)
	pw.Close()
	<-done
	session.Close()

	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	c.Emit(ExecEvent{Kind: "done", Error: errMsg})

	return session, err
}

func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

func (c *Client) IsConnected() bool {
	if c.client == nil {
		return false
	}
	session, err := c.client.NewSession()
	if err != nil {
		return false
	}
	defer session.Close()
	return session.Run("true") == nil
}

func TestConnection(server models.ServerConfig, jump Executor) (*models.TestConnectionResult, error) {
	client, err := Connect(server, jump)
	if err != nil {
		return &models.TestConnectionResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	defer client.Close()

	result, err := VerifyWGAndSudo(client, "SSH connected", true)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// VerifyWGAndSudo checks that WireGuard is installed and sudo works on the
// host. prefix is used in result messages (e.g. "SSH connected" or "Connected
// locally"). When requireSudoPassword is true and the user is not root, the
// check fails early if no sudo password is configured on an SSH client.
func VerifyWGAndSudo(client Executor, prefix string, requireSudoPassword bool) (models.TestConnectionResult, error) {
	stdout, _, err := client.Exec("wg --version")
	if err != nil {
		return models.TestConnectionResult{
			Success: true,
			Message: prefix + ", but WireGuard may not be installed. " + stdout,
		}, nil
	}

	uidOut, _, _ := client.Exec("id -u")
	isRoot := strings.TrimSpace(uidOut) == "0"

	if !isRoot {
		if requireSudoPassword {
			if c, ok := client.(*Client); ok && c.SudoPassword == "" {
				return models.TestConnectionResult{
					Success: false,
					Message: prefix + ", but no sudo password configured. Configure sudo credentials for this server.",
				}, nil
			}
		}
		_, stderr, err := client.Exec("sudo true")
		if err != nil {
			return models.TestConnectionResult{
				Success: false,
				Message: prefix + ", but sudo authentication failed: " + strings.TrimSpace(stderr),
			}, nil
		}
	}

	return models.TestConnectionResult{
		Success: true,
		Message: prefix + ". " + stdout,
	}, nil
}
