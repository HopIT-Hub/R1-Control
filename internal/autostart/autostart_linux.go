//go:build linux

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

const desktopFileName = "r1control.desktop"

const desktopEntryTemplate = `[Desktop Entry]
Type=Application
Name=R1 Control
Comment=Control your Rabbit R1 over USB
Exec=%s
Icon=r1control
Categories=Utility;
Terminal=false
X-GNOME-Autostart-enabled=true
`

func desktopFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	return filepath.Join(configDir, "autostart", desktopFileName), nil
}

// IsEnabled returns true if the autostart .desktop file exists.
func IsEnabled() bool {
	p, err := desktopFilePath()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// Enable creates an autostart .desktop entry for the current executable.
func Enable() error {
	exe, err := appPath()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	p, err := desktopFilePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("create autostart dir: %w", err)
	}

	content := fmt.Sprintf(desktopEntryTemplate, exe)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write desktop file: %w", err)
	}

	return nil
}

// Disable removes the autostart .desktop entry.
func Disable() error {
	p, err := desktopFilePath()
	if err != nil {
		return err
	}

	err = os.Remove(p)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
