# Roadmap

> **Pure Go system tray library for Windows, macOS, and Linux**

---

## Current State: v0.1.0

✅ **All three platforms implemented:**

- **Windows** — Shell_NotifyIconW, context menus, balloon notifications, dark mode auto-switching, explorer crash recovery
- **macOS** — NSStatusBar/NSStatusItem via goffi ObjC runtime, template icons, NSMenu, NSUserNotification
- **Linux** — D-Bus StatusNotifierItem (SNI) via godbus, com.canonical.dbusmenu, org.freedesktop.Notifications, watcher re-registration
- **Public API** — builder pattern, multiple trays, click/doubleclick/rightclick handlers
- **72 tests**, 84% coverage on public API

### Recent Highlights

| Version | Date | Key Changes |
|---------|------|-------------|
| **v0.1.0** | 2026-04-30 | Initial release — all 3 platforms, menus, notifications, dark mode, 72 tests |

---

## Upcoming

### v0.2.0 — Platform Polish

- [ ] macOS: UNUserNotificationCenter (modern notifications API, macOS 11+)
- [ ] macOS: test on Apple Silicon (M1/M2/M3/M4) + Intel
- [ ] Linux: test on KDE Plasma, GNOME + AppIndicator extension, XFCE, Sway/waybar
- [ ] Linux: X11 XEmbed fallback for legacy DEs without SNI
- [ ] Windows: GUID-based icon identification (persistent across app restarts)
- [ ] Windows: balloon notification callbacks (click on balloon)
- [ ] SVG icon support (render to PNG at correct size)
- [ ] HiDPI icon handling (provide @1x and @2x)

### v0.3.0 — gogpu Integration

- [ ] `gogpu.App.NewSystemTray()` — lifecycle-managed tray within gogpu app
- [ ] Window attachment — click tray to toggle window near tray position
- [ ] Minimize-to-tray pattern — `SetQuitBehavior(QuitOnExplicitQuit)`
- [ ] Shared message loop — tray events within gogpu event loop (no separate Run())

### v0.4.0 — Advanced Features

- [ ] Notification actions (buttons, reply field)
- [ ] Notification images/icons
- [ ] Menu item icons on all platforms
- [ ] Dynamic menu updates (add/remove items at runtime)
- [ ] Accessibility — screen reader support for tray menus
- [ ] Tray icon animation (rotating/pulsing for attention)

### v1.0.0 — Stable API

- [ ] API freeze
- [ ] awesome-go submission
- [ ] Comprehensive documentation
- [ ] 90%+ coverage
- [ ] Security audit

---

## Architecture

```
systray.go / menu.go           Public API (delegation wrappers)
internal/tray.go                Core state management
internal/platform.go            PlatformTray interface
internal/platform_windows.go    Win32 Shell_NotifyIconW
internal/platform_darwin.go     macOS NSStatusBar via goffi
internal/platform_linux.go      D-Bus StatusNotifierItem
internal/darwin/objc.go         ObjC runtime wrapper
```

Follows Qt6 `QPlatformSystemTrayIcon` three-layer pattern (ADR-011).

---

## Part of the GoGPU Ecosystem

| Library | Purpose |
|:--------|:--------|
| [gogpu](https://github.com/gogpu/gogpu) | Application framework, windowing |
| [wgpu](https://github.com/gogpu/wgpu) | Pure Go WebGPU |
| [naga](https://github.com/gogpu/naga) | Shader compiler |
| [gg](https://github.com/gogpu/gg) | 2D graphics |
| [ui](https://github.com/gogpu/ui) | GUI toolkit |
| **[systray](https://github.com/gogpu/systray)** | **System tray (this library)** |
