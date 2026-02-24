//go:build windows

package autostart

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

const (
	regKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	regValName = "R1Control"
)

// IsEnabled returns true if the auto-start registry key exists.
func IsEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, regKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	_, _, err = k.GetStringValue(regValName)
	return err == nil
}

// Enable adds an auto-start registry entry for the current executable.
func Enable() error {
	exe, err := appPath()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	k, err := registry.OpenKey(registry.CURRENT_USER, regKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open registry key: %w", err)
	}
	defer k.Close()

	if err := k.SetStringValue(regValName, exe); err != nil {
		return fmt.Errorf("set registry value: %w", err)
	}

	return nil
}

// Disable removes the auto-start registry entry.
func Disable() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, regKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open registry key: %w", err)
	}
	defer k.Close()

	err = k.DeleteValue(regValName)
	if err == registry.ErrNotExist {
		return nil
	}
	return err
}
