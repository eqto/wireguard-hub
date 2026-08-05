package ssh

import (
	"bytes"
	"fmt"
	"time"

	"wireguardadmin/internal/models"

	"golang.org/x/crypto/ssh"
)

type Client struct {
	client *ssh.Client
}

func Connect(server models.ServerConfig, jump *Client) (*Client, error) {
	config, addr, err := buildClientConfig(server)
	if err != nil {
		return nil, err
	}

	if jump != nil {
		conn, err := jump.client.Dial("tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("failed to dial %s through jump server: %w", addr, err)
		}

		nconn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to handshake with %s via jump server: %w", addr, err)
		}

		return &Client{client: ssh.NewClient(nconn, chans, reqs)}, nil
	}

	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	return &Client{client: conn}, nil
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

func (c *Client) ExecWithInput(cmd string, input string) (string, string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	if input != "" {
		session.Stdin = bytes.NewBufferString(input)
	}

	err = session.Run(cmd)
	return stdout.String(), stderr.String(), err
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
	_, _, err := c.Exec("true")
	return err == nil
}

func TestConnection(server models.ServerConfig, jump *Client) (*models.TestConnectionResult, error) {
	client, err := Connect(server, jump)
	if err != nil {
		return &models.TestConnectionResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	defer client.Close()

	stdout, _, err := client.Exec("wg --version")
	if err != nil {
		return &models.TestConnectionResult{
			Success: true,
			Message: "SSH connected, but WireGuard may not be installed: " + stdout,
		}, nil
	}

	return &models.TestConnectionResult{
		Success: true,
		Message: "Connected. " + stdout,
	}, nil
}
