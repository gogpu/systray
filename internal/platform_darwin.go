//go:build darwin

package internal

// darwinTray implements PlatformTray using NSStatusBar/NSStatusItem.
type darwinTray struct {
	callbacks *Callbacks
}

// NewPlatformTray creates a macOS system tray implementation.
func NewPlatformTray(callbacks *Callbacks) PlatformTray {
	return &darwinTray{callbacks: callbacks}
}

func (t *darwinTray) Create() error                      { return nil }
func (t *darwinTray) SetIcon([]byte) error               { return nil }
func (t *darwinTray) SetTooltip(string) error            { return nil }
func (t *darwinTray) SetMenu(*Menu) error                { return nil }
func (t *darwinTray) ShowNotification(_, _ string) error { return nil }
func (t *darwinTray) Show() error                        { return nil }
func (t *darwinTray) Hide() error                        { return nil }
func (t *darwinTray) Bounds() (int, int, int, int)       { return 0, 0, 0, 0 }
func (t *darwinTray) Destroy()                           {}
