<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/gogpu/.github/main/assets/logo.png">
    <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/gogpu/.github/main/assets/logo.png">
    <img src="https://raw.githubusercontent.com/gogpu/.github/main/assets/logo.png" alt="GoGPU Logo" width="100" />
  </picture>
</p>

<h1 align="center">systray</h1>

<p align="center">
  <strong>Pure Go system tray library for Windows, macOS, and Linux</strong><br>
  Zero CGO. Cross-platform. Multiple trays. Context menus. Notifications.
</p>

<p align="center">
  <a href="https://github.com/gogpu/systray/actions"><img src="https://github.com/gogpu/systray/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://pkg.go.dev/github.com/gogpu/systray"><img src="https://pkg.go.dev/badge/github.com/gogpu/systray.svg" alt="Go Reference"></a>
  <a href="https://goreportcard.com/report/github.com/gogpu/systray"><img src="https://goreportcard.com/badge/github.com/gogpu/systray" alt="Go Report Card"></a>
  <a href="https://github.com/gogpu/systray/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License"></a>
  <a href="https://github.com/gogpu/systray"><img src="https://img.shields.io/badge/Pure_Go-Zero_CGO-brightgreen" alt="Zero CGO"></a>
</p>

---

## Features

- **Pure Go** — zero CGO on all platforms. Single binary, easy cross-compilation
- **Multiple trays** — create as many tray icons as you need
- **Context menus** — nested menus with checkboxes, separators, icons, and submenus
- **Notifications** — balloon tips (Windows), notification center (macOS), D-Bus notifications (Linux)
- **Dark mode** — automatic icon switching for light/dark themes
- **Template icons** — macOS-native monochrome icons that adapt to system theme
- **Builder pattern** — fluent API for clean, readable code
- **Standalone** — no dependency on gogpu framework. Use in any Go application

## Platform Implementation

| Platform | API | Dependency | Status |
|:---------|:----|:-----------|:------:|
| **Windows** | `Shell_NotifyIconW` (shell32.dll) | `golang.org/x/sys/windows` | Planned |
| **macOS** | `NSStatusBar` / `NSStatusItem` (AppKit) | `github.com/go-webgpu/goffi` | Planned |
| **Linux** | StatusNotifierItem (D-Bus SNI) | `github.com/godbus/dbus/v5` | Planned |

All platform implementations use Pure Go FFI — no C compiler required.

## Installation

```bash
go get github.com/gogpu/systray
```

**Requirements:** Go 1.25+

## Quick Start

```go
package main

import (
    _ "embed"
    "github.com/gogpu/systray"
)

//go:embed icon.png
var iconPNG []byte

func main() {
    // Create a system tray icon
    tray := systray.New()
    tray.SetIcon(iconPNG)
    tray.SetTooltip("My Application")

    // Build context menu
    menu := systray.NewMenu()
    menu.Add("Open", func() { /* show main window */ })
    menu.Add("Settings", func() { /* open settings */ })
    menu.AddSeparator()
    menu.AddCheckbox("Start at Login", false, func() { /* toggle */ })
    menu.AddSeparator()
    menu.Add("Quit", func() { tray.Remove(); /* exit app */ })

    tray.SetMenu(menu)

    // Event handlers
    tray.OnClick(func() { /* toggle main window */ })

    // Show the tray icon
    tray.Show()

    // Run your application event loop...
    select {} // placeholder
}
```

## API Reference

### SystemTray

```go
// Create a new system tray icon
tray := systray.New()

// Icon management
tray.SetIcon(pngBytes []byte)          // Set tray icon (PNG format)
tray.SetDarkModeIcon(pngBytes []byte)  // Auto-switch in dark mode (Windows)
tray.SetTemplateIcon(pngBytes []byte)  // macOS template image (monochrome)

// Text
tray.SetTooltip(text string)           // Hover tooltip (Windows/Linux)
tray.SetLabel(text string)             // Text label next to icon (macOS only)

// Menu
tray.SetMenu(menu *Menu)               // Attach context menu

// Events
tray.OnClick(fn func())                // Left click handler
tray.OnDoubleClick(fn func())          // Double click handler
tray.OnRightClick(fn func())           // Right click handler

// Notifications
tray.ShowNotification(title, message string)  // OS-level notification

// Visibility
tray.Show()                            // Show tray icon
tray.Hide()                            // Hide tray icon
tray.Remove()                          // Remove and cleanup

// Position (for window placement near tray)
x, y, w, h := tray.Bounds()           // Tray icon screen position
```

All setter methods return `*SystemTray` for fluent chaining:
```go
tray.SetIcon(icon).SetTooltip("Ready").SetMenu(menu).Show()
```

