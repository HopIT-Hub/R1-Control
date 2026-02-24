package server

import (
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"net/http"

	"github.com/HopIT-Hub/R1-Control/internal/autostart"
	"github.com/HopIT-Hub/R1-Control/internal/hotkey"
	"github.com/HopIT-Hub/R1-Control/internal/web"
)

// handleIndex serves the settings page HTML.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	staticFS, _ := fs.Sub(web.StaticFiles, "static")
	f, err := staticFS.Open("index.html")
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.Copy(w, f)
}

// statusResponse is the JSON response for GET /status.
type statusResponse struct {
	State             string `json:"state"`
	Hotkey            string `json:"hotkey"`
	SwipeHotkey       string `json:"swipe_hotkey"`
	Version           string `json:"version"`
	AutoStart         bool   `json:"auto_start"`
	KeepAwake         bool   `json:"keep_awake"`
	SleepAfterMinutes int    `json:"sleep_after_minutes"`
}

// handleStatus returns the current device state and hotkey config.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", 405)
		return
	}

	hk := s.cfg.GetHotkey()
	shk := s.cfg.GetSwipeHotkey()

	resp := statusResponse{
		State:             s.deviceMgr.State().String(),
		Hotkey:            hk.String(),
		SwipeHotkey:       shk.String(),
		Version:           s.version,
		AutoStart:         s.cfg.GetAutoStart(),
		KeepAwake:         s.cfg.GetKeepAwake(),
		SleepAfterMinutes: s.cfg.GetSleepAfterMinutes(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// hotkeyRequest is the JSON body for POST /hotkey.
type hotkeyRequest struct {
	Modifiers []string `json:"modifiers"`
	JSCode    string   `json:"js_code"`
}

// hotkeyResponse is the JSON response for POST /hotkey.
type hotkeyResponse struct {
	Hotkey string `json:"hotkey,omitempty"`
	Error  string `json:"error,omitempty"`
}

// handleHotkey updates the hotkey configuration.
func (s *Server) handleHotkey(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}

	var req hotkeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, hotkeyResponse{Error: "invalid JSON"})
		return
	}

	// Validate modifiers
	if len(req.Modifiers) == 0 {
		writeJSON(w, hotkeyResponse{Error: "at least one modifier required"})
		return
	}

	// Convert JS code to our key name
	keyName, err := hotkey.JSCodeToKeyName(req.JSCode)
	if err != nil {
		writeJSON(w, hotkeyResponse{Error: "unsupported key: " + req.JSCode})
		return
	}

	// Try to register the new hotkey
	if err := s.hotkeyMgr.Register(req.Modifiers, keyName); err != nil {
		log.Printf("[server] hotkey register failed: %v", err)
		writeJSON(w, hotkeyResponse{Error: "failed to register hotkey: " + err.Error()})
		return
	}

	// Save to config
	if err := s.cfg.SetHotkey(req.Modifiers, keyName); err != nil {
		log.Printf("[server] config save failed: %v", err)
		writeJSON(w, hotkeyResponse{Error: "saved hotkey but failed to persist config"})
		return
	}

	hk := s.cfg.GetHotkey()
	log.Printf("[server] hotkey updated to: %s", hk.String())
	writeJSON(w, hotkeyResponse{Hotkey: hk.String()})
}

