# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.1] - 2026-07-28

### Fixed

- **macOS: crash on background goroutine menu updates.** `SetLabel()`, `SetChecked()`, `SetDisabled()`, `SetIcon()` called from a background goroutine crashed with `NSInternalInconsistencyException` ("modifying autolayout engine from background thread"). All AppKit mutations now dispatch to the main thread via `performSelectorOnMainThread:withObject:waitUntilDone:` (SDL pattern). ([#8 comment](https://github.com/gogpu/systray/issues/8#issuecomment-5101757207))

## [0.2.0] - 2026-07-27

### Added

- **Dynamic menu updates:** `MenuItem.SetLabel()`, `SetChecked()`, `SetDisabled()`, `SetIcon()` — update native menu items in-place without rebuilding the entire menu (Qt6 QAction pattern)
- **MenuItemUpdater interface** — optional platform interface for dispatching live menu updates
- **Windows:** `SetMenuItemInfoW` for in-place menu item updates (MIIM_STRING + MIIM_STATE)
- **macOS:** `NSMenuItem` setTitle/setState/setEnabled via `itemWithTag:` lookup
- **Linux:** D-Bus `ItemsPropertiesUpdated` signal for individual menu item property changes

### Fixed

- **macOS multi-tray:** replace global `activeTray` singleton with per-instance `trayRegistryMap` keyed by ObjC target — each tray now correctly receives its own callbacks ([#7](https://github.com/gogpu/systray/issues/7))
- **Linux multi-tray:** use `dbus.ConnectSessionBus()` (private connection per tray) instead of shared singleton, and unique `PID-{trayID}` bus name suffix per SNI spec ([#7](https://github.com/gogpu/systray/issues/7))

### Changed

- **Breaking:** `Menu.Add()`, `AddCheckbox()`, `AddSubmenu()`, `AddWithIcon()` now return `*MenuItem` instead of `*Menu` — enables dynamic updates on the returned item handle ([#8](https://github.com/gogpu/systray/issues/8))
- `Menu.AddSeparator()` still returns `*Menu` for chaining (separators don't need dynamic updates)
- **deps:** goffi v0.6.0 → v0.6.2

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

[Unreleased]: https://github.com/gogpu/systray/compare/v0.2.1...HEAD
[0.2.1]: https://github.com/gogpu/systray/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/gogpu/systray/compare/v0.1.2...v0.2.0
[0.1.2]: https://github.com/gogpu/systray/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/gogpu/systray/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/gogpu/systray/releases/tag/v0.1.0
