// Package device manages the USB connection to the Rabbit R1.
// It automatically detects the device when plugged in, reconnects
// on disconnect, and exposes PTT and navigation controls.
//
// PTT uses AOA2 HID over USB for low-latency hold-to-talk.
// Swipe gestures use AOA2 HID touch screen digitizer to simulate
// finger swipe input on the R1.
package device

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/HopIT-Hub/R1-Control/aoa"
)

// State represents the current device/PTT state.
type State int

const (
	Disconnected State = iota
	Connected
	PTTActive
)

func (s State) String() string {
	switch s {
	case Disconnected:
		return "disconnected"
	case Connected:
		return "connected"
	case PTTActive:
		return "ptt_active"
	default:
		return "unknown"
	}
}

// System Control HID reports.
var (
	powerDown = []byte{0x01} // System Power Down
	powerUp   = []byte{0x00} // Release
	wakeUp    = []byte{0x03} // System Wake Up
)

// Toggle/hold detection threshold.
const toggleThreshold = 300 * time.Millisecond

// Keep-awake defaults.
const (
	keepAwakeInterval = 25 * time.Second // beats R1's shortest 30s auto-sleep
)

// Manager handles the R1 USB device lifecycle.
type Manager struct {
	mu       sync.Mutex
	dev      *aoa.Device
	state    State
	onChange func(State) // callback when state changes
	serial   string      // optional serial filter

	// HID descriptor IDs (assigned on connect)
	pttHIDID   uint16
	touchHIDID uint16

	// PTT toggle state
	pttToggled   bool      // true if PTT is toggled on via short press
	pttPressTime time.Time // when the hotkey was last pressed down

	// Swipe direction state
	swipeLeft bool // true = next swipe is left, false = right

	// Keep-awake state
	keepAwake         bool      // whether to send periodic wake pings
	sleepAfterMinutes int       // 0 = never sleep
	lastActivity      time.Time // last PTT/Swipe action time
	sleeping          bool      // true when idle timer has expired
}

// NewManager creates a new device manager.
// onChange is called whenever the device state changes.
func NewManager(serial string, onChange func(State)) *Manager {
	return &Manager{
		state:        Disconnected,
		onChange:     onChange,
		serial:      serial,
		swipeLeft:    true, // first swipe will be left
		keepAwake:    true, // default: keep device awake
		sleepAfterMinutes: 60, // default: 1 hour
		lastActivity: time.Now(),
	}
}

// SetKeepAwake configures the keep-awake behaviour.
func (m *Manager) SetKeepAwake(enabled bool, sleepAfterMinutes int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.keepAwake = enabled
	m.sleepAfterMinutes = sleepAfterMinutes
	// Reset idle timer when settings change
	m.lastActivity = time.Now()
	m.sleeping = false
}

// touchActivity resets the idle timer. Must be called with m.mu held.
func (m *Manager) touchActivity() {
	m.lastActivity = time.Now()
	m.sleeping = false
}

// State returns the current device state.
func (m *Manager) State() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// Run starts the device auto-detection loop. It polls for the R1 every
// 2 seconds. When connected, it also checks device health and sends
// periodic keep-awake pings.
// Blocks until ctx is cancelled.
func (m *Manager) Run(ctx context.Context) {
	pollTicker := time.NewTicker(2 * time.Second)
	defer pollTicker.Stop()

	wakeTicker := time.NewTicker(keepAwakeInterval)
	defer wakeTicker.Stop()

	// Try immediately on start
	m.tryConnect()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pollTicker.C:
			m.mu.Lock()
			state := m.state
			m.mu.Unlock()

			if state == Disconnected {
				m.tryConnect()
			} else {
				m.healthCheck()
			}
		case <-wakeTicker.C:
			m.keepAwakePing()
		}
	}
}

