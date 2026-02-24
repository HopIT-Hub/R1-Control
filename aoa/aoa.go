// Package aoa implements the Android Open Accessory 2.0 HID protocol.
// It sends HID input events directly to an Android device over USB without
// requiring ADB, developer mode, or any setup on the Android side.
//
// Protocol reference: https://source.android.com/docs/core/interaction/accessories/aoa2
package aoa

import (
	"fmt"
	"time"

	"github.com/google/gousb"
)

const (
	// Rabbit R1 USB vendor/product ID in normal mode
	R1VendorID  = 0x0e8d
	R1ProductID = 0x2304

	// AOA HID control transfer request codes (bRequest values)
	reqRegisterHID   = 54 // ACCESSORY_REGISTER_HID
	reqUnregisterHID = 55 // ACCESSORY_UNREGISTER_HID
	reqSetHIDDesc    = 56 // ACCESSORY_SET_HID_REPORT_DESC
	reqSendHIDEvent  = 57 // ACCESSORY_SEND_HID_EVENT

	// bmRequestType for all AOA HID transfers:
	// host-to-device (0x00) | vendor (0x40) | device recipient (0x00) = 0x40
	bmRequestTypeOut = 0x40

	usbTimeout = 1000 * time.Millisecond
)

// DescriptorType identifies which HID descriptor to use.
type DescriptorType int

const (
	DescKeyboard       DescriptorType = iota // Standard Keyboard (Usage Page 0x07)
	DescConsumerControl                      // Consumer Control (Usage Page 0x0C)
	DescSystemControl                        // Generic Desktop / System Control (Usage Page 0x01)
	DescCameraControl                        // Camera Control (Usage Page 0x90)
	DescTouchScreen                          // Touch Screen Digitizer (Usage Page 0x0D)
)

func (d DescriptorType) String() string {
	switch d {
	case DescKeyboard:
		return "Keyboard (0x07)"
	case DescConsumerControl:
		return "Consumer Control (0x0C)"
	case DescSystemControl:
		return "System Control (0x01)"
	case DescCameraControl:
		return "Camera Control (0x90)"
	case DescTouchScreen:
		return "Touch Screen (0x0D)"
	default:
		return "Unknown"
	}
}

// KeyTest defines a single key to test within a descriptor.
type KeyTest struct {
	Name        string // Human-readable name (e.g., "Volume Up", "Power")
	AndroidKey  string // Expected Android KEYCODE name
	LinuxKey    string // Expected Linux KEY_ name
	ReportDown  []byte // HID report bytes for key-down
	ReportUp    []byte // HID report bytes for key-up (all zeros typically)
	Description string // What this might do on the R1
}

// Keyboard HID report descriptor.
// 8-byte reports: [modifier, reserved, key1, key2, key3, key4, key5, key6]
var keyboardDescriptor = []byte{
	0x05, 0x01, // Usage Page (Generic Desktop)
	0x09, 0x06, // Usage (Keyboard)
	0xA1, 0x01, // Collection (Application)
	// Modifier byte (8 bits: Ctrl, Shift, Alt, GUI x2)
	0x05, 0x07, //   Usage Page (Keyboard/Keypad)
	0x19, 0xE0, //   Usage Minimum (Left Control)
	0x29, 0xE7, //   Usage Maximum (Right GUI)
	0x15, 0x00, //   Logical Minimum (0)
	0x25, 0x01, //   Logical Maximum (1)
	0x75, 0x01, //   Report Size (1)
	0x95, 0x08, //   Report Count (8)
	0x81, 0x02, //   Input (Data, Variable, Absolute) — modifier byte
	// Reserved byte
	0x95, 0x01, //   Report Count (1)
	0x75, 0x08, //   Report Size (8)
	0x81, 0x01, //   Input (Constant) — reserved byte
	// Key array (6 keys)
	0x95, 0x06, //   Report Count (6)
	0x75, 0x08, //   Report Size (8)
	0x15, 0x00, //   Logical Minimum (0)
	0x26, 0xFF, 0x00, // Logical Maximum (255)
	0x05, 0x07, //   Usage Page (Keyboard/Keypad)
	0x19, 0x00, //   Usage Minimum (0)
	0x29, 0xFF, //   Usage Maximum (255)
	0x81, 0x00, //   Input (Data, Array)
	0xC0, // End Collection
}

