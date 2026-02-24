<p align="center">
  <img src="assets/header-image.png" alt="R1 Control — Control your Rabbit R1 from the comfort of your keyboard" width="100%">
</p>

<div align="center">

[![Latest Release](https://img.shields.io/github/v/release/HopIT-Hub/R1-Control?style=for-the-badge&color=FF6B2B&label=Latest+Release)](https://github.com/HopIT-Hub/R1-Control/releases/latest) ![macOS](https://img.shields.io/badge/macOS-arm64%20%7C%20Intel-lightgrey?style=for-the-badge&logo=apple) ![Windows](https://img.shields.io/badge/Windows-x64-lightgrey?style=for-the-badge&logo=windows) ![Linux](https://img.shields.io/badge/Linux-x64-lightgrey?style=for-the-badge&logo=linux&logoColor=white) [![License](https://img.shields.io/badge/License-Noncommercial-FF6B2B?style=for-the-badge)](LICENSE) [![Ko-Fi](https://img.shields.io/badge/Ko--Fi-Support_the_Project-FF5E5B?style=for-the-badge&logo=ko-fi&logoColor=white)](https://ko-fi.com/hopit)

</div>

---

## About

**R1 Control** is a lightweight, native system-tray app that lets you use your [Rabbit R1](https://www.rabbit.tech) from your computer — send push-to-talk messages, trigger swipes, and interact with Rabbit AI directly from your keyboard. No phone in hand required.

- 🎙️ **Push-to-Talk** — tap or hold a hotkey to talk to your R1
- ⬅️ **Swipe** — trigger left/right swipes from the keyboard
- 🖥️ **Native tray app** — lives quietly in your menu bar or system tray
- 🔌 **USB direct** — communicates over USB, no Wi-Fi required
- 🪶 **Lightweight** — minimal footprint, no subscriptions, no cloud account

---

## Download

Grab the latest release for your platform from the [**Releases page**](https://github.com/HopIT-Hub/R1-Control/releases/latest):

| Platform | File |
|---|---|
| macOS (Apple Silicon) | `R1-Control-*-macos-arm64.dmg` |
| macOS (Intel) | `R1-Control-*-macos-amd64.dmg` |
| Windows | `R1-Control-*-windows-amd64.zip` |
| Linux | `R1-Control-*-linux-amd64.AppImage` |

---

## Installation

### macOS

1. Download the `.dmg` for your Mac (Apple Silicon or Intel)
2. Open the `.dmg` and drag **R1 Control** to your Applications folder
3. **First launch only:** right-click the app → **Open** → **Open** to bypass Gatekeeper
4. The app runs in your menu bar — look for the circle icon

### Windows

1. Download and extract the `.zip`
2. First time only — install the WinUSB driver for your R1 using [Zadig](https://zadig.akeo.ie/):
   - Open Zadig → Options → List All Devices → select your R1 → **Install WinUSB**
3. Run `R1 Control.exe` — the app appears in your system tray

### Linux

1. Download the `.AppImage`
2. Make it executable and run:
   ```bash
   chmod +x R1-Control-*.AppImage && ./R1-Control-*.AppImage
   ```
3. For USB access without `sudo`, install the udev rules (included in the release):
   ```bash
   sudo cp 99-r1control.rules /etc/udev/rules.d/
   sudo udevadm control --reload-rules && sudo udevadm trigger
   ```

---

## Usage

Connect your R1 via USB and launch the app — no configuration needed. R1 Control auto-detects your device and creates its settings on first run. Default keyboard shortcuts:

`Ctrl+Alt+R` talks to your Rabbit R1 — tap to toggle, hold to talk. `Ctrl+Alt+W` switches between Rabbit and OpenClaw, or from Wabbit 🐰 to Wobster 🦞. Both hotkeys are fully customizable in Settings.

| Action | Shortcut |
|---|---|
| Push-to-Talk (tap to toggle / hold to talk) | `Ctrl + Alt + R` |
| Swipe (alternates left/right) | `Ctrl + Alt + W` |
| Open Settings | Click the tray icon → **Settings** |

---

## Building from Source

**Requirements:** Go 1.21+, libusb (`brew install libusb` / `apt install libusb-1.0-0-dev` / vcpkg on Windows)

```bash
git clone https://github.com/HopIT-Hub/R1-Control.git
cd R1-Control

make                # macOS Apple Silicon
make darwin-amd64   # macOS Intel
make linux          # Linux x64
make windows        # Windows x64 (requires mingw-w64)

make package-darwin # Build .app bundle + .dmg (macOS)
```

---

## Support the Project

If R1 Control is useful to you, a coffee goes a long way! ☕

<a href="https://ko-fi.com/hopit">
  <img src="https://img.shields.io/badge/Support_on_Ko--Fi-000000?style=for-the-badge&logo=ko-fi&logoColor=FF5E5B" alt="Support on Ko-Fi">
</a>

---

## License

**Free for personal use** — see [LICENSE](LICENSE) for full terms.

Commercial or closed-source use requires a license — contact [licensing@hopit.co](mailto:licensing@hopit.co).

> R1 Control is an independent, community-built project and is not affiliated with or endorsed by Rabbit Inc.
