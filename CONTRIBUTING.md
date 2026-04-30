# Contributing to systray

Thank you for your interest in contributing to gogpu/systray! This document covers how to build, test, and submit changes.

## Prerequisites

- **Go 1.25+** ([download](https://go.dev/dl/))
- **golangci-lint** (`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`)
- Platform SDK for the target OS (see [Platform Testing](#platform-testing) below)

## Building

```bash
go build ./...
```

## Running the Example

```bash
cd examples/basic
go run .
```

Right-click the tray icon to see the context menu. Toggle your OS dark/light mode to observe icon switching.

## Running Tests

```bash
go test ./...
```

## Code Style

- Run `go fmt ./...` before every commit. CI enforces this.
- Run `golangci-lint run --timeout=5m` and fix all issues.
- Follow standard Go naming conventions (`ID`, `URL`, `HTTP` are uppercase).
- Handle every error or explicitly ignore with `_ =` and a comment explaining why.
- Exported types and functions must have doc comments.

## Pull Request Workflow

1. Fork the repository and create a feature branch:
   ```bash
   git checkout -b feat/my-feature
   ```
2. Make your changes. Keep commits focused and well-described.
3. Verify locally:
   ```bash
   go fmt ./...
   go build ./...
   go test ./...
   golangci-lint run --timeout=5m
   ```
4. Push and open a pull request against `main`.
5. Wait for CI to pass. All checks must be green before merge.

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/):
```
feat: add SVG icon support
fix(linux): handle missing SNI host gracefully
docs: update notification examples
```

## Platform Testing

Each platform backend lives in `internal/platform_<os>.go` with build constraints. Testing on your native OS is the most valuable contribution.

**Windows** -- No extra dependencies. Uses `golang.org/x/sys/windows` for Win32 API calls (`Shell_NotifyIconW`, window message loop). Verify dark mode switching by toggling Settings > Personalization > Colors.

**macOS** -- Requires a Mac. Uses `github.com/go-webgpu/goffi` for ObjC runtime calls (`NSStatusBar`, `NSStatusItem`). Test with both light and dark menu bar appearances. Verify template icon rendering with `SetTemplateIcon()`.

**Linux** -- Requires a desktop environment with StatusNotifierItem (SNI) support: GNOME (with AppIndicator extension), KDE Plasma, XFCE. Uses `github.com/godbus/dbus/v5` for the D-Bus SNI protocol. Test on multiple DEs if possible.

## Architecture

```
systray.go / menu.go          Public API (SystemTray, Menu)
internal/tray.go               Core state (Tray struct)
internal/platform.go            PlatformTray interface
internal/platform_windows.go    Win32 Shell_NotifyIconW
internal/platform_darwin.go     macOS NSStatusBar via goffi
internal/platform_linux.go      D-Bus StatusNotifierItem
```

The design follows the Qt6 `QPlatformSystemTrayIcon` three-layer pattern: a public API layer delegates to a platform-specific implementation via the `PlatformTray` interface.

## Priority Areas

1. **Platform testing** -- especially macOS and Linux across desktop environments
2. **Icon handling** -- HiDPI, multi-resolution, SVG support
3. **Accessibility** -- screen reader support for tray menus
4. **Examples** -- real-world usage patterns (minimize-to-tray, status indicator)

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
