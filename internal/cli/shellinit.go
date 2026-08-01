package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// shellInits は `gw cd` を成立させるためのシェル別ラッパー定義。
// gw は子プロセスなので親シェルの cd はできない。gw path が出力したパスへ
// シェル側で移動する関数をかぶせる。
var shellInits = map[string]string{
	"zsh": `gw() {
  if [ "$1" = "cd" ]; then
    local selected
    shift
    selected=$(command gw path "$@") || return
    [ -n "$selected" ] && cd "$selected"
  else
    command gw "$@"
  fi
}`,
	"bash": `gw() {
  if [ "$1" = "cd" ]; then
    local selected
    shift
    selected=$(command gw path "$@") || return
    [ -n "$selected" ] && cd "$selected"
  else
    command gw "$@"
  fi
}`,
	"fish": `function gw
  if test "$argv[1]" = "cd"
    set -l selected (command gw path $argv[2..])
    or return
    test -n "$selected"; and cd "$selected"
  else
    command gw $argv
  end
end`,
}

func newShellInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "shell-init <zsh|bash|fish>",
		Short: "gw cd 用のシェル関数を出力する",
		Long: "gw cd を使うためのシェル関数を出力する。\n" +
			"zsh なら .zshrc に `eval \"$(gw shell-init zsh)\"` を書く。",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"zsh", "bash", "fish"},
		RunE: func(cmd *cobra.Command, args []string) error {
			script, ok := shellInits[args[0]]
			if !ok {
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
			cmd.Println(script)
			return nil
		},
	}
}
