package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Alesion30/gw/internal/finder"
	"github.com/Alesion30/gw/internal/git"
)

func TestRunPath(t *testing.T) {
	root, e, fdr := newTestEnv(t)
	wt := filepath.Join(root, ".worktrees", "feat/a")
	runGit(t, root, "worktree", "add", "-b", "feat/a", wt)

	out := &bytes.Buffer{}
	e.stdout = out
	fdr.result = finder.Result{Index: 0}

	if err := runPath(e, ""); err != nil {
		t.Fatalf("runPath() = %v", err)
	}

	if got := out.String(); got != wt {
		t.Errorf("output = %q, want %q", got, wt)
	}
	// 末尾に改行を付けない（シェルの $(...) でそのまま cd に渡すため）
	if strings.HasSuffix(out.String(), "\n") {
		t.Error("output ends with a newline")
	}
}

func TestRunPathExcludesCurrentWorktree(t *testing.T) {
	root, e, fdr := newTestEnv(t)
	wt := filepath.Join(root, ".worktrees", "feat/a")
	runGit(t, root, "worktree", "add", "-b", "feat/a", wt)

	e.cwd = wt
	fdr.result = finder.Result{Index: 0}

	if err := runPath(e, ""); err != nil {
		t.Fatalf("runPath() = %v", err)
	}

	if len(fdr.items) != 1 || !strings.HasPrefix(fdr.items[0], root+"  ") {
		t.Errorf("candidates = %v, want only the main worktree", fdr.items)
	}
}

func TestRunPathNoOtherWorktrees(t *testing.T) {
	_, e, _ := newTestEnv(t)

	err := runPath(e, "")
	if err == nil || !strings.Contains(err.Error(), "no other worktrees") {
		t.Fatalf("runPath() = %v, want no other worktrees error", err)
	}
}

func TestWorktreeLabel(t *testing.T) {
	tests := []struct {
		name string
		wt   git.Worktree
		want string
	}{
		{
			name: "branch",
			wt:   git.Worktree{Path: "/repo", Branch: "main"},
			want: "/repo  [main]",
		},
		{
			name: "bare",
			wt:   git.Worktree{Path: "/repo.git", Bare: true},
			want: "/repo.git  (bare)",
		},
		{
			name: "detached",
			wt:   git.Worktree{Path: "/repo/wt", Head: "abc1234def", Detached: true},
			want: "/repo/wt  (abc1234 detached)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := worktreeLabel(tt.wt); got != tt.want {
				t.Errorf("worktreeLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}
