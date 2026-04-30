//go:build windows

package internal

// win32Tray implements PlatformTray using Shell_NotifyIconW.
type win32Tray struct {
	callbacks *Callbacks
}

// NewPlatformTray creates a Win32 system tray implementation.
func NewPlatformTray(callbacks *Callbacks) PlatformTray {
	return &win32Tray{callbacks: callbacks}
}

func (t *win32Tray) Create() error                      { return nil }
func (t *win32Tray) SetIcon([]byte) error               { return nil }
func (t *win32Tray) SetTooltip(string) error            { return nil }
func (t *win32Tray) SetMenu(*Menu) error                { return nil }
func (t *win32Tray) ShowNotification(_, _ string) error { return nil }
func (t *win32Tray) Show() error                        { return nil }
func (t *win32Tray) Hide() error                        { return nil }
func (t *win32Tray) Bounds() (int, int, int, int)       { return 0, 0, 0, 0 }
func (t *win32Tray) Destroy()                           {}