// handleSwipeHotkey updates the swipe hotkey configuration.
func (s *Server) handleSwipeHotkey(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}

	var req hotkeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, hotkeyResponse{Error: "invalid JSON"})
		return
	}

	// Validate modifiers
	if len(req.Modifiers) == 0 {
		writeJSON(w, hotkeyResponse{Error: "at least one modifier required"})
		return
	}

	// Convert JS code to our key name
	keyName, err := hotkey.JSCodeToKeyName(req.JSCode)
	if err != nil {
		writeJSON(w, hotkeyResponse{Error: "unsupported key: " + req.JSCode})
		return
	}

	// Try to register the new hotkey
	if err := s.swipeHkMgr.Register(req.Modifiers, keyName); err != nil {
		log.Printf("[server] swipe hotkey register failed: %v", err)
		writeJSON(w, hotkeyResponse{Error: "failed to register hotkey: " + err.Error()})
		return
	}

	// Save to config
	if err := s.cfg.SetSwipeHotkey(req.Modifiers, keyName); err != nil {
		log.Printf("[server] config save failed: %v", err)
		writeJSON(w, hotkeyResponse{Error: "saved hotkey but failed to persist config"})
		return
	}

	shk := s.cfg.GetSwipeHotkey()
	log.Printf("[server] swipe hotkey updated to: %s", shk.String())
	writeJSON(w, hotkeyResponse{Hotkey: shk.String()})
}

// autoStartRequest is the JSON body for POST /autostart.
type autoStartRequest struct {
	Enabled bool `json:"enabled"`
}

// autoStartResponse is the JSON response for POST /autostart.
type autoStartResponse struct {
	AutoStart bool   `json:"auto_start"`
	Error     string `json:"error,omitempty"`
}

// handleAutoStart toggles the auto-start on login setting.
func (s *Server) handleAutoStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}

	var req autoStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, autoStartResponse{Error: "invalid JSON"})
		return
	}

	// Enable or disable OS autostart
	if req.Enabled {
		if err := autostart.Enable(); err != nil {
			log.Printf("[server] enable autostart: %v", err)
			writeJSON(w, autoStartResponse{Error: "failed to enable auto-start: " + err.Error()})
			return
		}
	} else {
		if err := autostart.Disable(); err != nil {
			log.Printf("[server] disable autostart: %v", err)
			writeJSON(w, autoStartResponse{Error: "failed to disable auto-start: " + err.Error()})
			return
		}
	}

	// Persist to config
	if err := s.cfg.SetAutoStart(req.Enabled); err != nil {
		log.Printf("[server] save autostart config: %v", err)
		writeJSON(w, autoStartResponse{Error: "setting changed but failed to persist"})
		return
	}

	log.Printf("[server] auto-start: %v", req.Enabled)
	writeJSON(w, autoStartResponse{AutoStart: req.Enabled})
}

// keepAwakeRequest is the JSON body for POST /keepawake.
type keepAwakeRequest struct {
	Enabled           bool `json:"enabled"`
	SleepAfterMinutes int  `json:"sleep_after_minutes"`
}

// keepAwakeResponse is the JSON response for POST /keepawake.
type keepAwakeResponse struct {
	KeepAwake         bool   `json:"keep_awake"`
	SleepAfterMinutes int    `json:"sleep_after_minutes"`
	Error             string `json:"error,omitempty"`
}

// handleKeepAwake updates the keep-awake settings.
func (s *Server) handleKeepAwake(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}

	var req keepAwakeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, keepAwakeResponse{Error: "invalid JSON"})
		return
	}

	// Validate sleep-after value
	validValues := map[int]bool{0: true, 30: true, 60: true, 120: true, 180: true, 300: true}
	if !validValues[req.SleepAfterMinutes] {
		writeJSON(w, keepAwakeResponse{Error: "invalid sleep_after_minutes value"})
		return
	}

	// Persist to config
	if err := s.cfg.SetKeepAwake(req.Enabled, req.SleepAfterMinutes); err != nil {
		log.Printf("[server] save keep-awake config: %v", err)
		writeJSON(w, keepAwakeResponse{Error: "failed to persist setting"})
		return
	}

	// Apply to device manager
	s.deviceMgr.SetKeepAwake(req.Enabled, req.SleepAfterMinutes)

	log.Printf("[server] keep-awake: enabled=%v, sleep_after=%dm", req.Enabled, req.SleepAfterMinutes)
	writeJSON(w, keepAwakeResponse{
		KeepAwake:         req.Enabled,
		SleepAfterMinutes: req.SleepAfterMinutes,
	})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
