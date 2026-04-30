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
  <a href="https://app.codecov.io/gh/gogpu/systray"><img src="https://codecov.io/gh/gogpu/systray/branch/main/graph/badge.svg" alt="Coverage"></a>
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
- **Dark mode** — automatic icon switching for light/dark themes (Windows)
- **Template icons** — macOS-native monochrome icons that adapt to system theme
- **Builder pattern** — fluent API for clean, readable code
- **Message loop** — built-in `Run()` blocks and pumps the platform event loop
- **Standalone** — no dependency on gogpu framework. Use in any Go application

## Platform Implementation

| Platform | API | Dependency | Status |
|:---------|:----|:-----------|:------:|
| **Windows** | `Shell_NotifyIconW` (shell32.dll) | `golang.org/x/sys/windows` | Implemented |
| **macOS** | `NSStatusBar` / `NSStatusItem` (AppKit) | `github.com/go-webgpu/goffi` | Implemented |
| **Linux** | StatusNotifierItem (D-Bus SNI) | `github.com/godbus/dbus/v5` | Implemented |

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
    "fmt"
    "os"

    "github.com/gogpu/systray"
)

func main() {
    tray := systray.New()

    // Build context menu
    menu := systray.NewMenu()
    menu.Add("Hello", func() { fmt.Println("Hello clicked!") })
    menu.Add("Show Notification", func() {
        tray.ShowNotification("My App", "Hello from systray!")
    })
    menu.AddSeparator()
    menu.AddCheckbox("Check me", false, func() { fmt.Println("Toggled") })
    menu.AddSeparator()
    menu.Add("Quit", func() {
        tray.Remove()
        os.Exit(0)
    })

    // Configure and show
    tray.SetIcon(iconPNG).
        SetTooltip("My Application").
        SetMenu(menu)
    tray.OnClick(func() { fmt.Println("Left click!") })
    tray.Show()

    // Run the platform message loop (blocks until Quit)
    if err := tray.Run(); err != nil {
        fmt.Println("error:", err)
    }
}
```

## API Reference

### SystemTray

```go
// Create and lifecycle
tray := systray.New()              // Create a new system tray icon
tray.Show()                        // Show tray icon
tray.Hide()                        // Hide tray icon (without removing)
tray.Run()                         // Block and pump the platform message loop
tray.Remove()                      // Destroy tray icon and release resources

// Icon management
tray.SetIcon(png []byte)           // Set tray icon (PNG format)
tray.SetDarkModeIcon(png []byte)   // Auto-switch in dark mode (Windows)
tray.SetTemplateIcon(png []byte)   // macOS template image (monochrome)

// Text and menu
tray.SetTooltip(text string)       // Hover tooltip
tray.SetMenu(menu *Menu)           // Attach context menu

// Events
tray.OnClick(fn func())            // Left click handler
tray.OnDoubleClick(fn func())      // Double click handler
tray.OnRightClick(fn func())       // Right click handler

// Notifications
tray.ShowNotification(title, message string)  // OS-level notification

// Position (for window placement near tray)
x, y, w, h := tray.Bounds()       // Tray icon screen position
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

All `Menu` methods return `*Menu` for chaining.

### Multiple Trays

```go
// Each tray is independent with its own icon, menu, and handlers
mainTray := systray.New().SetIcon(appIcon).SetMenu(mainMenu).Show()
statusTray := systray.New().SetIcon(statusIcon).SetTooltip("Status: OK").Show()
```

## Dark Mode

systray supports automatic icon switching based on the system theme.

**Windows** — Use `SetDarkModeIcon()` to provide an alternative icon for dark mode. The library detects theme changes via `WM_SETTINGCHANGE` with `"ImmersiveColorSet"` and switches icons automatically:

```go
tray.SetIcon(lightIcon).SetDarkModeIcon(darkIcon)
```

**macOS** — Use `SetTemplateIcon()` with a monochrome PNG. macOS renders template images with the correct color for the current menu bar appearance (light or dark). Only the alpha channel matters:

```go
tray.SetTemplateIcon(monochromeIcon)
```

**Linux** — The SNI protocol delivers the icon pixmap to the desktop environment, which handles theme adaptation. No special API is needed.

## Notifications

`ShowNotification` sends an OS-level notification from the tray icon:

```go
tray.ShowNotification("Update Available", "Version 2.0 is ready to install.")
```

| Platform | Mechanism | Notes |
|:---------|:----------|:------|
| **Windows** | Balloon tip (`Shell_NotifyIconW` + `NIF_INFO`) | Appears near the tray icon |
| **macOS** | `NSUserNotification` / Notification Center | Requires notification permission on macOS 13+ |
| **Linux** | `org.freedesktop.Notifications` D-Bus | Works on GNOME, KDE, XFCE, and other FreeDesktop-compliant DEs |

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
systray.New()  ->  SystemTray (public API)
                       |
                  PlatformTray (internal interface)
                       |
          +------------+------------+
          |            |            |
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
| Menu item icons | **Yes** | No | No |
| Notifications | **Yes** | No | No |
| Builder pattern | **Yes** | No | No |
| Built-in message loop | **Yes** | Yes | Yes |
| Wayland support | **Yes** (D-Bus SNI) | No | Partial |

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Part of the GoGPU Ecosystem

systray is part of the [GoGPU](https://github.com/gogpu) ecosystem — 790K+ lines of Pure Go, zero CGO. A GPU computing platform with a WebGPU implementation, shader compiler, 2D graphics library, and GUI toolkit.

| Library | Purpose |
|:--------|:--------|
| [gogpu](https://github.com/gogpu/gogpu) | Application framework, windowing |
| [wgpu](https://github.com/gogpu/wgpu) | Pure Go WebGPU (Vulkan/Metal/DX12/GLES) |
| [naga](https://github.com/gogpu/naga) | Shader compiler (WGSL to SPIR-V/MSL/GLSL/HLSL/DXIL) |
| [gg](https://github.com/gogpu/gg) | 2D graphics with GPU acceleration |
| [ui](https://github.com/gogpu/ui) | GUI toolkit (22+ widgets, 4 themes) |
| **[systray](https://github.com/gogpu/systray)** | **System tray (this library)** |

## License

MIT License — see [LICENSE](LICENSE) for details.
