// Command gw は git worktree のラッパーコマンド。
package main

import (
	"os"

	"github.com/Alesion30/gw/internal/cli"
)

// version はリリースビルド時に ldflags で埋め込む。
var version = "dev"

func main() {
	os.Exit(cli.Execute(version))
}
