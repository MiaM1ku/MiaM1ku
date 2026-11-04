//go:build !windows

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "miam1ku-status 是 Windows 程序：请用 build.sh 交叉编译出 dist/miam1ku-status.exe，在 Windows 上运行。")
	os.Exit(1)
}
