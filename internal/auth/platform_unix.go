//go:build !windows

package auth

import (
	"os/exec"
	"runtime"
)

func OpenBrowser(url string) error {
	if runtime.GOOS == "darwin" {
		return exec.Command("open", url).Run()
	}
	return exec.Command("xdg-open", url).Run()
}
