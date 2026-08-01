package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Alesion30/gw/internal/ui"
)

func newCopyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "copy <file> [file ...]",
		Short: "メインの worktree からファイルをコピーする",
		Long: "メインの worktree から現在の worktree へファイルをコピーする。\n" +
			".env のように gitignore していて worktree に持ち越されないファイル用。",
		Args:    cobra.MinimumNArgs(1),
		Example: "  gw copy .env .envrc.local",
		RunE: func(_ *cobra.Command, args []string) error {
			e, err := newEnv()
			if err != nil {
				return err
			}
			return runCopy(e, args)
		},
	}
}

func runCopy(e *env, files []string) error {
	worktrees, err := e.git.Worktrees()
	if err != nil {
		return err
	}
	if len(worktrees) == 0 {
		return errors.New("main worktree not found")
	}
	main := worktrees[0].Path

	current, err := e.git.RepoRoot()
	if err != nil {
		return err
	}
	if current == main {
		return fmt.Errorf("cannot run from the main worktree (%s)", main)
	}

	ui.Infof("Main:    %s", main)
	ui.Infof("Current: %s", current)
	ui.Infof("")

	failed := false
	for _, file := range files {
		src := filepath.Join(main, file)
		dst := filepath.Join(current, file)

		if err := copyFile(src, dst); err != nil {
			ui.Warnf("  ⚠ %s (%v)", file, err)
			failed = true
			continue
		}
		ui.Infof("  ✓ %s", file)
	}

	if failed {
		return errSilent
	}
	return nil
}

// copyFile は src を dst へコピーする。パーミッションは src に合わせる。
func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return errors.New("not found in main")
	}
	if !info.Mode().IsRegular() {
		return errors.New("not a regular file")
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