// Consumer Control HID report descriptor.
// 2-byte report: 16-bit usage value (little-endian).
var consumerDescriptor = []byte{
	0x05, 0x0C, // Usage Page (Consumer)
	0x09, 0x01, // Usage (Consumer Control)
	0xA1, 0x01, // Collection (Application)
	0x15, 0x00, // Logical Minimum (0)
	0x26, 0xFF, 0x0F, // Logical Maximum (4095)
	0x19, 0x00, // Usage Minimum (0)
	0x2A, 0xFF, 0x0F, // Usage Maximum (4095)
	0x75, 0x10, // Report Size (16 bits)
	0x95, 0x01, // Report Count (1)
	0x81, 0x00, // Input (Data, Array)
	0xC0, // End Collection
}

// System Control HID report descriptor.
// 1-byte report with system control usage.
var systemControlDescriptor = []byte{
	0x05, 0x01, // Usage Page (Generic Desktop)
	0x09, 0x80, // Usage (System Control)
	0xA1, 0x01, // Collection (Application)
	0x15, 0x01, // Logical Minimum (1)
	0x25, 0x03, // Logical Maximum (3)
	0x09, 0x81, // Usage (System Power Down)
	0x09, 0x82, // Usage (System Sleep)
	0x09, 0x83, // Usage (System Wake Up)
	0x75, 0x08, // Report Size (8 bits)
	0x95, 0x01, // Report Count (1)
	0x81, 0x00, // Input (Data, Array)
	0xC0, // End Collection
}

// Camera Control HID report descriptor.
// 1-byte report: bit 0 = Auto Focus, bit 1 = Shutter.
var cameraControlDescriptor = []byte{
	0x05, 0x90, // Usage Page (Camera Control)
	0x09, 0x20, // Usage (Camera Auto Focus)
	0xA1, 0x01, // Collection (Application)
	0x15, 0x00, // Logical Minimum (0)
	0x25, 0x01, // Logical Maximum (1)
	0x75, 0x01, // Report Size (1 bit)
	0x95, 0x01, // Report Count (1)
	0x09, 0x20, // Usage (Camera Auto Focus)
	0x81, 0x02, // Input (Data, Variable, Absolute)
	0x09, 0x21, // Usage (Camera Shutter)
	0x81, 0x02, // Input (Data, Variable, Absolute)
	0x95, 0x06, // Report Count (6 — padding bits)
	0x81, 0x03, // Input (Constant)
	0xC0,       // End Collection
}

// Touch Screen HID report descriptor (single-touch digitizer).
// 5-byte report: [tip_switch(1bit)+in_range(1bit)+pad(6bits), x_lo, x_hi, y_lo, y_hi]
// Used to simulate swipe gestures by sending a sequence of touch reports.
var touchScreenDescriptor = []byte{
	0x05, 0x0D, // Usage Page (Digitizers)
	0x09, 0x04, // Usage (Touch Screen)
	0xA1, 0x01, // Collection (Application)
	0x09, 0x22, //   Usage (Finger)
	0xA1, 0x02, //   Collection (Logical)

	// Tip Switch — 1 bit (finger contact)
	0x09, 0x42, //     Usage (Tip Switch)
	0x15, 0x00, //     Logical Minimum (0)
	0x25, 0x01, //     Logical Maximum (1)
	0x75, 0x01, //     Report Size (1)
	0x95, 0x01, //     Report Count (1)
	0x81, 0x02, //     Input (Data, Variable, Absolute)

	// In Range — 1 bit (finger near surface)
	0x09, 0x32, //     Usage (In Range)
	0x81, 0x02, //     Input (Data, Variable, Absolute)

	// Padding — 6 bits to fill the byte
	0x75, 0x06, //     Report Size (6)
	0x95, 0x01, //     Report Count (1)
	0x81, 0x03, //     Input (Constant)

	// X coordinate — 16 bits (0-32767)
	0x05, 0x01,       //     Usage Page (Generic Desktop)
	0x09, 0x30,       //     Usage (X)
	0x15, 0x00,       //     Logical Minimum (0)
	0x26, 0xFF, 0x7F, //     Logical Maximum (32767)
	0x75, 0x10,       //     Report Size (16)
	0x95, 0x01,       //     Report Count (1)
	0x81, 0x02,       //     Input (Data, Variable, Absolute)

	// Y coordinate — 16 bits (0-32767)
	0x09, 0x31, //     Usage (Y)
	0x81, 0x02, //     Input (Data, Variable, Absolute)

	0xC0, //   End Collection (Logical)
	0xC0, // End Collection (Application)
}

