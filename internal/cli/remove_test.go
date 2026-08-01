package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Alesion30/gw/internal/finder"
)

func TestRunRemoveSelected(t *testing.T) {
	root, e, fdr := newTestEnv(t)
	wt := filepath.Join(root, ".worktrees", "feat/a")
	runGit(t, root, "worktree", "add", "-b", "feat/a", wt)

	fdr.result = finder.Result{Index: 0}

	if err := runRemove(e, "", removeOptions{}); err != nil {
		t.Fatalf("runRemove() = %v", err)
	}
	assertNotExists(t, wt)
}

func TestRunRemoveExcludesMainWorktree(t *testing.T) {
	root, e, fdr := newTestEnv(t)
	wt := filepath.Join(root, ".worktrees", "feat/a")
	runGit(t, root, "worktree", "add", "-b", "feat/a", wt)

	fdr.result = finder.Result{Index: 0}

	if err := runRemove(e, "", removeOptions{}); err != nil {
		t.Fatalf("runRemove() = %v", err)
	}

	if len(fdr.items) != 1 {
		t.Fatalf("候補 = %v, want 1 件", fdr.items)
	}
	if strings.Contains(fdr.items[0], "[main]") {
		t.Errorf("メインの worktree が候補に含まれています: %v", fdr.items)
	}
}

func TestRunRemoveGoneRejectsQuery(t *testing.T) {
	_, e, _ := newTestEnv(t)

	err := runRemove(e, "feat", removeOptions{gone: true})
	if err == nil || !strings.Contains(err.Error(), "does not take a query") {
		t.Fatalf("runRemove() = %v, want query 拒否エラー", err)
	}
}

func TestRunRemoveDeleteBranchRequiresGone(t *testing.T) {
	_, e, _ := newTestEnv(t)

	err := runRemove(e, "", removeOptions{deleteBranch: true})
	if err == nil || !strings.Contains(err.Error(), "only available with --gone") {
		t.Fatalf("runRemove() = %v, want --gone 必須エラー", err)
	}
}

func TestRunRemoveGone(t *testing.T) {
	root, e, _ := newTestEnv(t)
	remote := setupRemote(t, root)

	// upstream つきのブランチを 2 本用意し、片方だけリモートから消す
	pushBranch(t, root, "feat/gone")
	pushBranch(t, root, "feat/alive")

	goneWt := filepath.Join(root, ".worktrees", "feat/gone")
	aliveWt := filepath.Join(root, ".worktrees", "feat/alive")
	runGit(t, root, "worktree", "add", goneWt, "feat/gone")
	runGit(t, root, "worktree", "add", aliveWt, "feat/alive")

	runGit(t, remote, "branch", "-D", "feat/gone")

	if err := runRemove(e, "", removeOptions{gone: true}); err != nil {
		t.Fatalf("runRemove() = %v", err)
	}

	assertNotExists(t, goneWt)
	assertDirExists(t, aliveWt)
	if !e.git.BranchExists("feat/gone") {
		t.Error("--delete-branch を指定していないのにローカルブランチが消えています")
	}
}

func TestRunRemoveGoneDeleteBranch(t *testing.T) {
	root, e, _ := newTestEnv(t)
	remote := setupRemote(t, root)

	pushBranch(t, root, "feat/gone")
	wt := filepath.Join(root, ".worktrees", "feat/gone")
	runGit(t, root, "worktree", "add", wt, "feat/gone")

	runGit(t, remote, "branch", "-D", "feat/gone")

	if err := runRemove(e, "", removeOptions{gone: true, deleteBranch: true}); err != nil {
		t.Fatalf("runRemove() = %v", err)
	}

	assertNotExists(t, wt)
	if e.git.BranchExists("feat/gone") {
		t.Error("ローカルブランチ feat/gone が残っています")
	}
}

func TestRunRemoveGoneSkipsCurrentWorktree(t *testing.T) {
	root, e, _ := newTestEnv(t)
	remote := setupRemote(t, root)

	pushBranch(t, root, "feat/gone")
	wt := filepath.Join(root, ".worktrees", "feat/gone")
	runGit(t, root, "worktree", "add", wt, "feat/gone")
	runGit(t, remote, "branch", "-D", "feat/gone")

	// 削除対象の worktree にいる状態を再現する
	e.cwd = wt

	if err := runRemove(e, "", removeOptions{gone: true}); err != nil {
		t.Fatalf("runRemove() = %v", err)
	}
	assertDirExists(t, wt)
}

func TestRunRemoveGoneDeclined(t *testing.T) {
	root, e, _ := newTestEnv(t)
	remote := setupRemote(t, root)

	pushBranch(t, root, "feat/gone")
	wt := filepath.Join(root, ".worktrees", "feat/gone")
	runGit(t, root, "worktree", "add", wt, "feat/gone")
	runGit(t, remote, "branch", "-D", "feat/gone")

	e.confirm = func(string) bool { return false }

	err := runRemove(e, "", removeOptions{gone: true})
	if err == nil || err.Error() != "canceled" {
		t.Fatalf("runRemove() = %v, want canceled", err)
	}
	assertDirExists(t, wt)
}

// setupRemote は bare リポジトリを origin として登録し、そのパスを返す。
func setupRemote(t *testing.T, root string) string {
	t.Helper()

	remote := filepath.Join(resolve(t, t.TempDir()), "origin.git")
	runGit(t, filepath.Dir(remote), "init", "--bare", "-b", "main", remote)
	runGit(t, root, "remote", "add", "origin", remote)
	runGit(t, root, "push", "-u", "origin", "main")

	return remote
}

// pushBranch は main からブランチを切って origin へ push し、main に戻る。
func pushBranch(t *testing.T, root, branch string) {
	t.Helper()

	runGit(t, root, "branch", branch)
	runGit(t, root, "push", "-u", "origin", branch)
}
