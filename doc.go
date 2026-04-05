// Package systray provides a cross-platform system tray (notification area) library
// for Go applications. It supports Windows, macOS, and Linux with zero CGO dependencies.
//
// Key features:
//   - Multiple independent tray icons per application
//   - Context menus with nested submenus, checkboxes, and separators
//   - OS-level notifications (balloon tips, notification center, D-Bus)
//   - Dark mode icon switching and macOS template images
//   - Builder pattern for fluent API
//
// Platform implementations:
//   - Windows: Shell_NotifyIconW via golang.org/x/sys/windows
//   - macOS: NSStatusBar/NSStatusItem via go-webgpu/goffi ObjC runtime
//   - Linux: StatusNotifierItem (SNI) via godbus/dbus D-Bus protocol
//
// Quick start:
//
//	tray := systray.New()
//	tray.SetIcon(iconPNG).SetTooltip("My App").SetMenu(menu).Show()
//
// Part of the GoGPU ecosystem: https://github.com/gogpu
package systray