// TouchReport builds a 5-byte touch screen report.
// tip: true = finger touching, false = finger lifted.
// x, y: coordinates in 0-32767 range.
func TouchReport(tip bool, x, y uint16) []byte {
	var flags byte
	if tip {
		flags = 0x03 // bit 0 = Tip Switch, bit 1 = In Range
	}
	return []byte{
		flags,
		byte(x & 0xFF), byte(x >> 8),
		byte(y & 0xFF), byte(y >> 8),
	}
}

// GetDescriptor returns the raw HID descriptor for the given type.
func GetDescriptor(dt DescriptorType) []byte {
	switch dt {
	case DescKeyboard:
		return keyboardDescriptor
	case DescConsumerControl:
		return consumerDescriptor
	case DescSystemControl:
		return systemControlDescriptor
	case DescCameraControl:
		return cameraControlDescriptor
	case DescTouchScreen:
		return touchScreenDescriptor
	default:
		return nil
	}
}

// GetKeyTests returns the list of keys to test for a given descriptor type.
func GetKeyTests(dt DescriptorType) []KeyTest {
	switch dt {
	case DescKeyboard:
		return keyboardTests()
	case DescConsumerControl:
		return consumerTests()
	case DescSystemControl:
		return systemControlTests()
	case DescCameraControl:
		return cameraControlTests()
	default:
		return nil
	}
}

// --- Keyboard key tests ---
// Report format: [modifier, 0x00, key1, 0, 0, 0, 0, 0]
func kbDown(scancode byte) []byte { return []byte{0, 0, scancode, 0, 0, 0, 0, 0} }

var kbUp = []byte{0, 0, 0, 0, 0, 0, 0, 0}

func keyboardTests() []KeyTest {
	return []KeyTest{
		{"Power (0x66)", "KEYCODE_POWER", "KEY_POWER", kbDown(0x66), kbUp,
			"Keyboard Power → KEY_POWER → might trigger KeyHandler → KEYCODE_FOCUS"},
		{"F13 (0x68)", "KEYCODE_F13", "KEY_F13", kbDown(0x68), kbUp,
			"F13 — unusual key, might pass through to app"},
		{"F14 (0x69)", "KEYCODE_F14", "KEY_F14", kbDown(0x69), kbUp,
			"F14 — unusual key, might pass through to app"},
		{"F15 (0x6A)", "KEYCODE_F15", "KEY_F15", kbDown(0x6A), kbUp,
			"F15 — unusual key, might pass through to app"},
		{"F16 (0x6B)", "KEYCODE_F16", "KEY_F16", kbDown(0x6B), kbUp,
			"F16 — unusual key, might pass through to app"},
		{"F17 (0x6C)", "KEYCODE_F17", "KEY_F17", kbDown(0x6C), kbUp,
			"F17 — unusual key, might pass through to app"},
		{"F18 (0x6D)", "KEYCODE_F18", "KEY_F18", kbDown(0x6D), kbUp,
			"F18 — unusual key, might pass through to app"},
		{"F19 (0x6E)", "KEYCODE_F19", "KEY_F19", kbDown(0x6E), kbUp,
			"F19 — unusual key, might pass through to app"},
		{"F20 (0x6F)", "KEYCODE_F20", "KEY_F20", kbDown(0x6F), kbUp,
			"F20 — unusual key, might pass through to app"},
		{"F24 (0x73)", "KEYCODE_F24", "KEY_F24", kbDown(0x73), kbUp,
			"F24 — unusual key, might pass through to app"},
		{"Up Arrow (0x52)", "KEYCODE_DPAD_UP", "KEY_UP", kbDown(0x52), kbUp,
			"Known working from scrcpy OTG — scrolls UI. Control test."},
		{"Enter (0x28)", "KEYCODE_ENTER", "KEY_ENTER", kbDown(0x28), kbUp,
			"Enter key — should select current item"},
		{"Space (0x2C)", "KEYCODE_SPACE", "KEY_SPACE", kbDown(0x2C), kbUp,
			"Space — some apps use this for play/pause or action"},
		{"A key (0x04)", "KEYCODE_A", "KEY_A", kbDown(0x04), kbUp,
			"Letter A — basic keyboard test. Control test."},
	}
}

