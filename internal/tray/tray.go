// Package tray manages the system tray icon and menu.
package tray

import (
	"strings"

	"github.com/HopIT-Hub/R1-Control/internal/device"

	"fyne.io/systray"
)

// RunOpts configures the system tray.
type RunOpts struct {
	Version          string // app version string (e.g., "1.0.0")
	AutoStartEnabled bool   // initial state of "Start on Login" checkbox
	KeepAwakeEnabled bool   // initial state of "Keep Awake" checkbox
	OnReady          func()
	OnSettings       func()
	OnAutoStart      func(enabled bool) // called when user toggles auto-start
	OnKeepAwake      func(enabled bool) // called when user toggles keep-awake
	OnQuit           func()
}

// Run starts the system tray. It blocks on the main thread.
func Run(opts RunOpts) {
	systray.Run(func() {
		systray.SetIcon(IconDisconnected)
		systray.SetTitle("")
		systray.SetTooltip("R1 Control — No device")

		// Version label (disabled — just informational)
		versionLabel := "R1 Control"
		if opts.Version != "" && opts.Version != "dev" {
			versionLabel += " v" + strings.TrimPrefix(opts.Version, "v")
		}
		mVersion := systray.AddMenuItem(versionLabel, "")
		mVersion.Disable()

		systray.AddSeparator()

		mSettings := systray.AddMenuItem("Settings...", "Configure hotkeys")
		mAutoStart := systray.AddMenuItemCheckbox("Start on Login", "Launch automatically on login", opts.AutoStartEnabled)
		mKeepAwake := systray.AddMenuItemCheckbox("Keep Awake", "Prevent R1 from sleeping while docked", opts.KeepAwakeEnabled)

		systray.AddSeparator()

		mStatus := systray.AddMenuItem("Status: Disconnected", "")
		mStatus.Disable()

		systray.AddSeparator()

		mQuit := systray.AddMenuItem("Quit", "Exit R1 Control")

		// Store status item for updates
		statusItem = mStatus

		if opts.OnReady != nil {
			opts.OnReady()
		}

		go func() {
			for {
				select {
				case <-mSettings.ClickedCh:
					if opts.OnSettings != nil {
						opts.OnSettings()
					}
				case <-mAutoStart.ClickedCh:
					if mAutoStart.Checked() {
						mAutoStart.Uncheck()
						if opts.OnAutoStart != nil {
							opts.OnAutoStart(false)
						}
					} else {
						mAutoStart.Check()
						if opts.OnAutoStart != nil {
							opts.OnAutoStart(true)
						}
					}
				case <-mKeepAwake.ClickedCh:
					if mKeepAwake.Checked() {
						mKeepAwake.Uncheck()
						if opts.OnKeepAwake != nil {
							opts.OnKeepAwake(false)
						}
					} else {
						mKeepAwake.Check()
						if opts.OnKeepAwake != nil {
							opts.OnKeepAwake(true)
						}
					}
				case <-mQuit.ClickedCh:
					if opts.OnQuit != nil {
						opts.OnQuit()
					}
					systray.Quit()
				}
			}
		}()
	}, func() {
		// cleanup on systray exit
	})
}

var statusItem *systray.MenuItem

// SetState updates the tray icon and tooltip based on device state.
func SetState(state device.State) {
	switch state {
	case device.Disconnected:
		systray.SetIcon(IconDisconnected)
		systray.SetTooltip("R1 Control — No device")
		if statusItem != nil {
			statusItem.SetTitle("Status: Disconnected")
		}
	case device.Connected:
		systray.SetIcon(IconConnected)
		systray.SetTooltip("R1 Control — Ready")
		if statusItem != nil {
			statusItem.SetTitle("Status: Connected")
		}
	case device.PTTActive:
		systray.SetIcon(IconActive)
		systray.SetTooltip("R1 Control — TALKING")
		if statusItem != nil {
			statusItem.SetTitle("Status: PTT Active")
		}
	}
}

// Quit stops the system tray.
func Quit() {
	systray.Quit()
}
