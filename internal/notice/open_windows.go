//go:build windows

package notice

import (
	"errors"
	"os/exec"
)

// OpenApp activates the app behind an AUMID. Windows has no plain API for
// this; the AppsFolder shell namespace resolves packaged app ids.
func OpenApp(aumid string) error {
	if aumid == "" {
		return errors.New("no app to open")
	}
	return exec.Command("explorer.exe", "shell:AppsFolder\\"+aumid).Run()
}
