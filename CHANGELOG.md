# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.7] - 2026-08-06

### Fixed

- **macOS: `SetLabel`/`SetChecked`/`SetDisabled` no-op for submenu items.** `applyItemUpdate` used `[NSMenu itemWithTag:]` which only searches the root menu — items inside submenus were silently dropped. Now stores `darwin.ID` (NSMenuItem handle) directly in `nsItems` map during `buildNSMenu`, including submenu containers. O(1) lookup replaces O(n) tag search. By @nange. ([#23](https://github.com/gogpu/systray/issues/23), [#24](https://github.com/gogpu/systray/pull/24))

## [0.2.6] - 2026-08-05

### Fixed

- **macOS: `Run()` never returning after `Remove()`.** `CFRunLoopStop` (v0.2.4-v0.2.5) did not work — `[NSApp run]` only re-checks the stop flag after processing a real event, and a `CFRunLoopStop` wake returns no event. Fix by @nange: `destroyOnMainThread` now posts an `NSEventTypeApplicationDefined` event via `[NSApp postEvent:atStart:YES]` after `[NSApp stop:]`. The event wakes the loop, which sees the stop flag and exits. Verified on macOS 26 (Apple Silicon). Removes unused CoreFoundation `CFRunLoopStop` FFI. ([#14](https://github.com/gogpu/systray/issues/14), [#21](https://github.com/gogpu/systray/pull/21))

## [0.2.5] - 2026-08-05

### Fixed

- **macOS: `Run()` exit attempt.** Call `[NSApp stop:]` + `CFRunLoopStop()` directly from `Destroy()` before `performSelectorOnMainThread`. Did not fully resolve — see v0.2.6.

## [0.2.4] - 2026-08-05

### Fixed

- **Windows: `Run()` still not exiting after `Remove()`.** The v0.2.3 fix deleted the tray from `trayRegistry` before posting `WM_CLOSE`, so the `WM_DESTROY` handler couldn't find the tray and never called `PostQuitMessage`. Now `Destroy()` only posts `WM_CLOSE`; all cleanup (registry, HICON, HMENU, `PostQuitMessage`) happens in the `WM_DESTROY` handler on the correct thread. ([#14](https://github.com/gogpu/systray/issues/14))
- **macOS: `Run()` still not exiting after `Remove()`.** `[NSApp stop:]` sets a flag but `[NSApp run]` only checks it after processing an event. Added `CFRunLoopStop()` via CoreFoundation FFI to immediately wake the blocked run loop (GLFW/SDL/Qt pattern). ([#14](https://github.com/gogpu/systray/issues/14))

## [0.2.3] - 2026-08-05

### Fixed

- **Windows: `tray.Remove()` from background goroutine failed.** `DestroyWindow` must be called from the thread that owns the HWND. Replaced with `PostMessage(WM_CLOSE)` which is thread-safe — `DefWindowProc` handles `WM_CLOSE` → `DestroyWindow` → `WM_DESTROY` → `PostQuitMessage` on the correct thread. ([#14](https://github.com/gogpu/systray/issues/14))
- **macOS: `tray.Remove()` from background goroutine crashed (SIGTRAP).** All AppKit calls (`removeStatusItem`, `release`, `stop`) must execute on the main thread. `Destroy()` now dispatches via `performSelectorOnMainThread` using a nil sentinel in the pending updates channel. ([#14](https://github.com/gogpu/systray/issues/14))

## [0.2.2] - 2026-08-05

### Fixed

- **Windows: submenu clicks dispatched to wrong callback.** `TrackPopupMenu` returned per-HMENU positional IDs (1,2,3 per submenu level), causing submenu items to invoke root-menu callbacks. Replaced with globally unique command IDs + flat `cmdItems` dispatch map across all menu levels. ([#12](https://github.com/gogpu/systray/issues/12))
- **Windows: `SetLabel`/`SetChecked`/`SetDisabled` had no effect on submenu items.** `SetMenuItemInfoW` searched only the root HMENU. Now stores per-item HMENU handle in `itemHMenus` map for correct submenu targeting. ([#13](https://github.com/gogpu/systray/issues/13))
- **Windows: `tray.Remove()` left process running.** `Run()` never returned because `DestroyWindow` posts `WM_DESTROY` but `GetMessage` only exits on `WM_QUIT`. Added `PostQuitMessage(0)` in `WM_DESTROY` handler. ([#14](https://github.com/gogpu/systray/issues/14))

### Added

- `MenuItem.IsChecked()` and `MenuItem.IsDisabled()` — thread-safe getters for current state. ([#15](https://github.com/gogpu/systray/issues/15))

### Changed

- **deps:** goffi v0.6.2 → v0.6.3

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

[Unreleased]: https://github.com/gogpu/systray/compare/v0.2.7...HEAD
[0.2.7]: https://github.com/gogpu/systray/compare/v0.2.6...v0.2.7
[0.2.6]: https://github.com/gogpu/systray/compare/v0.2.5...v0.2.6
[0.2.5]: https://github.com/gogpu/systray/compare/v0.2.4...v0.2.5
[0.2.4]: https://github.com/gogpu/systray/compare/v0.2.3...v0.2.4
[0.2.3]: https://github.com/gogpu/systray/compare/v0.2.2...v0.2.3
[0.2.2]: https://github.com/gogpu/systray/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/gogpu/systray/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/gogpu/systray/compare/v0.1.2...v0.2.0
[0.1.2]: https://github.com/gogpu/systray/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/gogpu/systray/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/gogpu/systray/releases/tag/v0.1.0
