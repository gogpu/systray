# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.2] - 2026-07-12

### Changed

- **deps:** goffi v0.5.5 → v0.6.0 — `CallFunction` returns `(syscall.Errno, error)`, assembly-level errno capture. 10 call sites migrated in darwin/objc.go.
- **deps:** golang.org/x/sys v0.46.0 → v0.47.0

## [0.1.1] - 2026-06-25

### Changed

- **deps:** goffi v0.5.3 → v0.5.5 (CGO_ENABLED=1 coexistence, zero-alloc FFI, ABI-safe structs)
- **deps:** godbus/dbus v5.1.0 → v5.2.2

## [0.1.0] - 2026-04-30

### Added

- **Windows:** Shell_NotifyIconW system tray with context menus, balloon notifications, dark mode auto-switching, explorer crash recovery
- **macOS:** NSStatusBar/NSStatusItem via goffi ObjC runtime, template icons, NSMenu, NSUserNotification
- **Linux:** D-Bus StatusNotifierItem (SNI) via godbus, com.canonical.dbusmenu menus, org.freedesktop.Notifications, watcher re-registration
- Public API with builder pattern: SystemTray, Menu, MenuItem
- Multiple tray icons per application
- Click, double-click, right-click event handlers
- Run() message loop for standalone usage
- 72 tests, 84% public API coverage

[Unreleased]: https://github.com/gogpu/systray/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/gogpu/systray/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/gogpu/systray/releases/tag/v0.1.0
