# AGENTS.md — systray

> Pure Go system tray library. Win32/macOS/Linux, zero CGO.

## What is systray

systray provides cross-platform system tray (notification area) functionality: icon, tooltip, context menus with submenus/checkboxes, and OS-level notifications. Uses native APIs on each platform via Pure Go FFI — no C compiler required.

Part of the [GoGPU ecosystem](https://github.com/gogpu) — but fully standalone, usable in any Go application.

## Quick Start

```go
import (
    "fmt"
    "os"

    "github.com/gogpu/systray"
)

tray := systray.New()

menu := systray.NewMenu()
menu.Add("Hello", func() { fmt.Println("Hello!") })
menu.AddSeparator()
menu.Add("Quit", func() {
    tray.Remove()
    os.Exit(0)
})

tray.SetIcon(iconPNG).SetTooltip("My App").SetMenu(menu).Show()

if err := tray.Run(); err != nil {
    fmt.Println("error:", err)
}
```

## Build & Test

```bash
go build ./...                              # build
go test ./...                               # test
golangci-lint run --timeout=5m              # lint
cd examples/basic && go run .               # run example
```

## Architecture

```
systray.go / menu.go           Public API (builder pattern, fluent chaining)
internal/tray.go                Core state management
internal/platform.go            PlatformTray interface
internal/platform_windows.go    Win32 Shell_NotifyIconW
internal/platform_darwin.go     macOS NSStatusBar via goffi
internal/platform_linux.go      D-Bus StatusNotifierItem
internal/darwin/objc.go         ObjC runtime wrapper
```

Three-layer pattern (Qt6 QPlatformSystemTrayIcon): public API → platform interface → native implementation.

## Platform Details

| Platform | API | Dependency | Zero CGO |
|----------|-----|-----------|----------|
| Windows | Shell_NotifyIconW (shell32.dll) | golang.org/x/sys | Yes |
| macOS | NSStatusBar / NSStatusItem | github.com/go-webgpu/goffi | Yes |
| Linux | D-Bus StatusNotifierItem (SNI) | github.com/godbus/dbus/v5 | Yes |

## Key Features

- Multiple independent tray icons per application
- Context menus: items, checkboxes, separators, submenus, icons
- OS-level notifications (balloon tips / notification center / D-Bus)
- Dark mode auto-switching (Windows) + template icons (macOS)
- Click, double-click, right-click handlers
- Builder pattern API with fluent chaining

## Community & Support

- GitHub: https://github.com/gogpu/systray
- Docs: https://pkg.go.dev/github.com/gogpu/systray
- Ecosystem: https://github.com/gogpu
- Sponsor: https://opencollective.com/gogpu
