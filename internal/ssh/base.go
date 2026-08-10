package ssh

import "strings"

// BaseExecutor holds the fields and helpers shared by ssh.Client (remote
// execution via SSH) and local.Client (execution via os/exec). Both
// implement the Executor interface; embedding *BaseExecutor gives them a
// single implementation of event emission, sudo-password rewriting, and the
// emit-wrapping used by the non-silent Exec methods.
type BaseExecutor struct {
	SudoPassword string
	ServerID     string
	OnExec       func(ExecEvent)
}

// execFunc is the inner command-runner signature shared by both clients.
type execFunc func(cmd, input string) (string, string, error)

// Emit sends a terminal event to the OnExec callback, if configured.
func (b *BaseExecutor) Emit(e ExecEvent) {
	if b.OnExec == nil {
		return
	}
	e.ServerID = b.ServerID
	b.OnExec(e)
}

// emitOutput emits stdout/stderr lines as "output" events to the terminal.
func (b *BaseExecutor) emitOutput(stdout, stderr string) {
	for _, line := range splitOutputLines(stdout) {
		if line != "" {
			b.Emit(ExecEvent{Kind: "output", Line: line})
		}
	}
	for _, line := range splitOutputLines(stderr) {
		if line != "" {
			b.Emit(ExecEvent{Kind: "output", Line: line})
		}
	}
}

// ExecWithEmit runs fn, emitting "command"/"output"/"done" events around it.
// Used by the non-silent Exec methods of both clients.
func (b *BaseExecutor) ExecWithEmit(fn execFunc, cmd, input string) (string, string, error) {
	b.Emit(ExecEvent{Kind: "command", Command: cmd})
	stdout, stderr, err := fn(cmd, input)
	b.emitOutput(stdout, stderr)
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	b.Emit(ExecEvent{Kind: "done", Error: errMsg})
	return stdout, stderr, err
}

// RewriteSudo converts a "sudo ..." command into "sudo -S -p ” ..." and
// prepends the sudo password to the input so it can be read from stdin. The
// command and input are returned unchanged when no sudo password is
// configured, the command doesn't start with "sudo ", or it already uses
// "sudo -S".
func (b *BaseExecutor) RewriteSudo(cmd, input string) (string, string) {
	if b.SudoPassword == "" || !NeedsSudoPassword(cmd) {
		return cmd, input
	}
	return "sudo -S -p '' " + strings.TrimPrefix(cmd, "sudo "), b.SudoPassword + "\n" + input
}

// NeedsSudoPassword reports whether cmd is a "sudo ..." command that has not
// already been rewritten to "sudo -S". Such commands need a password supplied
// via stdin when running non-interactively.
func NeedsSudoPassword(cmd string) bool {
	return strings.HasPrefix(cmd, "sudo ") && !strings.HasPrefix(cmd, "sudo -S")
}

// splitOutputLines splits s into lines, trimming a trailing newline so that a
// final newline doesn't produce an empty last element.
func splitOutputLines(s string) []string {
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}
