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
hello := menu.Add("Hello", func() { fmt.Println("Hello!") })
menu.AddSeparator()
menu.Add("Quit", func() {
    tray.Remove()
    os.Exit(0)
})

tray.SetIcon(iconPNG).SetTooltip("My App").SetMenu(menu).Show()

// Dynamic updates from any goroutine:
hello.SetLabel("Hello World!")
hello.SetDisabled(true)

if err := tray.Run(); err != nil {
    fmt.Println("error:", err)
}
```

## Build & Test

```bash
go build ./...                              # build
go test ./...                               # test (86 tests)
golangci-lint run --timeout=5m              # lint
cd examples/basic && go run .               # run example
```

## Architecture

```
systray.go / menu.go           Public API (MenuItem returned for dynamic updates)
internal/tray.go                Core state management
internal/platform.go            PlatformTray + MenuItemUpdater interfaces
internal/platform_windows.go    Win32 Shell_NotifyIconW + SetMenuItemInfoW
internal/platform_darwin.go     macOS NSStatusBar via goffi + per-instance registry
internal/platform_linux.go      D-Bus StatusNotifierItem + ItemsPropertiesUpdated
internal/darwin/objc.go         ObjC runtime wrapper
```

Three-layer pattern (Qt6 QPlatformSystemTrayIcon): public API → platform interface → native implementation.

## Platform Details

| Platform | API | Dependency | Zero CGO |
|----------|-----|-----------|----------|
| Windows | Shell_NotifyIconW + SetMenuItemInfoW | golang.org/x/sys | Yes |
| macOS | NSStatusBar / NSStatusItem / NSMenuItem | github.com/go-webgpu/goffi | Yes |
| Linux | D-Bus StatusNotifierItem (SNI) + dbusmenu | github.com/godbus/dbus/v5 | Yes |

## Key Features

- Multiple independent tray icons per application (per-instance callback routing)
- Context menus: items, checkboxes, separators, submenus, icons
- **Dynamic menu updates:** `MenuItem.SetLabel()`, `SetChecked()`, `SetDisabled()`, `SetIcon()` — in-place native updates on all platforms
- OS-level notifications (balloon tips / notification center / D-Bus)
- Dark mode auto-switching (Windows) + template icons (macOS)
- Click, double-click, right-click handlers

## Menu API

`Menu.Add()`, `AddCheckbox()`, `AddSubmenu()`, `AddWithIcon()` return `*MenuItem` for dynamic updates. `AddSeparator()` returns `*Menu` for chaining.

```go
menu := systray.NewMenu()
status := menu.Add("Status: idle", nil)
check := menu.AddCheckbox("Auto-refresh", true, nil)
menu.AddSeparator()
sub := menu.AddSubmenu("Options", optionsMenu)
menu.Add("Quit", quitFn)

// Update at runtime:
status.SetLabel("Status: running")
check.SetChecked(false)
sub.SetDisabled(true)
```

## Community & Support

- GitHub: https://github.com/gogpu/systray
- Docs: https://pkg.go.dev/github.com/gogpu/systray
- Ecosystem: https://github.com/gogpu
- Sponsor: https://opencollective.com/gogpu
