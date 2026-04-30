package internal

// MenuItemType identifies the kind of menu item.
type MenuItemType int

const (
	MenuItemNormal MenuItemType = iota
	MenuItemCheckbox
	MenuItemSeparator
	MenuItemSubmenu
)

// MenuItem represents a single item in a context menu.
type MenuItem struct {
	Label    string
	Icon     []byte
	Type     MenuItemType
	Checked  bool
	Disabled bool
	Submenu  *Menu
	OnClick  func()
}

// Menu represents a context menu with a list of items.
type Menu struct {
	Items []*MenuItem
}

// NewMenu creates an empty menu.
func NewMenu() *Menu {
	return &Menu{}
}

// Add appends a normal menu item.
func (m *Menu) Add(label string, onClick func()) *Menu {
	m.Items = append(m.Items, &MenuItem{
		Label:   label,
		Type:    MenuItemNormal,
		OnClick: onClick,
	})
	return m
}

// AddCheckbox appends a checkbox menu item.
func (m *Menu) AddCheckbox(label string, checked bool, onClick func()) *Menu {
	m.Items = append(m.Items, &MenuItem{
		Label:   label,
		Type:    MenuItemCheckbox,
		Checked: checked,
		OnClick: onClick,
	})
	return m
}

// AddSeparator appends a visual separator.
func (m *Menu) AddSeparator() *Menu {
	m.Items = append(m.Items, &MenuItem{Type: MenuItemSeparator})
	return m
}

// AddSubmenu appends a submenu item.
func (m *Menu) AddSubmenu(label string, submenu *Menu) *Menu {
	m.Items = append(m.Items, &MenuItem{
		Label:   label,
		Type:    MenuItemSubmenu,
		Submenu: submenu,
	})
	return m
}

// AddWithIcon appends a normal menu item with an icon.
func (m *Menu) AddWithIcon(label string, icon []byte, onClick func()) *Menu {
	m.Items = append(m.Items, &MenuItem{
		Label:   label,
		Icon:    icon,
		Type:    MenuItemNormal,
		OnClick: onClick,
	})
	return m
}
