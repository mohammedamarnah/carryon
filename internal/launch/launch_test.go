package launch

import (
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/mohammedamarnah/carryon/internal/model"
)

// fakeInfo implements os.FileInfo for tests.
type fakeInfo struct {
	dir  bool
	mode fs.FileMode
}

func (f fakeInfo) Name() string       { return "x" }
func (f fakeInfo) Size() int64        { return 0 }
func (f fakeInfo) Mode() fs.FileMode  { return f.mode }
func (f fakeInfo) ModTime() time.Time { return time.Time{} }
func (f fakeInfo) IsDir() bool        { return f.dir }
func (f fakeInfo) Sys() any           { return nil }

func goodEnv() Env {
	return Env{
		Shell: "/bin/zsh",
		Stat: func(p string) (os.FileInfo, error) {
			switch p {
			case "/bin/zsh":
				return fakeInfo{mode: 0o755}, nil
			case "/Users/me/proj":
				return fakeInfo{dir: true, mode: fs.ModeDir | 0o755}, nil
			}
			return nil, os.ErrNotExist
		},
		Getwd: func() (string, error) { return "/current/dir", nil },
	}
}

func validConv() model.Conversation {
	return model.Conversation{
		Tool:      model.Claude,
		SessionID: "a82756b2-a42e-4d8b-ab86-a2cc889e890d",
		Cwd:       "/Users/me/proj",
	}
}

func TestBuildHappyPath(t *testing.T) {
	spec, warnings, err := Build(validConv(), goodEnv())
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if spec.Shell != "/bin/zsh" {
		t.Errorf("Shell = %q", spec.Shell)
	}
	if spec.Dir != "/Users/me/proj" {
		t.Errorf("Dir = %q", spec.Dir)
	}
	want := []string{"/bin/zsh", "-ic", `exec claude --resume "$1"`, "carryon", "a82756b2-a42e-4d8b-ab86-a2cc889e890d"}
	if len(spec.Argv) != len(want) {
		t.Fatalf("Argv = %v", spec.Argv)
	}
	for i := range want {
		if spec.Argv[i] != want[i] {
			t.Errorf("Argv[%d] = %q, want %q", i, spec.Argv[i], want[i])
		}
	}
}

func TestBuildRejectsBadSessionID(t *testing.T) {
	conv := validConv()
	conv.SessionID = `'; rm -rf ~ #`
	if _, _, err := Build(conv, goodEnv()); err == nil {
		t.Fatal("expected error for non-UUID session ID")
	}
}

func TestBuildIDNeverInCommandString(t *testing.T) {
	spec, _, err := Build(validConv(), goodEnv())
	if err != nil {
		t.Fatal(err)
	}
	// argv[2] is the command the shell parses; it must be the constant template,
	// never contain the session ID.
	if spec.Argv[2] != `exec claude --resume "$1"` {
		t.Errorf("command string = %q (must be constant)", spec.Argv[2])
	}
	if spec.Argv[len(spec.Argv)-1] != validConv().SessionID {
		t.Errorf("session ID must be the final positional arg")
	}
}

func TestBuildShellFallback(t *testing.T) {
	env := goodEnv()
	env.Shell = "" // unset
	spec, warnings, err := Build(validConv(), env)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Shell != "/bin/sh" {
		t.Errorf("Shell = %q, want /bin/sh fallback", spec.Shell)
	}
	if len(warnings) == 0 {
		t.Error("expected a warning about shell fallback")
	}
}

func TestBuildMissingCwd(t *testing.T) {
	conv := validConv()
	conv.Cwd = "/gone"
	spec, warnings, err := Build(conv, goodEnv())
	if err != nil {
		t.Fatal(err)
	}
	if spec.Dir != "/current/dir" {
		t.Errorf("Dir = %q, want current-dir fallback", spec.Dir)
	}
	if len(warnings) == 0 {
		t.Error("expected a warning about missing cwd")
	}
}
