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

// SetDarkModeIcon stores the dark mode icon variant.
func (t *Tray) SetDarkModeIcon(png []byte) {
	t.DarkModeIcon = png
}

// SetTemplateIcon stores the macOS template icon.
func (t *Tray) SetTemplateIcon(png []byte) {
	t.TemplateIcon = png
}

// SetTooltip stores the tooltip and forwards to the platform.
func (t *Tray) SetTooltip(text string) error {
	t.Tooltip = text
	return t.Platform.SetTooltip(text)
}

// SetMenu stores the menu and forwards to the platform.
func (t *Tray) SetMenu(menu *Menu) error {
	t.Menu = menu
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