// --- Consumer Control key tests ---
// Report format: 2-byte little-endian usage ID
func ccDown(usage uint16) []byte { return []byte{byte(usage & 0xFF), byte(usage >> 8)} }

var ccUp = []byte{0x00, 0x00}

func consumerTests() []KeyTest {
	return []KeyTest{
		{"Volume Up (0xE9)", "KEYCODE_VOLUME_UP", "KEY_VOLUMEUP", ccDown(0x00E9), ccUp,
			"Volume Up — previously scrolled UI. Re-test with hold."},
		{"Volume Down (0xEA)", "KEYCODE_VOLUME_DOWN", "KEY_VOLUMEDOWN", ccDown(0x00EA), ccUp,
			"Volume Down — the original research said PTT = volume down"},
		{"Mute (0xE2)", "KEYCODE_VOLUME_MUTE", "KEY_MUTE", ccDown(0x00E2), ccUp,
			"Mute toggle"},
		{"Play/Pause (0xCD)", "KEYCODE_MEDIA_PLAY_PAUSE", "KEY_PLAYPAUSE", ccDown(0x00CD), ccUp,
			"Media Play/Pause — might be handled differently"},
		{"Stop (0xB7)", "KEYCODE_MEDIA_STOP", "KEY_STOPCD", ccDown(0x00B7), ccUp,
			"Media Stop"},
		{"Record (0xB2)", "KEYCODE_MEDIA_RECORD", "KEY_RECORD", ccDown(0x00B2), ccUp,
			"Record — could be mapped to PTT/recording"},
		{"AL Camera App (0x192)", "KEYCODE_CAMERA", "KEY_CAMERA", ccDown(0x0192), ccUp,
			"Consumer Camera usage — might trigger camera/PTT"},
		{"Power (0x30)", "KEYCODE_POWER", "KEY_POWER", ccDown(0x0030), ccUp,
			"Consumer Power — same as physical power key?"},
		{"Sleep (0x34)", "KEYCODE_SLEEP", "KEY_SLEEP", ccDown(0x0034), ccUp,
			"Consumer Sleep"},
		{"Channel Up (0x9C)", "KEYCODE_CHANNEL_UP", "KEY_CHANNELUP", ccDown(0x009C), ccUp,
			"Channel Up — unusual, might pass through"},
		{"Channel Down (0x9D)", "KEYCODE_CHANNEL_DOWN", "KEY_CHANNELDOWN", ccDown(0x009D), ccUp,
			"Channel Down — unusual, might pass through"},
		{"AC Search (0x221)", "KEYCODE_SEARCH", "KEY_SEARCH", ccDown(0x0221), ccUp,
			"Search — might trigger voice search / assistant"},
		{"AC Home (0x223)", "KEYCODE_HOME", "KEY_HOMEPAGE", ccDown(0x0223), ccUp,
			"Home — might navigate home"},
		{"AC Back (0x224)", "KEYCODE_BACK", "KEY_BACK", ccDown(0x0224), ccUp,
			"Back — navigation test"},
		{"Menu (0x40)", "KEYCODE_MENU", "KEY_MENU", ccDown(0x0040), ccUp,
			"Menu — might open app menu"},
		{"Voice Command (0xCF)", "KEYCODE_VOICE_ASSIST", "KEY_VOICECOMMAND", ccDown(0x00CF), ccUp,
			"Voice Command — could trigger voice/PTT!"},
		{"AL Task/Project Manager (0x19F)", "KEYCODE_APP_SWITCH", "KEY_APPSELECT", ccDown(0x019F), ccUp,
			"App Switch — recent apps"},
		{"Assist (0x1CB)", "KEYCODE_ASSIST", "KEY_ASSISTANT", ccDown(0x01CB), ccUp,
			"Assistant — could trigger voice assistant / PTT!"},
	}
}

