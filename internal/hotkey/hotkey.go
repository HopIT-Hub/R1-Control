package hotkey

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"golang.design/x/hotkey"
)

// Manager handles global hotkey registration with hold-to-talk support.
type Manager struct {
	mu     sync.Mutex
	hk     *hotkey.Hotkey
	cancel context.CancelFunc
	onDown func()
	onUp   func()
}

// NewManager creates a hotkey manager with callbacks for key-down and key-up.
func NewManager(onDown, onUp func()) *Manager {
	return &Manager{
		onDown: onDown,
		onUp:   onUp,
	}
}

// Register sets up a global hotkey with the given modifiers and key.
// If a hotkey is already registered, it is unregistered first.
func (m *Manager) Register(mods []string, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Unregister existing hotkey
	m.unregisterLocked()

	// Parse modifiers and key
	parsedMods, err := ParseModifiers(mods)
	if err != nil {
		return fmt.Errorf("parse modifiers: %w", err)
	}
	parsedKey, err := ParseKey(key)
	if err != nil {
		return fmt.Errorf("parse key: %w", err)
	}

	// Create and register the hotkey
	hk := hotkey.New(parsedMods, parsedKey)
	if err := hk.Register(); err != nil {
		return fmt.Errorf("register hotkey: %w", err)
	}

	m.hk = hk

	// Start listening for events
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	go m.listen(ctx, hk)

	log.Printf("[hotkey] registered: %v", mods)
	return nil
}

// listen loops on keydown/keyup channels and calls the callbacks.
func (m *Manager) listen(ctx context.Context, hk *hotkey.Hotkey) {
	// Linux X11 auto-repeat generates spurious keyup/keydown pairs.
	// Debounce: on keyup, wait 50ms; if keydown fires within that window,
	// treat it as auto-repeat and ignore both events.
	isLinux := runtime.GOOS == "linux"
	var debounceTimer *time.Timer

	for {
		select {
		case <-ctx.Done():
			return
		case <-hk.Keydown():
			if isLinux && debounceTimer != nil {
				// Cancel pending keyup — this is auto-repeat, not a real release
				debounceTimer.Stop()
				debounceTimer = nil
				continue
			}
			if m.onDown != nil {
				m.onDown()
			}
		case <-hk.Keyup():
			if isLinux {
				// Delay the keyup callback to check for auto-repeat
				debounceTimer = time.AfterFunc(50*time.Millisecond, func() {
					if m.onUp != nil {
						m.onUp()
					}
					m.mu.Lock()
					debounceTimer = nil
					m.mu.Unlock()
				})
			} else {
				if m.onUp != nil {
					m.onUp()
				}
			}
		}
	}
}

// Unregister removes the current global hotkey.
func (m *Manager) Unregister() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unregisterLocked()
}

func (m *Manager) unregisterLocked() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	if m.hk != nil {
		m.hk.Unregister()
		m.hk = nil
	}
}
