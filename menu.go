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

// MenuItem represents a single item in a context menu.
// Use SetLabel, SetChecked, SetDisabled, SetIcon for dynamic updates.
type MenuItem struct {
	impl *internal.MenuItem
}

// SetLabel changes the display text and updates the native menu item.
func (item *MenuItem) SetLabel(label string) {
	item.impl.SetLabel(label)
}

// SetChecked changes the checked state.
func (item *MenuItem) SetChecked(checked bool) {
	item.impl.SetChecked(checked)
}

// SetDisabled changes the enabled/disabled state.
func (item *MenuItem) SetDisabled(disabled bool) {
	item.impl.SetDisabled(disabled)
}

// SetIcon changes the menu item icon.
func (item *MenuItem) SetIcon(png []byte) {
	item.impl.SetIcon(png)
}

// Menu represents a context menu for a system tray icon.
type Menu struct {
	impl *internal.Menu
}

// NewMenu creates an empty context menu.
func NewMenu() *Menu {
	return &Menu{impl: internal.NewMenu()}
}

// Add appends a normal menu item and returns it for dynamic updates.
func (m *Menu) Add(label string, onClick func()) *MenuItem {
	item := m.impl.Add(label, onClick)
	return &MenuItem{impl: item}
}

// AddCheckbox appends a checkbox menu item and returns it for dynamic updates.
func (m *Menu) AddCheckbox(label string, checked bool, onClick func()) *MenuItem {
	item := m.impl.AddCheckbox(label, checked, onClick)
	return &MenuItem{impl: item}
}

// AddSeparator appends a visual separator.
func (m *Menu) AddSeparator() *Menu {
	m.impl.AddSeparator()
	return m
}

// AddSubmenu appends a nested submenu and returns the item for dynamic updates.
func (m *Menu) AddSubmenu(label string, submenu *Menu) *MenuItem {
	item := m.impl.AddSubmenu(label, submenu.impl)
	return &MenuItem{impl: item}
}

// AddWithIcon appends a normal menu item with a PNG icon.
func (m *Menu) AddWithIcon(label string, icon []byte, onClick func()) *MenuItem {
	item := m.impl.AddWithIcon(label, icon, onClick)
	return &MenuItem{impl: item}
}