// --- System Control key tests ---
// Report format: 1-byte usage selector (1=Power, 2=Sleep, 3=Wake)
func systemControlTests() []KeyTest {
	return []KeyTest{
		{"System Power Down (0x81)", "KEYCODE_POWER", "KEY_POWER", []byte{0x01}, []byte{0x00},
			"WARNING: May turn off screen. System Power Down → KEY_POWER"},
		{"System Sleep (0x82)", "KEYCODE_SLEEP", "KEY_SLEEP", []byte{0x02}, []byte{0x00},
			"System Sleep — may put device to sleep"},
		{"System Wake Up (0x83)", "KEYCODE_WAKEUP", "KEY_WAKEUP", []byte{0x03}, []byte{0x00},
			"System Wake Up — should wake the screen"},
	}
}

// --- Camera Control key tests ---
// Report format: 1-byte bitfield (bit0=AutoFocus, bit1=Shutter)
func cameraControlTests() []KeyTest {
	return []KeyTest{
		{"Camera Auto Focus (0x20)", "KEYCODE_FOCUS", "KEY_CAMERA_FOCUS", []byte{0x01}, []byte{0x00},
			"Camera Auto Focus → KEY_CAMERA_FOCUS → KEYCODE_FOCUS. Requires kernel support."},
		{"Camera Shutter (0x21)", "KEYCODE_CAMERA", "KEY_CAMERA", []byte{0x02}, []byte{0x00},
			"Camera Shutter → KEY_CAMERA → KEYCODE_CAMERA. Requires kernel support."},
		{"Both Focus+Shutter", "FOCUS+CAMERA", "KEY_CAMERA_FOCUS+KEY_CAMERA", []byte{0x03}, []byte{0x00},
			"Both Focus and Shutter simultaneously"},
	}
}

// Device wraps a libusb handle to an Android device with AOA HID set up.
type Device struct {
	ctx        *gousb.Context
	dev        *gousb.Device
	serial     string
	nextHIDID  uint16   // next HID ID to assign
	registered []uint16 // all registered HID IDs for cleanup
	lastHIDID  uint16   // most recently registered ID (for compat methods)
}

// Open finds a connected R1 and opens a USB connection (no HID registration yet).
func Open(serial string) (*Device, error) {
	ctx := gousb.NewContext()

	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return desc.Vendor == R1VendorID && desc.Product == R1ProductID
	})
	if err != nil && len(devs) == 0 {
		ctx.Close()
		return nil, fmt.Errorf("no Rabbit R1 found (VID:0x%04x PID:0x%04x): %w", R1VendorID, R1ProductID, err)
	}

	var dev *gousb.Device
	for _, d := range devs {
		s, _ := d.SerialNumber()
		if serial == "" || s == serial {
			dev = d
		} else {
			d.Close()
		}
	}
	if dev == nil {
		ctx.Close()
		return nil, fmt.Errorf("R1 with serial %q not found", serial)
	}

	dev.SetAutoDetach(true)

	return &Device{ctx: ctx, dev: dev, nextHIDID: 1}, nil
}