### Menu

```go
menu := systray.NewMenu()

menu.Add("Label", onClick)                          // Normal item
menu.AddCheckbox("Toggle", checked, onChange)        // Checkbox item
menu.AddSeparator()                                 // Visual separator
menu.AddSubmenu("More", submenu)                    // Nested submenu
menu.AddWithIcon("Save", iconPNG, onClick)          // Item with icon
```

### Multiple Trays

```go
// Each tray is independent with its own icon, menu, and handlers
mainTray := systray.New().SetIcon(appIcon).SetMenu(mainMenu).Show()
statusTray := systray.New().SetIcon(statusIcon).SetTooltip("Status: OK").Show()
```

## Icon Guidelines

| Platform | Recommended Size | Format | Notes |
|:---------|:----------------|:-------|:------|
| **Windows** | 16x16, 32x32 | PNG | Provide both sizes for standard and HiDPI |
| **macOS** | 22x22, 44x44 (@2x) | PNG | Must be monochrome (template) for proper theme adaptation |
| **Linux** | 22x22, 24x24 | PNG | SNI spec recommends 22x22 |

**Input format:** PNG bytes (`[]byte`). The library handles conversion to native format (HICON, NSImage, ARGB pixmap) internally.

For macOS, use `SetTemplateIcon()` with a **monochrome** PNG (only alpha channel matters). The system automatically adjusts the icon color for light/dark menu bar.

## Architecture

```
systray.New()  →  SystemTray (public API)
                       │
                  PlatformTray (internal interface)
                       │
          ┌────────────┼────────────┐
          │            │            │
     Win32 impl   macOS impl   Linux impl
     Shell_Notify  NSStatusBar   D-Bus SNI
     IconW         NSStatusItem  StatusNotifierItem
```

Follows the Qt6 `QPlatformSystemTrayIcon` three-layer pattern. Each platform implementation is isolated in its own file with build constraints.

## Usage with gogpu

While systray is fully standalone, it integrates seamlessly with the [gogpu](https://github.com/gogpu/gogpu) application framework:

```go
import (
    "github.com/gogpu/gogpu"
    "github.com/gogpu/systray"
)

app := gogpu.NewApp(config)

// Create tray through the app (lifecycle managed automatically)
tray := systray.New()
tray.SetIcon(icon).SetMenu(menu).Show()

// Minimize to tray pattern
app.SetQuitBehavior(gogpu.QuitOnExplicitQuit)
app.OnClose(func() bool {
    app.Hide()       // hide window instead of closing
    return false     // reject close
})
tray.OnClick(func() {
    app.Show()       // restore window on tray click
})
```

## Comparison with Alternatives

| Feature | gogpu/systray | getlantern/systray | fyne-io/systray |
|:--------|:------------:|:------------------:|:---------------:|
| Pure Go (zero CGO) | **Yes** | No (CGO on macOS/Linux) | No (CGO on macOS/Linux) |
| Multiple trays | **Yes** | No (single global) | No (single global) |
| Dark mode icons | **Yes** | No | No |
| Template icons (macOS) | **Yes** | No | Yes |
| Nested menus | **Yes** | Yes | Yes |
| Notifications | **Yes** | No | No |
| Builder pattern | **Yes** | No | No |
| Window attachment | **Yes** | No | No |
| Wayland support | **Yes** (D-Bus SNI) | No | Partial |

## Contributing

We welcome contributions! Priority areas:

1. **Platform testing** — especially macOS and Linux (various DEs)
2. **Icon handling** — HiDPI, multi-resolution, SVG support
3. **Accessibility** — screen reader support for tray menus
4. **Examples** — real-world usage patterns

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Part of the GoGPU Ecosystem

systray is part of the [GoGPU](https://github.com/gogpu) ecosystem — a Pure Go GPU computing platform with 632K+ lines of code, including a WebGPU implementation, shader compiler, 2D graphics library, and GUI toolkit.

| Library | Purpose |
|:--------|:--------|
| [gogpu](https://github.com/gogpu/gogpu) | Application framework, windowing |
| [wgpu](https://github.com/gogpu/wgpu) | Pure Go WebGPU (Vulkan/Metal/DX12/GLES) |
| [naga](https://github.com/gogpu/naga) | Shader compiler (WGSL to SPIR-V/MSL/GLSL/HLSL) |
| [gg](https://github.com/gogpu/gg) | 2D graphics with GPU acceleration |
| [ui](https://github.com/gogpu/ui) | GUI toolkit (22+ widgets, 4 themes) |
| **[systray](https://github.com/gogpu/systray)** | **System tray (this library)** |

## License

MIT License — see [LICENSE](LICENSE) for details.
