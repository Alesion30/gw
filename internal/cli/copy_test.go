package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Alesion30/gw/internal/git"
)

func TestRunCopy(t *testing.T) {
	root, e, _ := newTestEnv(t)
	writeFile(t, filepath.Join(root, ".env"), "TOKEN=secret\n")
	writeFile(t, filepath.Join(root, "config/local.yml"), "key: value\n")

	wt := addWorktree(t, root, e, "feat/a")

	if err := runCopy(e, []string{".env", "config/local.yml"}); err != nil {
		t.Fatalf("runCopy() = %v", err)
	}

	assertFileContent(t, filepath.Join(wt, ".env"), "TOKEN=secret\n")
	assertFileContent(t, filepath.Join(wt, "config/local.yml"), "key: value\n")
}

func TestRunCopyPreservesPermission(t *testing.T) {
	root, e, _ := newTestEnv(t)
	src := filepath.Join(root, "run.sh")
	writeFile(t, src, "#!/bin/sh\n")
	if err := os.Chmod(src, 0o700); err != nil {
		t.Fatal(err)
	}

	wt := addWorktree(t, root, e, "feat/a")

	if err := runCopy(e, []string{"run.sh"}); err != nil {
		t.Fatalf("runCopy() = %v", err)
	}

	info, err := os.Stat(filepath.Join(wt, "run.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Errorf("permission = %o, want 700", got)
	}
}

func TestRunCopyRejectsMainWorktree(t *testing.T) {
	root, e, _ := newTestEnv(t)
	writeFile(t, filepath.Join(root, ".env"), "TOKEN=secret\n")

	err := runCopy(e, []string{".env"})
	if err == nil || !strings.Contains(err.Error(), "main worktree") {
		t.Fatalf("runCopy() = %v, want main worktree rejection error", err)
	}
}

func TestRunCopyMissingFile(t *testing.T) {
	root, e, _ := newTestEnv(t)
	writeFile(t, filepath.Join(root, ".env"), "TOKEN=secret\n")

	wt := addWorktree(t, root, e, "feat/a")

	// 1 件でも失敗したらエラー扱いにするが、残りのコピーは続ける
	if err := runCopy(e, []string{".missing", ".env"}); err == nil {
		t.Fatal("runCopy() = nil, want error")
	}
	assertFileContent(t, filepath.Join(wt, ".env"), "TOKEN=secret\n")
}

// addWorktree は worktree を作り、env をその worktree の中にいる状態へ移す。
func addWorktree(t *testing.T, root string, e *env, branch string) string {
	t.Helper()

	wt := filepath.Join(root, ".worktrees", branch)
	runGit(t, root, "worktree", "add", "-b", branch, wt)

	e.cwd = wt
	e.git = git.Client{Dir: wt}

	return wt
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read %s: %v", path, err)
	}
	if string(got) != want {
		t.Errorf("%s = %q, want %q", path, got, want)
	}
}
