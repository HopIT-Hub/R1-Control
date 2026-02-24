// Package config handles loading and saving the R1 PTT configuration.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Config holds the application configuration.
type Config struct {
	mu                sync.RWMutex `json:"-"`
	Hotkey            HotkeyConfig `json:"hotkey"`
	SwipeHotkey       HotkeyConfig `json:"swipe_hotkey"`
	AutoStart         bool         `json:"auto_start"`
	KeepAwake         bool         `json:"keep_awake"`
	SleepAfterMinutes int          `json:"sleep_after_minutes"`
}

// HotkeyConfig defines a global hotkey binding.
type HotkeyConfig struct {
	Modifiers []string `json:"modifiers"` // "ctrl", "shift", "alt", "super"
	Key       string   `json:"key"`       // "r", "space", "f5", etc.
}

// String returns a human-readable representation like "Ctrl+Alt+R".
func (h HotkeyConfig) String() string {
	s := ""
	for _, m := range h.Modifiers {
		switch m {
		case "ctrl":
			s += "Ctrl+"
		case "shift":
			s += "Shift+"
		case "alt":
			s += "Alt+"
		case "super":
			s += "Super+"
		}
	}
	if len(h.Key) == 1 {
		s += string(h.Key[0] - 32) // uppercase single letter
	} else {
		s += h.Key
	}
	return s
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Hotkey: HotkeyConfig{
			Modifiers: []string{"ctrl", "alt"},
			Key:       "r",
		},
		SwipeHotkey: HotkeyConfig{
			Modifiers: []string{"ctrl", "alt"},
			Key:       "w",
		},
		KeepAwake:         true,
		SleepAfterMinutes: 60,
	}
}

// Dir returns the OS-appropriate config directory for r1ptt.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	return filepath.Join(base, "r1ptt"), nil
}

// Path returns the full path to the config file.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads the config from disk. If the file doesn't exist, it creates
// a default config and saves it.
func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		cfg := DefaultConfig()
		if saveErr := cfg.Save(); saveErr != nil {
			return nil, fmt.Errorf("create default config: %w", saveErr)
		}
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := DefaultConfig() // start with defaults so new fields get populated
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// Save writes the config to disk atomically (write temp, rename).
func (c *Config) Save() error {
	c.mu.RLock()
	data, err := json.MarshalIndent(c, "", "  ")
	c.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	p, err := Path()
	if err != nil {
		return err
	}

	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := os.Rename(tmp, p); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename config: %w", err)
	}
	return nil
}

// SetHotkey updates the PTT hotkey configuration and saves to disk.
func (c *Config) SetHotkey(mods []string, key string) error {
	c.mu.Lock()
	c.Hotkey = HotkeyConfig{Modifiers: mods, Key: key}
	c.mu.Unlock()
	return c.Save()
}

// GetHotkey returns a copy of the current PTT hotkey configuration.
func (c *Config) GetHotkey() HotkeyConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	mods := make([]string, len(c.Hotkey.Modifiers))
	copy(mods, c.Hotkey.Modifiers)
	return HotkeyConfig{Modifiers: mods, Key: c.Hotkey.Key}
}

// SetSwipeHotkey updates the swipe hotkey configuration and saves to disk.
func (c *Config) SetSwipeHotkey(mods []string, key string) error {
	c.mu.Lock()
	c.SwipeHotkey = HotkeyConfig{Modifiers: mods, Key: key}
	c.mu.Unlock()
	return c.Save()
}

// GetSwipeHotkey returns a copy of the current swipe hotkey configuration.
func (c *Config) GetSwipeHotkey() HotkeyConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	mods := make([]string, len(c.SwipeHotkey.Modifiers))
	copy(mods, c.SwipeHotkey.Modifiers)
	return HotkeyConfig{Modifiers: mods, Key: c.SwipeHotkey.Key}
}

// GetAutoStart returns the current auto-start setting.
func (c *Config) GetAutoStart() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.AutoStart
}

// SetAutoStart updates the auto-start setting and saves to disk.
func (c *Config) SetAutoStart(enabled bool) error {
	c.mu.Lock()
	c.AutoStart = enabled
	c.mu.Unlock()
	return c.Save()
}

// GetKeepAwake returns the current keep-awake setting.
func (c *Config) GetKeepAwake() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.KeepAwake
}

// GetSleepAfterMinutes returns the current sleep-after-idle duration.
func (c *Config) GetSleepAfterMinutes() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.SleepAfterMinutes
}

// SetKeepAwake updates the keep-awake setting and saves to disk.
func (c *Config) SetKeepAwake(enabled bool, sleepAfterMinutes int) error {
	c.mu.Lock()
	c.KeepAwake = enabled
	c.SleepAfterMinutes = sleepAfterMinutes
	c.mu.Unlock()
	return c.Save()
}
