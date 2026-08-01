package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Alesion30/gw/internal/finder"
	"github.com/Alesion30/gw/internal/git"
)

func newPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path [query]",
		Short: "worktree を選択してパスを出力する",
		Long: "カレント以外の worktree を選んでパスを標準出力へ書く。\n" +
			"シェル関数から `cd \"$(gw path)\"` の形で使う。",
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			e, err := newEnv()
			if err != nil {
				return err
			}
			return runPath(e, firstArg(args))
		},
	}
}

func runPath(e *env, query string) error {
	worktrees, err := e.git.Worktrees()
	if err != nil {
		return err
	}

	var candidates []git.Worktree
	for _, wt := range worktrees {
		if wt.Path != e.cwd {
			candidates = append(candidates, wt)
		}
	}
	if len(candidates) == 0 {
		return errors.New("no other worktrees found")
	}

	selected, err := selectWorktree(e, candidates, "worktree> ", query)
	if err != nil {
		return err
	}

	fmt.Fprint(e.stdout, selected.Path)
	return nil
}

// selectWorktree は worktree 一覧から 1 つ選ばせる。
func selectWorktree(e *env, worktrees []git.Worktree, prompt, query string) (git.Worktree, error) {
	labels := make([]string, len(worktrees))
	for i, wt := range worktrees {
		labels[i] = worktreeLabel(wt)
	}

	res, err := e.finder.Find(labels, finder.Options{
		Prompt:    prompt,
		Query:     query,
		SelectOne: query != "",
	})
	if err != nil || res.Index < 0 {
		if err != nil && !errors.Is(err, finder.ErrAborted) {
			return git.Worktree{}, err
		}
		return git.Worktree{}, errors.New("no worktree selected")
	}

	return worktrees[res.Index], nil
}

func worktreeLabel(wt git.Worktree) string {
	switch {
	case wt.Bare:
		return wt.Path + "  (bare)"
	case wt.Branch != "":
		return fmt.Sprintf("%s  [%s]", wt.Path, wt.Branch)
	default:
		return fmt.Sprintf("%s  (%s detached)", wt.Path, shortHead(wt.Head))
	}
}

func shortHead(head string) string {
	if len(head) > 7 {
		return head[:7]
	}
	return head
}

func firstArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return ""
}