// keepAwakePing sends a wake tap if keep-awake is enabled and the idle
// timer hasn't expired. This prevents the R1 from auto-sleeping.
func (m *Manager) keepAwakePing() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Only ping if connected (not during PTT active — screen is already on)
	if m.dev == nil || m.state != Connected {
		return
	}

	if !m.keepAwake {
		return
	}

	// Check idle timer (0 = never sleep)
	if m.sleepAfterMinutes > 0 {
		idleLimit := time.Duration(m.sleepAfterMinutes) * time.Minute
		if time.Since(m.lastActivity) >= idleLimit {
			if !m.sleeping {
				m.sleeping = true
				log.Printf("[device] idle for %v — letting device sleep", idleLimit)
			}
			return
		}
	}

	// Two-step keep-alive:
	// 1. System Wake Up — wakes the screen if the device is sleeping
	// 2. Touch tap — resets the R1's sleep countdown timer
	//    (Wake Up alone doesn't count as "user interaction")
	_ = m.dev.SendReportTo(m.pttHIDID, wakeUp)
	time.Sleep(50 * time.Millisecond)
	_ = m.dev.SendReportTo(m.pttHIDID, powerUp)
	time.Sleep(150 * time.Millisecond) // let the screen come on before touching

	touchDown := aoa.TouchReport(true, 32590, 32590) // bottom-right corner
	touchUp := aoa.TouchReport(false, 32590, 32590)
	_ = m.dev.SendReportTo(m.touchHIDID, touchDown)
	time.Sleep(30 * time.Millisecond)
	_ = m.dev.SendReportTo(m.touchHIDID, touchUp)
}

// tryConnect attempts to open the R1 and register HID descriptors.
func (m *Manager) tryConnect() {
	dev, err := aoa.Open(m.serial)
	if err != nil {
		return // device not found, will retry
	}

	// Register System Control descriptor for PTT (Power key)
	pttID, err := dev.RegisterDescriptor(aoa.DescSystemControl)
	if err != nil {
		log.Printf("[device] PTT HID register failed: %v", err)
		dev.Close()
		return
	}

	// Register Touch Screen descriptor for swipe gestures
	touchID, err := dev.RegisterDescriptor(aoa.DescTouchScreen)
	if err != nil {
		log.Printf("[device] Touch HID register failed: %v", err)
		dev.Close()
		return
	}

	m.mu.Lock()
	m.dev = dev
	m.state = Connected
	m.pttHIDID = pttID
	m.touchHIDID = touchID
	m.pttToggled = false
	m.lastActivity = time.Now()
	m.sleeping = false
	m.mu.Unlock()

	log.Println("[device] R1 connected")
	if m.onChange != nil {
		m.onChange(Connected)
	}

	// Immediately wake the device on connect if keep-awake is enabled
	m.keepAwakePing()
}

// healthCheck verifies the device is still connected.
func (m *Manager) healthCheck() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.dev == nil {
		return
	}

	if err := m.dev.Ping(); err != nil {
		log.Printf("[device] R1 disconnected: %v", err)
		m.dev.Close()
		m.dev = nil
		m.state = Disconnected
		m.pttToggled = false
		if m.onChange != nil {
			m.onChange(Disconnected)
		}
	}
}

// wake sends a System Wake Up tap to ensure the R1 screen is on.
// Must be called with m.mu held and m.dev != nil.
func (m *Manager) wake() {
	// Send wake-up key tap: down then up
	if err := m.dev.SendReportTo(m.pttHIDID, wakeUp); err != nil {
		return // best-effort, don't fail the caller
	}
	time.Sleep(50 * time.Millisecond)
	_ = m.dev.SendReportTo(m.pttHIDID, powerUp)
	time.Sleep(100 * time.Millisecond) // give the screen time to turn on
}

