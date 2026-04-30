# Changelog

## [0.1.0] - 2026-04-30

### Added

- **Windows:** Shell_NotifyIconW system tray with context menus, balloon notifications, dark mode auto-switching, explorer crash recovery
- **macOS:** NSStatusBar/NSStatusItem via goffi ObjC runtime, template icons, NSMenu, NSUserNotification
- **Linux:** D-Bus StatusNotifierItem (SNI) via godbus, com.canonical.dbusmenu menus, org.freedesktop.Notifications, watcher re-registration
- Public API with builder pattern: SystemTray, Menu, MenuItem
- Multiple tray icons per application
- Click, double-click, right-click event handlers
- Run() message loop for standalone usage
