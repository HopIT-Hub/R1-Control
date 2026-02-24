// Package autostart manages registering the app to start on login.
// Each platform has its own implementation file.
package autostart

import "os"

// appPath returns the path to the currently running executable.
func appPath() (string, error) {
	return os.Executable()
}
