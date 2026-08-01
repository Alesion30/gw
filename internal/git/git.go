// Package git は git コマンドの実行と出力のパースを担う。
package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Client は特定のディレクトリで git を実行する。
// Dir が空ならカレントディレクトリで実行する。
type Client struct {
	Dir string
}

// Output は git を実行し、標準出力を末尾の改行を除いて返す。
// 失敗した場合は標準エラー出力をエラーメッセージに含める。
func (c Client) Output(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = c.Dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}

// Run は git を実行し、出力をそのまま親プロセスの stdout/stderr へ流す。
func (c Client) Run(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = c.Dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Worktree は `git worktree list --porcelain` の 1 ブロックに対応する。
type Worktree struct {
	Path     string
	Head     string
	Branch   string // refs/heads/ を剥がした短縮名。detached なら空
	Bare     bool
	Detached bool
	Locked   bool
	Prunable bool
}

// ParseWorktreeList は `git worktree list --porcelain` の出力を解析する。
// ブロックは空行区切りで、先頭が必ずメインの worktree になる。
func ParseWorktreeList(out string) []Worktree {
	var (
		worktrees []Worktree
		current   *Worktree
	)

	flush := func() {
		if current != nil {
			worktrees = append(worktrees, *current)
			current = nil
		}
	}

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			flush()
			continue
		}

		key, value, _ := strings.Cut(line, " ")
		switch key {
		case "worktree":
			flush()
			current = &Worktree{Path: value}
		case "HEAD":
			if current != nil {
				current.Head = value
			}
		case "branch":
			if current != nil {
				current.Branch = strings.TrimPrefix(value, "refs/heads/")
			}
		case "bare":
			if current != nil {
				current.Bare = true
			}
		case "detached":
			if current != nil {
				current.Detached = true
			}
		case "locked":
			if current != nil {
				current.Locked = true
			}
		case "prunable":
			if current != nil {
				current.Prunable = true
			}
		}
	}
	flush()

	return worktrees
}

// Worktrees は worktree の一覧を返す。先頭がメインの worktree。
func (c Client) Worktrees() ([]Worktree, error) {
	out, err := c.Output("worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return ParseWorktreeList(out), nil
}

// RepoRoot はリポジトリのルートを返す。
// bare リポジトリでは --show-toplevel が失敗するため、メインの worktree で代替する。
func (c Client) RepoRoot() (string, error) {
	if root, err := c.Output("rev-parse", "--show-toplevel"); err == nil {
		return root, nil
	}

	worktrees, err := c.Worktrees()
	if err != nil {
		return "", err
	}
	if len(worktrees) == 0 {
		return "", fmt.Errorf("リポジトリのルートを特定できません")
	}
	return worktrees[0].Path, nil
}

// LocalBranches はローカルブランチの短縮名を一覧で返す。
func (c Client) LocalBranches() ([]string, error) {
	out, err := c.Output("branch", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}
	return splitLines(out), nil
}

// GoneBranches は upstream が消えた（gone）ローカルブランチを返す。
func (c Client) GoneBranches() ([]string, error) {
	out, err := c.Output("for-each-ref", "--format=%(refname:short) %(upstream:track)", "refs/heads")
	if err != nil {
		return nil, err
	}
	return parseGoneBranches(out), nil
}

func parseGoneBranches(out string) []string {
	var branches []string
	for _, line := range splitLines(out) {
		name, track, found := strings.Cut(line, " ")
		if found && strings.TrimSpace(track) == "[gone]" {
			branches = append(branches, name)
		}
	}
	return branches
}

// BranchExists はローカルブランチの存在を確認する。
func (c Client) BranchExists(name string) bool {
	_, err := c.Output("show-ref", "--verify", "--quiet", "refs/heads/"+name)
	return err == nil
}

// FetchPrune は消えたリモート追跡ブランチを掃除する。
func (c Client) FetchPrune() error {
	return c.Run("fetch", "--prune", "--quiet")
}

// AddWorktree は既存ブランチの worktree を作成する。
func (c Client) AddWorktree(path, branch string) error {
	return c.Run("worktree", "add", path, branch)
}

// AddWorktreeNewBranch は新規ブランチを作って worktree を作成する。
// base が空ならカレントブランチを起点にする。
func (c Client) AddWorktreeNewBranch(path, branch, base string) error {
	args := []string{"worktree", "add", "-b", branch, path}
	if base != "" {
		args = append(args, base)
	}
	return c.Run(args...)
}

// RemoveWorktree は worktree を削除する。
func (c Client) RemoveWorktree(path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	return c.Run(args...)
}

// DeleteBranch はローカルブランチを強制削除する。
func (c Client) DeleteBranch(name string) error {
	_, err := c.Output("branch", "-D", name)
	return err
}

func splitLines(out string) []string {
	if out == "" {
		return nil
	}
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
