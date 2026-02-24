// R1 Control — Rabbit R1 system tray application.
//
// Cross-platform tray app that controls the R1 over USB-C.
//
// PTT hotkey (default: Ctrl+Alt+R):
//   - Short press: toggle PTT on/off
//   - Hold: PTT active until release
//
// Swipe hotkey (default: Ctrl+Alt+W):
//   - Each press alternates between swipe left and swipe right
package main

import (
	"context"
	"log"
	"os/exec"
	"runtime"

	"github.com/HopIT-Hub/R1-Control/internal/autostart"
	"github.com/HopIT-Hub/R1-Control/internal/config"
	"github.com/HopIT-Hub/R1-Control/internal/device"
	"github.com/HopIT-Hub/R1-Control/internal/hotkey"
	"github.com/HopIT-Hub/R1-Control/internal/server"
	"github.com/HopIT-Hub/R1-Control/internal/tray"
)

var version = "dev"

func main() {
	// Load or create config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[r1control] config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Device manager — auto-detects R1, reconnects on disconnect
	devMgr := device.NewManager("", func(state device.State) {
		tray.SetState(state)
		log.Printf("[r1control] device: %s", state)
	})

	// Apply keep-awake settings from config
	devMgr.SetKeepAwake(cfg.GetKeepAwake(), cfg.GetSleepAfterMinutes())

	// PTT hotkey manager — toggle/hold-to-talk
	pttHkMgr := hotkey.NewManager(
		func() {
			if err := devMgr.PTTDown(); err != nil {
				log.Printf("[r1control] PTT down error: %v", err)
			} else {
				log.Println("[r1control] PTT ON")
			}
		},
		func() {
			if err := devMgr.PTTUp(); err != nil {
				log.Printf("[r1control] PTT up error: %v", err)
			} else {
				log.Println("[r1control] PTT OFF")
			}
		},
	)

	// Swipe hotkey manager — alternating left/right on each press
	swipeHkMgr := hotkey.NewManager(
		func() {
			if err := devMgr.Swipe(); err != nil {
				log.Printf("[r1control] swipe error: %v", err)
			}
		},
		nil, // no keyup action needed for swipe
	)

	// Settings HTTP server
	srv := server.New(pttHkMgr, swipeHkMgr, devMgr, cfg, version)

	// System tray — blocks on main thread
	tray.Run(tray.RunOpts{
		Version:          version,
		AutoStartEnabled: cfg.GetAutoStart(),
		KeepAwakeEnabled: cfg.GetKeepAwake(),

		// onReady — start background services after tray is initialized
		OnReady: func() {
			// Start device manager
			go devMgr.Run(ctx)

			// Register PTT hotkey
			hk := cfg.GetHotkey()
			if err := pttHkMgr.Register(hk.Modifiers, hk.Key); err != nil {
				log.Printf("[r1control] PTT hotkey register failed: %v", err)
				log.Printf("[r1control] you can change the hotkey via Settings")
			} else {
				log.Printf("[r1control] PTT hotkey: %s (short press=toggle, hold=talk)", hk.String())
			}

			// Register Swipe hotkey
			shk := cfg.GetSwipeHotkey()
			if err := swipeHkMgr.Register(shk.Modifiers, shk.Key); err != nil {
				log.Printf("[r1control] swipe hotkey register failed: %v", err)
			} else {
				log.Printf("[r1control] swipe hotkey: %s (alternates left/right)", shk.String())
			}

			// Start settings server
			if _, err := srv.Start(); err != nil {
				log.Printf("[r1control] settings server: %v", err)
			}

			log.Printf("[r1control] ready (version %s)", version)
		},

		// onSettings — open browser to settings page
		OnSettings: func() {
			url := srv.URL()
			if url == "" {
				log.Println("[r1control] settings server not running")
				return
			}
			openBrowser(url)
		},

		// onAutoStart — toggle auto-start on login
		OnAutoStart: func(enabled bool) {
			if enabled {
				if err := autostart.Enable(); err != nil {
					log.Printf("[r1control] enable autostart: %v", err)
					return
				}
			} else {
				if err := autostart.Disable(); err != nil {
					log.Printf("[r1control] disable autostart: %v", err)
					return
				}
			}
			if err := cfg.SetAutoStart(enabled); err != nil {
				log.Printf("[r1control] save autostart config: %v", err)
			}
			log.Printf("[r1control] auto-start: %v", enabled)
		},

		// onKeepAwake — toggle keep-awake
		OnKeepAwake: func(enabled bool) {
			if err := cfg.SetKeepAwake(enabled, cfg.GetSleepAfterMinutes()); err != nil {
				log.Printf("[r1control] save keep-awake config: %v", err)
			}
			devMgr.SetKeepAwake(enabled, cfg.GetSleepAfterMinutes())
			log.Printf("[r1control] keep-awake: %v", enabled)
		},

		// onQuit — clean shutdown
		OnQuit: func() {
			cancel()
			pttHkMgr.Unregister()
			swipeHkMgr.Unregister()
			devMgr.Close()
			srv.Stop()
		},
	})
}

func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default: // linux, bsd
		cmd = "xdg-open"
		args = []string{url}
	}

	if err := exec.Command(cmd, args...).Start(); err != nil {
		log.Printf("[r1control] open browser: %v", err)
	}
}
