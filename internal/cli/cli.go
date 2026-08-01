// Package cli は gw のコマンド定義を持つ。
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Alesion30/gw/internal/finder"
	"github.com/Alesion30/gw/internal/git"
	"github.com/Alesion30/gw/internal/ui"
)

// env はコマンド間で共有する実行環境。
// テストでは finder と confirm を差し替えて対話を排除する。
type env struct {
	git     git.Client
	finder  finder.Finder
	confirm func(prompt string) bool
	stdout  io.Writer
	cwd     string
}

func newEnv() (*env, error) {
	cwd, err := currentDir()
	if err != nil {
		return nil, err
	}
	return &env{
		git:     git.Client{Dir: cwd},
		finder:  finder.TUI{},
		confirm: ui.Confirm,
		stdout:  os.Stdout,
		cwd:     cwd,
	}, nil
}

// currentDir は symlink を解決したカレントディレクトリを返す（`pwd -P` 相当）。
// worktree のパスと文字列比較するため、両者の表現を揃えておく必要がある。
func currentDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(cwd); err == nil {
		return resolved, nil
	}
	return cwd, nil
}

// NewRootCmd は gw のルートコマンドを組み立てる。
func NewRootCmd(version string) *cobra.Command {
	root := &cobra.Command{
		Use:           "gw",
		Short:         "A wrapper around git worktree",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_ = cmd.Help()
			return errSilent
		},
	}

	root.SetVersionTemplate("gw {{.Version}}\n")
	root.AddCommand(
		newPathCmd(),
		newUseCmd(),
		newRemoveCmd(),
		newListCmd(),
		newCopyCmd(),
		newShellInitCmd(),
	)

	return root
}

// errSilent はメッセージを出さずに終了コード 1 を返すための番兵。
// 呼び出し側が already 出力を済ませているケースで使う。
var errSilent = errors.New("silent")

// Execute はルートコマンドを実行し、プロセスの終了コードを返す。
func Execute(version string) int {
	if err := NewRootCmd(version).Execute(); err != nil {
		if !errors.Is(err, errSilent) {
			fmt.Fprintln(os.Stderr, "Error: "+err.Error())
		}
		return 1
	}
	return 0
}
