package systray

import (
	"github.com/gogpu/systray/internal"
)

// TrayID uniquely identifies a system tray icon. Zero is invalid.
type TrayID uint32

// SystemTray represents a system tray icon with context menu.
// Create with New(). All setter methods return *SystemTray for chaining.
type SystemTray struct {
	impl *internal.Tray
}

// New creates a new system tray icon.
func New() *SystemTray {
	t := &SystemTray{}
	t.impl = &internal.Tray{
		ID: internal.NewTrayID(),
	}
	// Pass pointer to impl.Callbacks so platform sees updates from OnClick/OnDoubleClick/OnRightClick.
	platform := internal.NewPlatformTray(&t.impl.Callbacks)
	t.impl.Platform = platform
	if err := platform.Create(); err != nil {
		return t
	}
	return t
}

// ID returns the unique identifier for this tray icon.
func (t *SystemTray) ID() TrayID {
	return TrayID(t.impl.ID)
}

// SetIcon sets the tray icon from PNG bytes.
func (t *SystemTray) SetIcon(png []byte) *SystemTray {
	_ = t.impl.SetIcon(png)
	return t
}

// SetDarkModeIcon sets an alternative icon for dark mode (Windows).
func (t *SystemTray) SetDarkModeIcon(png []byte) *SystemTray {
	t.impl.SetDarkModeIcon(png)
	return t
}

// SetTemplateIcon sets a macOS template image (monochrome, adapts to theme).
func (t *SystemTray) SetTemplateIcon(png []byte) *SystemTray {
	t.impl.SetTemplateIcon(png)
	return t
}

// SetTooltip sets the hover tooltip text.
func (t *SystemTray) SetTooltip(text string) *SystemTray {
	_ = t.impl.SetTooltip(text)
	return t
}

// SetMenu attaches a context menu to the tray icon.
func (t *SystemTray) SetMenu(menu *Menu) *SystemTray {
	_ = t.impl.SetMenu(menu.impl)
	return t
}

// OnClick registers a left-click handler.
func (t *SystemTray) OnClick(fn func()) *SystemTray {
	t.impl.Callbacks.OnClick = fn
	return t
}

// OnDoubleClick registers a double-click handler.
func (t *SystemTray) OnDoubleClick(fn func()) *SystemTray {
	t.impl.Callbacks.OnDoubleClick = fn
	return t
}

// OnRightClick registers a right-click handler.
func (t *SystemTray) OnRightClick(fn func()) *SystemTray {
	t.impl.Callbacks.OnRightClick = fn
	return t
}

// ShowNotification displays an OS-level notification.
func (t *SystemTray) ShowNotification(title, message string) *SystemTray {
	_ = t.impl.ShowNotification(title, message)
	return t
}

// Show makes the tray icon visible.
func (t *SystemTray) Show() *SystemTray {
	_ = t.impl.Show()
	return t
}

// Hide makes the tray icon invisible without removing it.
func (t *SystemTray) Hide() *SystemTray {
	_ = t.impl.Hide()
	return t
}

// Run blocks the calling goroutine, pumping the platform message loop.
// Call from main() after Show(). Returns when Quit() is called.
func (t *SystemTray) Run() error {
	return t.impl.Run()
}

// Remove destroys the tray icon and releases all resources.
func (t *SystemTray) Remove() {
	t.impl.Remove()
}

// Bounds returns the tray icon's screen position (x, y, width, height).
func (t *SystemTray) Bounds() (x, y, w, h int) {
	return t.impl.Bounds()
}
