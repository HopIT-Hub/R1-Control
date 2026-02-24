APP_NAME = r1ptt
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS = -ldflags="-s -w -X main.version=$(VERSION)"

.PHONY: all darwin darwin-amd64 windows linux clean package-darwin

all: darwin

# macOS (arm64 — Apple Silicon)
darwin:
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(APP_NAME)-darwin-arm64 ./cmd/tray

# macOS (amd64 — Intel)
darwin-amd64:
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(APP_NAME)-darwin-amd64 ./cmd/tray

# Windows (amd64, hides console window)
windows:
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build $(LDFLAGS)'-H windowsgui' -o $(APP_NAME)-windows-amd64.exe ./cmd/tray

# Linux (amd64)
linux:
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(APP_NAME)-linux-amd64 ./cmd/tray

# macOS .app bundle + .dmg (local build)
package-darwin: darwin
	./packaging/macos/build.sh $(APP_NAME)-darwin-arm64 $(VERSION) arm64

clean:
	rm -f $(APP_NAME)-darwin-* $(APP_NAME)-windows-* $(APP_NAME)-linux-* r1ptt-tray
	rm -rf "R1 Control.app" *.dmg *.AppImage *.zip R1Control.AppDir
