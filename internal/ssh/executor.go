package ssh

import "io"

// Executor is the interface that both SSH and local clients implement.
// wireguard.Service uses this abstraction to run commands on either
// a remote server (via SSH) or the local machine (via os/exec).
type Executor interface {
	Exec(cmd string) (string, string, error)
	ExecWithInput(cmd string, input string) (string, string, error)
	ExecStreaming(cmd string, onLine func(string)) (io.Closer, error)
	// ExecSilent and ExecWithInputSilent behave like their non-silent
	// counterparts but do not emit command/output events to the terminal
	// panel. Use them for commands that read or write sensitive data
	// (e.g. WireGuard .conf files containing private keys).
	ExecSilent(cmd string) (string, string, error)
	ExecWithInputSilent(cmd string, input string) (string, string, error)
	// CommandExists checks whether a binary is available on the server's
	// PATH. It runs `command -v <name>` silently and returns true if the
	// command was found.
	CommandExists(name string) bool
	IsConnected() bool
	Close() error
}
