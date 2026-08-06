# Roadmap

> **Pure Go system tray library for Windows, macOS, and Linux**

---

## Current State: v0.2.0

All three platforms implemented and production-ready:

- **Windows** — Shell_NotifyIconW, context menus, balloon notifications, dark mode auto-switching, explorer crash recovery, dynamic menu updates via SetMenuItemInfoW
- **macOS** — NSStatusBar/NSStatusItem via goffi ObjC runtime, template icons, NSMenu, NSUserNotification, per-instance callback routing, dynamic menu updates via NSMenuItem setTitle/setState/setEnabled
- **Linux** — D-Bus StatusNotifierItem (SNI) via godbus, com.canonical.dbusmenu, org.freedesktop.Notifications, watcher re-registration, private D-Bus connection per tray, ItemsPropertiesUpdated for dynamic updates
- **Public API** — builder pattern, multiple trays, click/doubleclick/rightclick handlers, dynamic menu updates (SetLabel/SetChecked/SetDisabled/SetIcon)
- **86 tests**, 85% coverage on public API

### Release History

| Version | Date | Key Changes |
|---------|------|-------------|
| **v0.2.8** | 2026-08-06 | Windows submenu container UpdateItem: position-based lookup (PR #27 by @nange) |
| **v0.2.7** | 2026-08-06 | macOS submenu UpdateItem: nsItems map replaces itemWithTag (PR #24 by @nange) |
| **v0.2.6** | 2026-08-05 | macOS Run() exit: PostAppDefinedEvent (PR #21 by @nange) |
| **v0.2.5** | 2026-08-05 | macOS Run() exit attempt: stop+CFRunLoopStop |
| **v0.2.4** | 2026-08-05 | Fix Run() exit: Windows WM_DESTROY cleanup ordering, macOS CFRunLoopStop wake |
| **v0.2.3** | 2026-08-05 | Fix Destroy() from background goroutine (Windows PostMessage, macOS main thread dispatch) |
| **v0.2.2** | 2026-08-05 | Fix Windows submenu dispatch, UpdateItem on submenus, Run() exit; IsChecked/IsDisabled getters |
| **v0.2.1** | 2026-07-28 | Fix macOS crash on background goroutine menu updates (main thread dispatch) |
| **v0.2.0** | 2026-07-27 | Multi-tray fix (macOS/Linux), dynamic menu updates, breaking API (Add returns *MenuItem) |
| **v0.1.2** | 2026-07-12 | deps: goffi v0.6.0, x/sys v0.47.0 |
| **v0.1.1** | 2026-06-25 | deps: goffi v0.5.5, godbus v5.2.2 |
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
- [x] ~~Dynamic menu updates (add/remove items at runtime)~~ — shipped in v0.2.0
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

Follows Qt6 `QPlatformSystemTrayIcon` three-layer pattern.

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
