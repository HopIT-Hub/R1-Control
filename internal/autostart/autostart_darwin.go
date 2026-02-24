//go:build darwin

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const (
	launchAgentLabel = "co.hopit.r1control"
	launchAgentFile  = "co.hopit.r1control.plist"
)

var plistTemplate = template.Must(template.New("plist").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{ .Label }}</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{ .Program }}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
</dict>
</plist>
`))

func plistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("user home dir: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents", launchAgentFile), nil
}

// IsEnabled returns true if the LaunchAgent plist exists.
func IsEnabled() bool {
	p, err := plistPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// Enable creates a LaunchAgent plist so the app starts on login.
func Enable() error {
	exe, err := appPath()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	p, err := plistPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}

	f, err := os.Create(p)
	if err != nil {
		return fmt.Errorf("create plist: %w", err)
	}
	defer f.Close()

	data := struct {
		Label   string
		Program string
	}{
		Label:   launchAgentLabel,
		Program: exe,
	}

	if err := plistTemplate.Execute(f, data); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}

	return nil
}

// Disable removes the LaunchAgent plist.
func Disable() error {
	p, err := plistPath()
	if err != nil {
		return err
	}

	err = os.Remove(p)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
