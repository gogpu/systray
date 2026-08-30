package internal

import "sync/atomic"

// TrayID uniquely identifies a system tray icon. Zero is invalid.
type TrayID uint32

var nextTrayID atomic.Uint32

// NewTrayID returns a new unique tray identifier.
func NewTrayID() TrayID {
	return TrayID(nextTrayID.Add(1))
}

// Tray holds core tray state shared across platforms.
type Tray struct {
	ID           TrayID
	Platform     PlatformTray
	Callbacks    Callbacks
	Icon         []byte
	DarkModeIcon []byte
	TemplateIcon []byte
	Tooltip      string
	AppName      string
	Menu         *Menu
	Visible      bool
}

// NewTray creates a tray with a platform implementation.
func NewTray(platform PlatformTray) *Tray {
	return &Tray{
		ID:       NewTrayID(),
		Platform: platform,
	}
}

// SetIcon stores the icon and forwards to the platform.
func (t *Tray) SetIcon(png []byte) error {
	t.Icon = png
	return t.Platform.SetIcon(png)
}

// SetDarkModeIcon stores the dark mode icon variant and forwards it to the
// platform implementation. On Windows, this enables automatic icon switching
// when the system theme changes between dark and light mode.
func (t *Tray) SetDarkModeIcon(png []byte) error {
	t.DarkModeIcon = png

	if setter, ok := t.Platform.(interface{ SetDarkModeIcon([]byte) error }); ok {
		return setter.SetDarkModeIcon(png)
	}

	return nil
}

// SetTemplateIcon stores the macOS template icon and forwards it to the
// platform implementation. On macOS, template images are monochrome and the
// system automatically adjusts their appearance for the current menu bar theme.
// On other platforms this is a no-op.
func (t *Tray) SetTemplateIcon(png []byte) {
	t.TemplateIcon = png

	if setter, ok := t.Platform.(interface{ SetTemplateIcon([]byte) error }); ok {
		if err := setter.SetTemplateIcon(png); err != nil {
			// Non-fatal: icon may not render but tray remains functional.
			_ = err
		}
	}
}

// SetAppName stores the application name used for notifications on Linux
// (D-Bus org.freedesktop.Notifications app_name parameter). On Windows and
// macOS, the OS determines the application identity automatically.
func (t *Tray) SetAppName(name string) {
	t.AppName = name
	if setter, ok := t.Platform.(interface{ SetAppName(string) }); ok {
		setter.SetAppName(name)
	}
}

// SetTooltip stores the tooltip and forwards to the platform.
func (t *Tray) SetTooltip(text string) error {
	t.Tooltip = text
	return t.Platform.SetTooltip(text)
}

// SetMenu stores the menu, wires the MenuItemUpdater (if supported), and forwards to the platform.
func (t *Tray) SetMenu(menu *Menu) error {
	t.Menu = menu
	switch updater := t.Platform.(type) {
	case menuItemSnapshotUpdater:
		setMenuSnapshotUpdater(menu, updater)
	case MenuItemUpdater:
		SetMenuUpdater(menu, updater)
	}
	return t.Platform.SetMenu(menu)
}

// Show makes the tray icon visible.
func (t *Tray) Show() error {
	t.Visible = true
	return t.Platform.Show()
}

// Hide makes the tray icon invisible without removing it.
func (t *Tray) Hide() error {
	t.Visible = false
	return t.Platform.Hide()
}

// ShowNotification displays an OS-level notification.
func (t *Tray) ShowNotification(title, message string) error {
	return t.Platform.ShowNotification(title, message)
}

// Bounds returns the tray icon position on screen.
func (t *Tray) Bounds() (x, y, w, h int) {
	return t.Platform.Bounds()
}

// Run blocks the calling goroutine, pumping the platform message loop.
// Returns when Quit() is called or the platform loop exits.
func (t *Tray) Run() error {
	return t.Platform.Run()
}

// Remove destroys the tray icon and releases resources.
func (t *Tray) Remove() {
	t.Visible = false
	t.Platform.Destroy()
}