// RegisterDescriptor registers an HID descriptor with the device via AOA2.
// Returns the assigned HID ID for use with SendReportTo/TapTo.
func (d *Device) RegisterDescriptor(dt DescriptorType) (uint16, error) {
	desc := GetDescriptor(dt)
	if desc == nil {
		return 0, fmt.Errorf("unknown descriptor type %d", dt)
	}

	id := d.nextHIDID
	d.nextHIDID++

	// Register HID device (wValue = HID ID, wIndex = descriptor length)
	if err := d.controlTransfer(reqRegisterHID, id, uint16(len(desc)), nil); err != nil {
		return 0, fmt.Errorf("REGISTER_HID failed: %w", err)
	}

	// Send the HID report descriptor
	if err := d.controlTransfer(reqSetHIDDesc, id, 0, desc); err != nil {
		_ = d.controlTransfer(reqUnregisterHID, id, 0, nil)
		return 0, fmt.Errorf("SET_HID_REPORT_DESC failed: %w", err)
	}

	// Give Android time to create the input device
	time.Sleep(300 * time.Millisecond)

	d.registered = append(d.registered, id)
	d.lastHIDID = id
	return id, nil
}

// UnregisterDescriptor removes the most recently registered HID device.
func (d *Device) UnregisterDescriptor() error {
	if len(d.registered) == 0 {
		return nil
	}
	id := d.registered[len(d.registered)-1]
	d.registered = d.registered[:len(d.registered)-1]
	err := d.controlTransfer(reqUnregisterHID, id, 0, nil)
	time.Sleep(200 * time.Millisecond)
	return err
}

// SendReport sends a raw HID report to the most recently registered descriptor.
func (d *Device) SendReport(report []byte) error {
	return d.SendReportTo(d.lastHIDID, report)
}

// SendReportTo sends a raw HID report to a specific descriptor by HID ID.
func (d *Device) SendReportTo(hidID uint16, report []byte) error {
	return d.controlTransfer(reqSendHIDEvent, hidID, 0, report)
}

// Tap sends a key-down followed by a key-up with a short delay.
func (d *Device) Tap(down, up []byte) error {
	return d.TapTo(d.lastHIDID, down, up)
}

// TapTo sends a key-down followed by a key-up to a specific descriptor.
func (d *Device) TapTo(hidID uint16, down, up []byte) error {
	if err := d.SendReportTo(hidID, down); err != nil {
		return fmt.Errorf("key down: %w", err)
	}
	time.Sleep(80 * time.Millisecond)
	if err := d.SendReportTo(hidID, up); err != nil {
		return fmt.Errorf("key up: %w", err)
	}
	return nil
}

// HoldDown sends a key-down report and keeps it held.
func (d *Device) HoldDown(down []byte) error {
	return d.SendReport(down)
}

// Release sends a key-up report.
func (d *Device) Release(up []byte) error {
	return d.SendReport(up)
}

// Ping checks if the device is still connected by reading its serial number.
func (d *Device) Ping() error {
	_, err := d.dev.SerialNumber()
	return err
}

// Close releases USB resources.
func (d *Device) Close() {
	for _, id := range d.registered {
		_ = d.controlTransfer(reqUnregisterHID, id, 0, nil)
	}
	d.registered = nil
	d.dev.Close()
	d.ctx.Close()
}

// controlTransfer sends a vendor control transfer to the device.
func (d *Device) controlTransfer(bRequest uint8, wValue uint16, wIndex uint16, data []byte) error {
	if data == nil {
		data = []byte{}
	}
	_, err := d.dev.Control(
		bmRequestTypeOut,
		bRequest,
		wValue,
		wIndex,
		data,
	)
	if err != nil {
		return fmt.Errorf("control transfer (req=%d wValue=%d wIndex=%d): %w", bRequest, wValue, wIndex, err)
	}
	return nil
}
