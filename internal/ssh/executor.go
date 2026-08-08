package ssh

import "io"

// Executor is the interface that both SSH and local clients implement.
// wireguard.Service uses this abstraction to run commands on either
// a remote server (via SSH) or the local machine (via os/exec).
type Executor interface {
	Exec(cmd string) (string, string, error)
	ExecWithInput(cmd string, input string) (string, string, error)
	ExecStreaming(cmd string, onLine func(string)) (io.Closer, error)
	IsConnected() bool
	Close() error
}
