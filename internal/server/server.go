// Package server provides the local HTTP server for the settings UI.
package server

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/HopIT-Hub/R1-Control/internal/config"
	"github.com/HopIT-Hub/R1-Control/internal/device"
	"github.com/HopIT-Hub/R1-Control/internal/hotkey"
	"github.com/HopIT-Hub/R1-Control/internal/web"
)

// Server serves the settings UI on localhost.
type Server struct {
	httpServer  *http.Server
	listener    net.Listener
	hotkeyMgr   *hotkey.Manager
	swipeHkMgr  *hotkey.Manager
	deviceMgr   *device.Manager
	cfg         *config.Config
	version     string
}

// New creates a settings server.
func New(hotkeyMgr *hotkey.Manager, swipeHkMgr *hotkey.Manager, deviceMgr *device.Manager, cfg *config.Config, version string) *Server {
	return &Server{
		hotkeyMgr:  hotkeyMgr,
		swipeHkMgr: swipeHkMgr,
		deviceMgr:  deviceMgr,
		cfg:        cfg,
		version:    version,
	}
}

// Start begins serving on a random localhost port.
// Returns the URL to open in the browser.
func (s *Server) Start() (string, error) {
	mux := http.NewServeMux()

	// Serve embedded static files
	staticFS, err := fs.Sub(web.StaticFiles, "static")
	if err != nil {
		return "", fmt.Errorf("static fs: %w", err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Settings page
	mux.HandleFunc("/", s.handleIndex)

	// API endpoints
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/hotkey", s.handleHotkey)
	mux.HandleFunc("/swipe-hotkey", s.handleSwipeHotkey)
	mux.HandleFunc("/autostart", s.handleAutoStart)
	mux.HandleFunc("/keepawake", s.handleKeepAwake)

	// Bind to random localhost port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("listen: %w", err)
	}
	s.listener = ln

	s.httpServer = &http.Server{
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("[server] error: %v", err)
		}
	}()

	url := fmt.Sprintf("http://%s", ln.Addr().String())
	log.Printf("[server] settings available at %s", url)
	return url, nil
}

// Stop shuts down the HTTP server.
func (s *Server) Stop() {
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		s.httpServer.Shutdown(ctx)
	}
}

// URL returns the server's URL, or empty string if not started.
func (s *Server) URL() string {
	if s.listener == nil {
		return ""
	}
	return fmt.Sprintf("http://%s", s.listener.Addr().String())
}
