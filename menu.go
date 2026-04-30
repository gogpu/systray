package systray

import (
	"github.com/gogpu/systray/internal"
)

// MenuItemType identifies the kind of menu item.
type MenuItemType = internal.MenuItemType

const (
	MenuItemNormal    = internal.MenuItemNormal
	MenuItemCheckbox  = internal.MenuItemCheckbox
	MenuItemSeparator = internal.MenuItemSeparator
	MenuItemSubmenu   = internal.MenuItemSubmenu
)

// Menu represents a context menu for a system tray icon.
type Menu struct {
	impl *internal.Menu
}

// NewMenu creates an empty context menu.
func NewMenu() *Menu {
	return &Menu{impl: internal.NewMenu()}
}

// Add appends a normal menu item.
func (m *Menu) Add(label string, onClick func()) *Menu {
	m.impl.Add(label, onClick)
	return m
}

// AddCheckbox appends a checkbox menu item.
func (m *Menu) AddCheckbox(label string, checked bool, onClick func()) *Menu {
	m.impl.AddCheckbox(label, checked, onClick)
	return m
}

// AddSeparator appends a visual separator.
func (m *Menu) AddSeparator() *Menu {
	m.impl.AddSeparator()
	return m
}

// AddSubmenu appends a nested submenu.
func (m *Menu) AddSubmenu(label string, submenu *Menu) *Menu {
	m.impl.AddSubmenu(label, submenu.impl)
	return m
}

// AddWithIcon appends a normal menu item with a PNG icon.
func (m *Menu) AddWithIcon(label string, icon []byte, onClick func()) *Menu {
	m.impl.AddWithIcon(label, icon, onClick)
	return m
}
