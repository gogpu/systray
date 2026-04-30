package internal

// PlatformTray is the per-platform system tray implementation.
// Each platform (Win32, macOS, Linux) implements this interface.
type PlatformTray interface {
	Create() error
	SetIcon(png []byte) error
	SetTooltip(text string) error
	SetMenu(menu *Menu) error
	ShowNotification(title, message string) error
	Show() error
	Hide() error
	Bounds() (x, y, w, h int)
	Destroy()
}

// Callbacks holds event handlers set by the public API layer.
type Callbacks struct {
	OnClick       func()
	OnDoubleClick func()
	OnRightClick  func()
}
