// Package hotkey provides cross-platform global hotkey registration
// with hold-to-talk (key-down and key-up) support.
package hotkey

import (
	"fmt"
	"strings"

	"golang.design/x/hotkey"
)

// ParseModifiers converts string modifier names to hotkey.Modifier values.
// The modMap variable is defined in platform-specific files (keymap_*.go).
func ParseModifiers(names []string) ([]hotkey.Modifier, error) {
	var mods []hotkey.Modifier
	for _, name := range names {
		m, ok := modMap[strings.ToLower(name)]
		if !ok {
			return nil, fmt.Errorf("unknown modifier: %q (available: ctrl, shift, alt, super)", name)
		}
		mods = append(mods, m)
	}
	return mods, nil
}

// ParseKey converts a string key name to a hotkey.Key value.
// The keyMap variable is defined in platform-specific files (keymap_*.go).
func ParseKey(name string) (hotkey.Key, error) {
	k, ok := keyMap[strings.ToLower(name)]
	if !ok {
		return 0, fmt.Errorf("unknown key: %q", name)
	}
	return k, nil
}

// JSCodeToKeyName converts a JavaScript event.code to our config key name.
// e.g., "KeyR" → "r", "F5" → "f5", "Space" → "space"
func JSCodeToKeyName(jsCode string) (string, error) {
	name, ok := jsCodeToName[jsCode]
	if !ok {
		return "", fmt.Errorf("unsupported key code: %q", jsCode)
	}
	return name, nil
}

var jsCodeToName = map[string]string{
	"KeyA": "a", "KeyB": "b", "KeyC": "c", "KeyD": "d",
	"KeyE": "e", "KeyF": "f", "KeyG": "g", "KeyH": "h",
	"KeyI": "i", "KeyJ": "j", "KeyK": "k", "KeyL": "l",
	"KeyM": "m", "KeyN": "n", "KeyO": "o", "KeyP": "p",
	"KeyQ": "q", "KeyR": "r", "KeyS": "s", "KeyT": "t",
	"KeyU": "u", "KeyV": "v", "KeyW": "w", "KeyX": "x",
	"KeyY": "y", "KeyZ": "z",
	"Digit0": "0", "Digit1": "1", "Digit2": "2", "Digit3": "3",
	"Digit4": "4", "Digit5": "5", "Digit6": "6", "Digit7": "7",
	"Digit8": "8", "Digit9": "9",
	"F1": "f1", "F2": "f2", "F3": "f3", "F4": "f4",
	"F5": "f5", "F6": "f6", "F7": "f7", "F8": "f8",
	"F9": "f9", "F10": "f10", "F11": "f11", "F12": "f12",
	"F13": "f13", "F14": "f14", "F15": "f15", "F16": "f16",
	"F17": "f17", "F18": "f18", "F19": "f19", "F20": "f20",
	"Space": "space", "Enter": "return", "Escape": "escape",
	"Backspace": "delete", "Tab": "tab",
	"ArrowUp": "up", "ArrowDown": "down",
	"ArrowLeft": "left", "ArrowRight": "right",
}
