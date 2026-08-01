package cli

import (
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "list [git-worktree-list-options]",
		Aliases:            []string{"ls"},
		Short:              "List worktrees",
		Long:               "Pass the arguments through to `git worktree list`.",
		DisableFlagParsing: true,
		RunE: func(_ *cobra.Command, args []string) error {
			e, err := newEnv()
			if err != nil {
				return err
			}
			return e.git.Run(append([]string{"worktree", "list"}, args...)...)
		},
	}
}
