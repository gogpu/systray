package internal

import (
	"sync"
	"sync/atomic"
)

// MenuItemType identifies the kind of menu item.
type MenuItemType int

const (
	MenuItemNormal MenuItemType = iota
	MenuItemCheckbox
	MenuItemSeparator
	MenuItemSubmenu
)

var nextMenuItemID atomic.Uint32

func newMenuItemID() uint32 {
	return nextMenuItemID.Add(1)
}

// MenuItem represents a single item in a context menu.
type MenuItem struct {
	Label    string
	Icon     []byte
	Type     MenuItemType
	Checked  bool
	Disabled bool
	Submenu  *Menu
	OnClick  func()

	id      uint32                  // unique ID, assigned at creation
	mu      sync.Mutex              // protects mutable fields
	updater menuItemSnapshotUpdater // platform dispatch for live updates (nil until SetMenu)
}

// menuItemSnapshot is an immutable copy of the fields that may be changed by
// the dynamic MenuItem setters. Platform code uses snapshots so it never reads
// those fields concurrently with a setter. Icon owns its backing bytes.
type menuItemSnapshot struct {
	id       uint32
	label    string
	icon     []byte
	itemType MenuItemType
	checked  bool
	disabled bool
}

// ID returns the unique identifier for this menu item.
func (item *MenuItem) ID() uint32 { return item.id }

func (item *MenuItem) snapshotLocked() menuItemSnapshot {
	return menuItemSnapshot{
		id:       item.id,
		label:    item.Label,
		icon:     append([]byte(nil), item.Icon...),
		itemType: item.Type,
		checked:  item.Checked,
		disabled: item.Disabled,
	}
}

func (item *MenuItem) updateLocked() (menuItemSnapshotUpdater, menuItemSnapshot) {
	if item.updater == nil {
		return nil, menuItemSnapshot{}
	}
	return item.updater, item.snapshotLocked()
}

func (item *MenuItem) snapshot() menuItemSnapshot {
	item.mu.Lock()
	snapshot := item.snapshotLocked()
	item.mu.Unlock()
	return snapshot
}

// IsChecked returns the current checked state. Thread-safe.
func (item *MenuItem) IsChecked() bool {
	item.mu.Lock()
	v := item.Checked
	item.mu.Unlock()
	return v
}

// IsDisabled returns the current disabled state. Thread-safe.
func (item *MenuItem) IsDisabled() bool {
	item.mu.Lock()
	v := item.Disabled
	item.mu.Unlock()
	return v
}

// SetLabel changes the display text and dispatches the update to the native platform.
func (item *MenuItem) SetLabel(label string) {
	item.mu.Lock()
	item.Label = label
	u, snapshot := item.updateLocked()
	item.mu.Unlock()
	if u != nil {
		_ = u.updateItem(snapshot)
	}
}

// SetChecked changes the checked state and dispatches the update to the native platform.
func (item *MenuItem) SetChecked(checked bool) {
	item.mu.Lock()
	item.Checked = checked
	u, snapshot := item.updateLocked()
	item.mu.Unlock()
	if u != nil {
		_ = u.updateItem(snapshot)
	}
}

// SetDisabled changes the enabled/disabled state and dispatches the update to the native platform.
func (item *MenuItem) SetDisabled(disabled bool) {
	item.mu.Lock()
	item.Disabled = disabled
	u, snapshot := item.updateLocked()
	item.mu.Unlock()
	if u != nil {
		_ = u.updateItem(snapshot)
	}
}

// SetIcon changes the menu item icon and dispatches the update to the native platform.
func (item *MenuItem) SetIcon(png []byte) {
	item.mu.Lock()
	item.Icon = append([]byte(nil), png...)
	u, snapshot := item.updateLocked()
	item.mu.Unlock()
	if u != nil {
		_ = u.updateItem(snapshot)
	}
}

// setUpdater wires a platform dispatch for live updates. Called by SetMenu.
func (item *MenuItem) setUpdater(u menuItemSnapshotUpdater) {
	item.mu.Lock()
	item.updater = u
	item.mu.Unlock()
}

// SetUpdater wires the legacy pointer-based platform dispatch. Built-in
// platforms are wired through the snapshot-based seam by Tray.SetMenu.
func (item *MenuItem) SetUpdater(u MenuItemUpdater) {
	if u == nil {
		item.setUpdater(nil)
		return
	}
	item.setUpdater(legacyMenuItemUpdater{updater: u, item: item})
}

// Menu represents a context menu with a list of items.
type Menu struct {
	Items []*MenuItem
}

// NewMenu creates an empty menu.
func NewMenu() *Menu {
	return &Menu{}
}

// Add appends a normal menu item and returns it for dynamic updates.
func (m *Menu) Add(label string, onClick func()) *MenuItem {
	item := &MenuItem{
		id:      newMenuItemID(),
		Label:   label,
		Type:    MenuItemNormal,
		OnClick: onClick,
	}
	m.Items = append(m.Items, item)
	return item
}

// AddCheckbox appends a checkbox menu item and returns it for dynamic updates.
func (m *Menu) AddCheckbox(label string, checked bool, onClick func()) *MenuItem {
	item := &MenuItem{
		id:      newMenuItemID(),
		Label:   label,
		Type:    MenuItemCheckbox,
		Checked: checked,
		OnClick: onClick,
	}
	m.Items = append(m.Items, item)
	return item
}

// AddSeparator appends a visual separator and returns the item.
func (m *Menu) AddSeparator() *MenuItem {
	item := &MenuItem{
		id:   newMenuItemID(),
		Type: MenuItemSeparator,
	}
	m.Items = append(m.Items, item)
	return item
}

// AddSubmenu appends a submenu item and returns it for dynamic updates.
func (m *Menu) AddSubmenu(label string, submenu *Menu) *MenuItem {
	item := &MenuItem{
		id:      newMenuItemID(),
		Label:   label,
		Type:    MenuItemSubmenu,
		Submenu: submenu,
	}
	m.Items = append(m.Items, item)
	return item
}

// AddWithIcon appends a normal menu item with an icon and returns it for dynamic updates.
func (m *Menu) AddWithIcon(label string, icon []byte, onClick func()) *MenuItem {
	item := &MenuItem{
		id:      newMenuItemID(),
		Label:   label,
		Icon:    append([]byte(nil), icon...),
		Type:    MenuItemNormal,
		OnClick: onClick,
	}
	m.Items = append(m.Items, item)
	return item
}

// SetMenuUpdater recursively wires the updater on all items in a menu tree.
func SetMenuUpdater(menu *Menu, u MenuItemUpdater) {
	if menu == nil {
		return
	}
	for _, item := range menu.Items {
		item.SetUpdater(u)
		if item.Type == MenuItemSubmenu && item.Submenu != nil {
			SetMenuUpdater(item.Submenu, u)
		}
	}
}

// legacyMenuItemUpdater preserves the original pointer-based extension seam,
// including canonical MenuItem identity. Built-in platforms use snapshots;
// custom legacy updaters must synchronize direct field reads themselves.
type legacyMenuItemUpdater struct {
	updater MenuItemUpdater
	item    *MenuItem
}

func (u legacyMenuItemUpdater) updateItem(menuItemSnapshot) error {
	return u.updater.UpdateItem(u.item)
}

func setMenuSnapshotUpdater(menu *Menu, u menuItemSnapshotUpdater) {
	if menu == nil {
		return
	}
	for _, item := range menu.Items {
		item.setUpdater(u)
		if item.Type == MenuItemSubmenu && item.Submenu != nil {
			setMenuSnapshotUpdater(item.Submenu, u)
		}
	}
}
