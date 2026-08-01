// Package ui は対話プロンプトとメッセージ出力を提供する。
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Confirm は yes/no を対話で確認する。
// 端末を開けない場合（パイプ経由の実行など）は no 扱いにする。
func Confirm(prompt string) bool {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	defer func() { _ = tty.Close() }()

	fmt.Fprintf(os.Stderr, "%s [y/N] ", prompt)

	line, err := bufio.NewReader(tty).ReadString('\n')
	if err != nil && line == "" {
		fmt.Fprintln(os.Stderr)
		return false
	}

	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}

// Infof は進捗メッセージを標準出力へ書く。
func Infof(format string, args ...any) {
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}

// Warnf は警告・エラーメッセージを標準エラー出力へ書く。
func Warnf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}
