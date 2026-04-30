//go:build linux

package internal

// linuxTray implements PlatformTray using D-Bus StatusNotifierItem.
type linuxTray struct {
	callbacks *Callbacks
}

// NewPlatformTray creates a Linux system tray implementation via D-Bus SNI.
func NewPlatformTray(callbacks *Callbacks) PlatformTray {
	return &linuxTray{callbacks: callbacks}
}

func (t *linuxTray) Create() error                      { return nil }
func (t *linuxTray) SetIcon([]byte) error               { return nil }
func (t *linuxTray) SetTooltip(string) error            { return nil }
func (t *linuxTray) SetMenu(*Menu) error                { return nil }
func (t *linuxTray) ShowNotification(_, _ string) error { return nil }
func (t *linuxTray) Show() error                        { return nil }
func (t *linuxTray) Hide() error                        { return nil }
func (t *linuxTray) Bounds() (int, int, int, int)       { return 0, 0, 0, 0 }
func (t *linuxTray) Destroy()                           {}
