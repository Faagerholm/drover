package tmux

import "testing"

func TestSocketFromEnv(t *testing.T) {
	for env, want := range map[string]string{
		"/private/tmp/tmux-501/default,1234,0": "/private/tmp/tmux-501/default",
		"/tmp/a,b/sock,99,12":                  "/tmp/a,b/sock",
		"":                                     "",
		"nocommas":                             "",
		"one,1":                                "",
	} {
		got, ok := SocketFromEnv(env)
		if got != want || ok != (want != "") {
			t.Errorf("SocketFromEnv(%q) = %q, %v; want %q", env, got, ok, want)
		}
	}
}

func TestOuterFromEnv(t *testing.T) {
	for _, c := range []struct {
		name, tmux, pane string
		ok               bool
	}{
		{"outside tmux", "", "", false},
		{"no pane (a raw popup)", "/tmp/tmux-501/default,1,0", "", false},
		{"in a pane", "/tmp/tmux-501/default,1,0", "%3", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("TMUX", c.tmux)
			t.Setenv("TMUX_PANE", c.pane)
			o, ok := OuterFromEnv()
			if ok != c.ok || (ok && (o.Socket != "/tmp/tmux-501/default" || o.Pane != "%3")) {
				t.Errorf("OuterFromEnv() = %+v, %v; want ok %v", o, ok, c.ok)
			}
		})
	}
}
