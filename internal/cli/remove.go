package cli

import (
	"errors"
	"fmt"
	"slices"

	"github.com/spf13/cobra"

	"github.com/Alesion30/gw/internal/git"
	"github.com/Alesion30/gw/internal/ui"
)

type removeOptions struct {
	force        bool
	gone         bool
	deleteBranch bool
}

func newRemoveCmd() *cobra.Command {
	var opts removeOptions

	cmd := &cobra.Command{
		Use:     "remove [query]",
		Aliases: []string{"rm"},
		Short:   "worktree を削除する",
		Long: "worktree を選んで削除する。\n" +
			"--gone を付けると、upstream が消えたブランチの worktree をまとめて削除する。",
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			e, err := newEnv()
			if err != nil {
				return err
			}
			return runRemove(e, firstArg(args), opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.force, "force", "f", false, "変更が残っていても削除する")
	flags.BoolVar(&opts.gone, "gone", false, "upstream が消えたブランチの worktree をすべて削除する")
	flags.BoolVar(&opts.deleteBranch, "delete-branch", false, "--gone のときにローカルブランチも削除する")

	return cmd
}

func runRemove(e *env, query string, opts removeOptions) error {
	if opts.gone {
		if query != "" {
			return errors.New("gw remove --gone does not take a query")
		}
		return removeGone(e, opts)
	}
	if opts.deleteBranch {
		return errors.New("--delete-branch is only available with --gone")
	}

	worktrees, err := e.git.Worktrees()
	if err != nil {
		return err
	}
	if len(worktrees) < 2 {
		return errors.New("no worktrees to remove")
	}

	// 先頭はメインの worktree なので削除候補から外す
	selected, err := selectWorktree(e, worktrees[1:], "remove> ", query)
	if err != nil {
		return err
	}

	if err := e.git.RemoveWorktree(selected.Path, opts.force); err != nil {
		return err
	}
	ui.Infof("Removed worktree at %s", selected.Path)

	return nil
}

// removeGone は upstream が消えた（gone）ブランチの worktree をまとめて削除する。
func removeGone(e *env, opts removeOptions) error {
	ui.Warnf("Fetching with --prune ...")
	if err := e.git.FetchPrune(); err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	gone, err := e.git.GoneBranches()
	if err != nil {
		return err
	}
	if len(gone) == 0 {
		ui.Infof("No branches with a gone upstream.")
		return nil
	}

	targets, err := goneTargets(e, gone)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		ui.Infof("No worktrees with a gone upstream.")
		return nil
	}

	ui.Infof("Worktrees with a gone upstream:")
	for _, wt := range targets {
		ui.Infof("  %s  [%s]", wt.Path, wt.Branch)
	}

	if !opts.force && !e.confirm(fmt.Sprintf("Remove %d worktree(s)?", len(targets))) {
		return errors.New("canceled")
	}

	// 1 つ失敗しても残りは処理する（未コミットの変更が残っているケースなど）
	for _, wt := range targets {
		if err := e.git.RemoveWorktree(wt.Path, opts.force); err != nil {
			ui.Warnf("Failed to remove %s. Skipped.", wt.Path)
			continue
		}
		ui.Infof("Removed worktree at %s", wt.Path)

		if opts.deleteBranch {
			if err := e.git.DeleteBranch(wt.Branch); err != nil {
				ui.Warnf("Failed to delete branch %q. Skipped.", wt.Branch)
				continue
			}
			ui.Infof("Deleted branch %q", wt.Branch)
		}
	}

	return nil
}

// goneTargets は gone ブランチに紐づく worktree のうち、削除してよいものを返す。
// メインの worktree とカレントの worktree は対象から外す。
func goneTargets(e *env, gone []string) ([]git.Worktree, error) {
	worktrees, err := e.git.Worktrees()
	if err != nil {
		return nil, err
	}
	if len(worktrees) < 2 {
		return nil, nil
	}

	var targets []git.Worktree
	for _, wt := range worktrees[1:] {
		if wt.Branch == "" || !slices.Contains(gone, wt.Branch) {
			continue
		}
		if wt.Path == e.cwd {
			ui.Warnf("Skipping the current worktree at %s [%s]", wt.Path, wt.Branch)
			continue
		}
		targets = append(targets, wt)
	}

	return targets, nil
}
