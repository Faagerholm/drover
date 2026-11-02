package tmux

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Outer is the user's own tmux, the one drover itself runs in (not the
// private server): its socket and drover's pane on it.
type Outer struct {
	Socket, Pane string
}

// OuterFromEnv reads the outer tmux from $TMUX and $TMUX_PANE. ok is false
// when drover does not run in a tmux pane.
func OuterFromEnv() (o Outer, ok bool) {
	socket, ok := SocketFromEnv(os.Getenv("TMUX"))
	pane := os.Getenv("TMUX_PANE")
	if !ok || pane == "" {
		return Outer{}, false
	}
	return Outer{Socket: socket, Pane: pane}, true
}

// SocketFromEnv extracts the socket from $TMUX, "socket,server-pid,session-id".
// It is split from the right because the socket path may itself hold commas.
func SocketFromEnv(env string) (string, bool) {
	for range 2 {
		i := strings.LastIndexByte(env, ',')
		if i < 0 {
			return "", false
		}
		env = env[:i]
	}
	return env, env != ""
}

// selectTimeout bounds a select-pane on the outer server, so a hung server
// cannot pile up a stuck command per key press.
const selectTimeout = time.Second

// SelectPane moves to the pane next to drover's in dir ("L", "D", "U" or
// "R", as select-pane takes it). It is how drover hands ctrl+hjkl back at its
// edges, the way vim-tmux-navigator does from vim.
func (o Outer) SelectPane(dir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), selectTimeout)
	defer cancel()
	// -f /dev/null: should the server have gone, never load the user's
	// configuration. -S, not $TMUX: the private server is addressed with -L
	// and must not be confused with this one.
	return exec.CommandContext(ctx, "tmux", "-f", "/dev/null", "-S", o.Socket,
		"select-pane", "-t", o.Pane, "-"+dir).Run()
}
