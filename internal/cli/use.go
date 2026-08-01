package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"

	"github.com/spf13/cobra"

	"github.com/Alesion30/gw/internal/finder"
	"github.com/Alesion30/gw/internal/ui"
)

// setupScript は worktree 作成後に実行するスクリプトのファイル名。
const setupScript = ".gw-setup"

func newUseCmd() *cobra.Command {
	var base string

	cmd := &cobra.Command{
		Use:   "use [query]",
		Short: "ブランチを選んで worktree を作成する",
		Long: "worktree 化していないローカルブランチから選んで worktree を作る。\n" +
			"どれにもマッチしない文字列で確定すると、その名前で新しいブランチを作るか確認する。\n" +
			"作成先は $GW_WORKTREE_DIR（既定: <repo-root>/.worktrees）配下。\n" +
			"リポジトリルートに ." + "gw-setup があれば、作成した worktree 内で実行する。",
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			e, err := newEnv()
			if err != nil {
				return err
			}
			return runUse(e, firstArg(args), base)
		},
	}

	cmd.Flags().StringVar(&base, "base", "", "新規ブランチの起点（既定: カレントブランチ）")

	return cmd
}

func runUse(e *env, query, base string) error {
	candidates, err := availableBranches(e)
	if err != nil {
		return err
	}

	branch, typed := "", query

	if len(candidates) > 0 {
		res, err := e.finder.Find(candidates, finder.Options{
			Prompt:       "branch> ",
			Query:        query,
			AllowNoMatch: true,
		})
		if err != nil {
			if errors.Is(err, finder.ErrAborted) {
				return errors.New("canceled")
			}
			return err
		}

		typed = res.Query
		if res.Index >= 0 {
			branch = candidates[res.Index]
		}
	}

	// 既存ブランチが選ばれなかったときは、入力文字列で新規ブランチを作るか確認する
	newBranch := ""
	if branch == "" {
		if typed == "" {
			return errors.New("no branch selected")
		}
		if e.git.BranchExists(typed) {
			return fmt.Errorf("branch %q is already checked out in another worktree", typed)
		}
		if !e.confirm(fmt.Sprintf("Branch %q does not exist. Create it?", typed)) {
			return errors.New("canceled")
		}
		newBranch, branch = typed, typed
	}

	root, err := e.git.RepoRoot()
	if err != nil {
		return err
	}

	baseDir := os.Getenv("GW_WORKTREE_DIR")
	if baseDir == "" {
		baseDir = filepath.Join(root, ".worktrees")
	}
	wtPath := filepath.Join(baseDir, branch)

	if newBranch != "" {
		if err := e.git.AddWorktreeNewBranch(wtPath, branch, base); err != nil {
			return err
		}
		ui.Infof("Created worktree for new branch %q at %s", branch, wtPath)
	} else {
		if err := e.git.AddWorktree(wtPath, branch); err != nil {
			return err
		}
		ui.Infof("Created worktree for branch %q at %s", branch, wtPath)
	}

	return runSetup(root, wtPath)
}

// availableBranches は worktree 化していないローカルブランチを返す。
func availableBranches(e *env) ([]string, error) {
	worktrees, err := e.git.Worktrees()
	if err != nil {
		return nil, err
	}

	inUse := make(map[string]bool, len(worktrees))
	for _, wt := range worktrees {
		if wt.Branch != "" {
			inUse[wt.Branch] = true
		}
	}

	branches, err := e.git.LocalBranches()
	if err != nil {
		return nil, err
	}

	return slices.DeleteFunc(branches, func(b string) bool { return inUse[b] }), nil
}

// runSetup はリポジトリルートの .gw-setup を、作成した worktree 内で実行する。
func runSetup(root, wtPath string) error {
	script := filepath.Join(root, setupScript)
	if _, err := os.Stat(script); err != nil {
		return nil
	}

	ui.Infof("Running %s in %s ...", script, wtPath)

	cmd := exec.Command("sh", script)
	cmd.Dir = wtPath
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