// PTTDown is called when the PTT hotkey is pressed down.
// Implements toggle/hold: short press toggles, hold activates until release.
func (m *Manager) PTTDown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.dev == nil {
		return fmt.Errorf("no device connected")
	}

	m.pttPressTime = time.Now()
	m.touchActivity() // reset idle timer

	if m.pttToggled {
		// PTT is already on from toggle — don't re-send key down
		return nil
	}

	// Wake screen before starting PTT
	m.wake()

	// Start PTT
	if err := m.dev.SendReportTo(m.pttHIDID, powerDown); err != nil {
		m.handleError(err)
		return err
	}

	m.state = PTTActive
	if m.onChange != nil {
		m.onChange(PTTActive)
	}
	return nil
}

// PTTUp is called when the PTT hotkey is released.
// Short press (<300ms) toggles PTT on/off; long press releases PTT.
func (m *Manager) PTTUp() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.dev == nil {
		return fmt.Errorf("no device connected")
	}

	duration := time.Since(m.pttPressTime)

	if duration < toggleThreshold {
		// Short press — toggle
		if m.pttToggled {
			// Toggle OFF
			m.pttToggled = false
			if err := m.dev.SendReportTo(m.pttHIDID, powerUp); err != nil {
				m.handleError(err)
				return err
			}
			m.state = Connected
			if m.onChange != nil {
				m.onChange(Connected)
			}
		} else {
			// Toggle ON — leave PTT active
			m.pttToggled = true
		}
		return nil
	}

	// Long press — always release
	m.pttToggled = false
	if err := m.dev.SendReportTo(m.pttHIDID, powerUp); err != nil {
		m.handleError(err)
		return err
	}

	m.state = Connected
	if m.onChange != nil {
		m.onChange(Connected)
	}
	return nil
}

// Swipe sends a swipe gesture via AOA2 touch screen HID.
// Alternates between swipe left and swipe right on each call.
// Simulates a finger swipe by sending interpolated touch reports.
func (m *Manager) Swipe() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.dev == nil {
		return fmt.Errorf("no device connected")
	}

	m.touchActivity() // reset idle timer

	// Wake screen before swiping
	m.wake()

	// Determine swipe direction
	var startX, endX uint16
	var dir string
	if m.swipeLeft {
		startX, endX, dir = 27000, 5000, "LEFT"
	} else {
		startX, endX, dir = 5000, 27000, "RIGHT"
	}
	m.swipeLeft = !m.swipeLeft

	// Y coordinate: near bottom of screen to minimize cursor visibility
	const y uint16 = 32590

	// Send 8 interpolated touch points with finger down
	const steps = 8
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := uint16(float64(startX) + t*float64(int(endX)-int(startX)))
		report := aoa.TouchReport(true, x, y)
		if err := m.dev.SendReportTo(m.touchHIDID, report); err != nil {
			m.handleError(err)
			return fmt.Errorf("swipe step %d: %w", i, err)
		}
		if i < steps {
			time.Sleep(25 * time.Millisecond)
		}
	}

	// Lift finger
	liftReport := aoa.TouchReport(false, endX, y)
	if err := m.dev.SendReportTo(m.touchHIDID, liftReport); err != nil {
		m.handleError(err)
		return fmt.Errorf("swipe lift: %w", err)
	}

	log.Printf("[device] swipe %s", dir)
	return nil
}

// handleError marks the device as disconnected on USB errors.
// Must be called with m.mu held.
func (m *Manager) handleError(err error) {
	log.Printf("[device] USB error: %v — will reconnect", err)
	if m.dev != nil {
		m.dev.Close()
		m.dev = nil
	}
	m.state = Disconnected
	m.pttToggled = false
	if m.onChange != nil {
		m.onChange(Disconnected)
	}
}

// Close shuts down the device connection cleanly.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.dev != nil {
		// Release PTT if active
		if m.state == PTTActive {
			_ = m.dev.SendReportTo(m.pttHIDID, powerUp)
		}
		m.dev.Close()
		m.dev = nil
	}
	m.state = Disconnected
	m.pttToggled = false
}
