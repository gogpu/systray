//go:build linux

package internal

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/png"
	"log/slog"
	"os"
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

// D-Bus well-known names and paths for the StatusNotifierItem protocol.
const (
	sniInterface     = "org.kde.StatusNotifierItem"
	sniPath          = "/StatusNotifierItem"
	menuInterface    = "com.canonical.dbusmenu"
	menuPath         = "/MenuBar"
	watcherInterface = "org.kde.StatusNotifierWatcher"
	watcherPath      = "/StatusNotifierWatcher"
	notifInterface   = "org.freedesktop.Notifications"
	notifPath        = "/org/freedesktop/Notifications"
)

// dbusPixmap represents an icon pixmap in the StatusNotifierItem protocol.
// D-Bus signature: (iiay) — width, height, ARGB32 big-endian pixel data.
type dbusPixmap struct {
	W    int32
	H    int32
	Data []byte
}

// dbusTooltip represents a tooltip in the StatusNotifierItem protocol.
// D-Bus signature: (sa(iiay)ss) — icon name, pixmaps, title, text.
type dbusTooltip struct {
	IconName string
	Pixmaps  []dbusPixmap
	Title    string
	Text     string
}

// linuxTray implements PlatformTray using D-Bus StatusNotifierItem (SNI).
// This is the standard tray protocol on modern Linux desktops (KDE, GNOME
// with AppIndicator, XFCE, Cinnamon, MATE, etc.) and replaces the legacy
// XEmbed system tray.
type linuxTray struct {
	conn      *dbus.Conn
	props     *prop.Properties
	menuProps *prop.Properties
	busName   string
	callbacks *Callbacks

	menu      *Menu
	menuItems map[int32]*MenuItem // id -> item for Event dispatch
	menuRev   uint32              // layout revision for dbusmenu

	iconPixmap []dbusPixmap // cached ARGB pixmap
	tooltip    string
	status     string // "Active" or "Passive"

	mu   sync.RWMutex
	quit chan struct{}
}

// NewPlatformTray creates a Linux system tray implementation via D-Bus SNI.
func NewPlatformTray(callbacks *Callbacks) PlatformTray {
	return &linuxTray{
		callbacks: callbacks,
		menuItems: make(map[int32]*MenuItem),
		status:    "Passive",
		quit:      make(chan struct{}),
	}
}

// Create connects to the session D-Bus, registers the SNI bus name, exports
// the StatusNotifierItem and dbusmenu objects, and registers with the
// StatusNotifierWatcher.
func (t *linuxTray) Create() error {
	conn, err := dbus.SessionBus()
	if err != nil {
		return fmt.Errorf("connect to session bus: %w", err)
	}
	t.conn = conn

	// Request a unique bus name following the SNI convention.
	t.busName = fmt.Sprintf("org.kde.StatusNotifierItem-%d-1", os.Getpid())
	reply, err := conn.RequestName(t.busName, dbus.NameFlagDoNotQueue)
	if err != nil {
		return fmt.Errorf("request bus name %s: %w", t.busName, err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return fmt.Errorf("bus name %s already taken (reply=%d)", t.busName, reply)
	}

	// Export StatusNotifierItem service object.
	sniSvc := &sniService{tray: t}
	if err := conn.Export(sniSvc, sniPath, sniInterface); err != nil {
		return fmt.Errorf("export SNI service: %w", err)
	}

	// Export SNI properties.
	sniProps, err := prop.Export(conn, sniPath, prop.Map{
		sniInterface: {
			"Category":      {Value: "ApplicationStatus", Writable: false, Emit: prop.EmitConst, Callback: nil},
			"Id":            {Value: t.busName, Writable: false, Emit: prop.EmitConst, Callback: nil},
			"Title":         {Value: "", Writable: false, Emit: prop.EmitTrue, Callback: nil},
			"Status":        {Value: "Passive", Writable: false, Emit: prop.EmitTrue, Callback: nil},
			"IconName":      {Value: "", Writable: false, Emit: prop.EmitTrue, Callback: nil},
			"IconPixmap":    {Value: []dbusPixmap{}, Writable: false, Emit: prop.EmitTrue, Callback: nil},
			"ToolTip":       {Value: dbusTooltip{}, Writable: false, Emit: prop.EmitTrue, Callback: nil},
			"Menu":          {Value: dbus.ObjectPath(menuPath), Writable: false, Emit: prop.EmitConst, Callback: nil},
			"ItemIsMenu":    {Value: true, Writable: false, Emit: prop.EmitConst, Callback: nil},
			"WindowId":      {Value: int32(0), Writable: false, Emit: prop.EmitConst, Callback: nil},
			"IconThemePath": {Value: "", Writable: false, Emit: prop.EmitConst, Callback: nil},
		},
	})
	if err != nil {
		return fmt.Errorf("export SNI properties: %w", err)
	}
	t.props = sniProps

	// Export SNI introspection.
	sniIntro := introspect.NewIntrospectable(&introspect.Node{
		Name: sniPath,
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name: sniInterface,
				Methods: []introspect.Method{
					{Name: "Activate", Args: []introspect.Arg{
						{Name: "x", Type: "i", Direction: "in"},
						{Name: "y", Type: "i", Direction: "in"},
					}},
					{Name: "SecondaryActivate", Args: []introspect.Arg{
						{Name: "x", Type: "i", Direction: "in"},
						{Name: "y", Type: "i", Direction: "in"},
					}},
					{Name: "ContextMenu", Args: []introspect.Arg{
						{Name: "x", Type: "i", Direction: "in"},
						{Name: "y", Type: "i", Direction: "in"},
					}},
					{Name: "Scroll", Args: []introspect.Arg{
						{Name: "delta", Type: "i", Direction: "in"},
						{Name: "orientation", Type: "s", Direction: "in"},
					}},
				},
				Signals: []introspect.Signal{
					{Name: "NewTitle"},
					{Name: "NewIcon"},
					{Name: "NewToolTip"},
					{Name: "NewStatus", Args: []introspect.Arg{
						{Name: "status", Type: "s"},
					}},
				},
				Properties: sniProps.Introspection(sniInterface),
			},
		},
	})
	if err := conn.Export(sniIntro, sniPath, "org.freedesktop.DBus.Introspectable"); err != nil {
		return fmt.Errorf("export SNI introspection: %w", err)
	}

	// Export dbusmenu service object.
	menuSvc := &dbusMenuService{tray: t}
	if err := conn.Export(menuSvc, menuPath, menuInterface); err != nil {
		return fmt.Errorf("export dbusmenu service: %w", err)
	}

	// Export dbusmenu properties.
	menuPropMap, err := prop.Export(conn, menuPath, prop.Map{
		menuInterface: {
			"Version":       {Value: uint32(3), Writable: false, Emit: prop.EmitConst, Callback: nil},
			"TextDirection": {Value: "ltr", Writable: false, Emit: prop.EmitConst, Callback: nil},
			"Status":        {Value: "normal", Writable: false, Emit: prop.EmitConst, Callback: nil},
			"IconThemePath": {Value: []string{}, Writable: false, Emit: prop.EmitConst, Callback: nil},
		},
	})
	if err != nil {
		return fmt.Errorf("export dbusmenu properties: %w", err)
	}
	t.menuProps = menuPropMap

	// Export dbusmenu introspection.
	menuIntro := introspect.NewIntrospectable(&introspect.Node{
		Name: menuPath,
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name: menuInterface,
				Methods: []introspect.Method{
					{Name: "GetLayout", Args: []introspect.Arg{
						{Name: "parentId", Type: "i", Direction: "in"},
						{Name: "recursionDepth", Type: "i", Direction: "in"},
						{Name: "propertyNames", Type: "as", Direction: "in"},
						{Name: "revision", Type: "u", Direction: "out"},
						{Name: "layout", Type: "(ia{sv}av)", Direction: "out"},
					}},
					{Name: "GetGroupProperties", Args: []introspect.Arg{
						{Name: "ids", Type: "ai", Direction: "in"},
						{Name: "propertyNames", Type: "as", Direction: "in"},
						{Name: "properties", Type: "a(ia{sv})", Direction: "out"},
					}},
					{Name: "Event", Args: []introspect.Arg{
						{Name: "id", Type: "i", Direction: "in"},
						{Name: "eventId", Type: "s", Direction: "in"},
						{Name: "data", Type: "v", Direction: "in"},
						{Name: "timestamp", Type: "u", Direction: "in"},
					}},
					{Name: "AboutToShow", Args: []introspect.Arg{
						{Name: "id", Type: "i", Direction: "in"},
						{Name: "needUpdate", Type: "b", Direction: "out"},
					}},
					{Name: "EventGroup", Args: []introspect.Arg{
						{Name: "events", Type: "a(isvu)", Direction: "in"},
						{Name: "idErrors", Type: "ai", Direction: "out"},
					}},
					{Name: "AboutToShowGroup", Args: []introspect.Arg{
						{Name: "ids", Type: "ai", Direction: "in"},
						{Name: "updatesNeeded", Type: "ai", Direction: "out"},
						{Name: "idErrors", Type: "ai", Direction: "out"},
					}},
				},
				Signals: []introspect.Signal{
					{Name: "LayoutUpdated", Args: []introspect.Arg{
						{Name: "revision", Type: "u"},
						{Name: "parent", Type: "i"},
					}},
					{Name: "ItemsPropertiesUpdated", Args: []introspect.Arg{
						{Name: "updatedProps", Type: "a(ia{sv})"},
						{Name: "removedProps", Type: "a(ias)"},
					}},
				},
				Properties: menuPropMap.Introspection(menuInterface),
			},
		},
	})
	if err := conn.Export(menuIntro, menuPath, "org.freedesktop.DBus.Introspectable"); err != nil {
		return fmt.Errorf("export dbusmenu introspection: %w", err)
	}

	// Register with the StatusNotifierWatcher.
	if err := t.registerWithWatcher(); err != nil {
		// Non-fatal: watcher may not be running yet.
		// The watcher restart goroutine will re-register when it starts.
		slog.Warn("systray: initial watcher registration failed (watcher may not be running)",
			"err", err)
	}

	// Start watching for watcher restarts.
	go t.watchWatcherRestart()

	return nil
}

// registerWithWatcher calls RegisterStatusNotifierItem on the StatusNotifierWatcher.
func (t *linuxTray) registerWithWatcher() error {
	obj := t.conn.Object(watcherInterface, watcherPath)
	call := obj.Call(watcherInterface+".RegisterStatusNotifierItem", 0, t.busName)
	if call.Err != nil {
		return fmt.Errorf("register status notifier item: %w", call.Err)
	}
	return nil
}

// watchWatcherRestart monitors D-Bus for the StatusNotifierWatcher restarting
// (e.g., when the DE panel restarts) and re-registers when it comes back.
func (t *linuxTray) watchWatcherRestart() {
	if err := t.conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.DBus"),
		dbus.WithMatchMember("NameOwnerChanged"),
		dbus.WithMatchArg(0, watcherInterface),
	); err != nil {
		slog.Warn("systray: failed to add NameOwnerChanged match", "err", err)
		return
	}

	ch := make(chan *dbus.Signal, 4)
	t.conn.Signal(ch)
	defer t.conn.RemoveSignal(ch)

	for {
		select {
		case sig, ok := <-ch:
			if !ok {
				return
			}
			if sig.Name != "org.freedesktop.DBus.NameOwnerChanged" {
				continue
			}
			// NameOwnerChanged args: name, old_owner, new_owner
			if len(sig.Body) < 3 {
				continue
			}
			newOwner, ok := sig.Body[2].(string)
			if !ok || newOwner == "" {
				// Watcher disappeared, not restarted.
				continue
			}
			// Watcher has a new owner — re-register.
			if err := t.registerWithWatcher(); err != nil {
				slog.Warn("systray: re-registration with watcher failed", "err", err)
			}

		case <-t.quit:
			return
		}
	}
}

// SetIcon converts PNG bytes to an ARGB32 pixmap and updates the SNI icon.
func (t *linuxTray) SetIcon(png []byte) error {
	if len(png) == 0 {
		return fmt.Errorf("empty icon data")
	}

	w, h, argb, err := pngToARGB(png)
	if err != nil {
		return fmt.Errorf("convert PNG to ARGB: %w", err)
	}

	t.mu.Lock()
	t.iconPixmap = []dbusPixmap{{W: int32(w), H: int32(h), Data: argb}}
	t.mu.Unlock()

	if t.props != nil {
		t.props.SetMust(sniInterface, "IconPixmap", t.iconPixmap)
		if err := t.emitSignal(sniInterface + ".NewIcon"); err != nil {
			slog.Warn("systray: emit NewIcon failed", "err", err)
		}
	}

	return nil
}

// SetTooltip updates the tooltip text displayed on hover.
func (t *linuxTray) SetTooltip(text string) error {
	t.mu.Lock()
	t.tooltip = text
	t.mu.Unlock()

	if t.props != nil {
		t.props.SetMust(sniInterface, "Title", text)
		t.props.SetMust(sniInterface, "ToolTip", dbusTooltip{
			IconName: "",
			Pixmaps:  nil,
			Title:    text,
			Text:     "",
		})
		if err := t.emitSignal(sniInterface + ".NewTitle"); err != nil {
			slog.Warn("systray: emit NewTitle failed", "err", err)
		}
	}

	return nil
}

// SetMenu attaches a context menu. The menu is exposed via the dbusmenu protocol.
func (t *linuxTray) SetMenu(menu *Menu) error {
	t.mu.Lock()
	t.menu = menu
	t.menuItems = make(map[int32]*MenuItem)
	if menu != nil {
		t.buildMenuItemMap(menu.Items, 1)
	}
	t.menuRev++
	rev := t.menuRev
	t.mu.Unlock()

	// Signal the DE that the menu layout has changed.
	if t.conn != nil {
		if err := t.conn.Emit(menuPath, menuInterface+".LayoutUpdated", rev, int32(0)); err != nil {
			slog.Warn("systray: emit LayoutUpdated failed", "err", err)
		}
	}

	return nil
}

// buildMenuItemMap recursively assigns IDs to menu items and stores them
// in the menuItems map for event dispatch. Returns the next available ID.
func (t *linuxTray) buildMenuItemMap(items []*MenuItem, nextID int32) int32 {
	for _, item := range items {
		id := nextID
		t.menuItems[id] = item
		nextID++
		if item.Type == MenuItemSubmenu && item.Submenu != nil {
			nextID = t.buildMenuItemMap(item.Submenu.Items, nextID)
		}
	}
	return nextID
}

// ShowNotification displays a desktop notification via org.freedesktop.Notifications.
func (t *linuxTray) ShowNotification(title, message string) error {
	if t.conn == nil {
		return fmt.Errorf("dbus connection not established")
	}

	obj := t.conn.Object(notifInterface, notifPath)
	call := obj.Call(notifInterface+".Notify", 0,
		"gogpu-systray",           // app_name
		uint32(0),                 // replaces_id
		"",                        // app_icon (empty — no icon)
		title,                     // summary
		message,                   // body
		[]string{},                // actions
		map[string]dbus.Variant{}, // hints
		int32(-1),                 // expire_timeout (-1 = server default)
	)
	if call.Err != nil {
		return fmt.Errorf("notify: %w", call.Err)
	}

	return nil
}

// Show makes the tray icon visible by setting the SNI status to "Active".
func (t *linuxTray) Show() error {
	t.mu.Lock()
	t.status = "Active"
	t.mu.Unlock()

	if t.props != nil {
		t.props.SetMust(sniInterface, "Status", "Active")
		if err := t.emitSignal(sniInterface+".NewStatus", "Active"); err != nil {
			slog.Warn("systray: emit NewStatus failed", "err", err)
		}
	}

	return nil
}

// Hide makes the tray icon invisible by setting the SNI status to "Passive".
func (t *linuxTray) Hide() error {
	t.mu.Lock()
	t.status = "Passive"
	t.mu.Unlock()

	if t.props != nil {
		t.props.SetMust(sniInterface, "Status", "Passive")
		if err := t.emitSignal(sniInterface+".NewStatus", "Passive"); err != nil {
			slog.Warn("systray: emit NewStatus failed", "err", err)
		}
	}

	return nil
}

// Bounds returns (0, 0, 0, 0) because the SNI protocol does not expose
// the physical position of the tray icon on screen.
func (t *linuxTray) Bounds() (int, int, int, int) {
	return 0, 0, 0, 0
}

// Run blocks the calling goroutine until Destroy is called.
// D-Bus message dispatch is handled internally by godbus on its own
// goroutine, so we simply wait for the quit signal.
func (t *linuxTray) Run() error {
	<-t.quit
	return nil
}

// Destroy releases all D-Bus resources: emits "Passive" status, releases
// the bus name, and closes the connection.
func (t *linuxTray) Destroy() {
	// Signal Passive status to the DE.
	if t.conn != nil && t.props != nil {
		t.props.SetMust(sniInterface, "Status", "Passive")
		if err := t.emitSignal(sniInterface+".NewStatus", "Passive"); err != nil {
			slog.Warn("systray: emit NewStatus (Passive) on destroy failed", "err", err)
		}
	}

	if t.conn != nil {
		if _, err := t.conn.ReleaseName(t.busName); err != nil {
			slog.Warn("systray: ReleaseName failed", "err", err)
		}
		if err := t.conn.Close(); err != nil {
			slog.Warn("systray: Close connection failed", "err", err)
		}
	}

	// Unblock Run().
	select {
	case <-t.quit:
		// Already closed.
	default:
		close(t.quit)
	}
}

// emitSignal emits a D-Bus signal on the SNI object path.
func (t *linuxTray) emitSignal(name string, values ...interface{}) error {
	return t.conn.Emit(sniPath, name, values...)
}

// --- StatusNotifierItem D-Bus service ---

// sniService implements the org.kde.StatusNotifierItem D-Bus methods.
type sniService struct {
	tray *linuxTray
}

// Activate is called when the user clicks the tray icon (primary action).
func (s *sniService) Activate(x, y int32) *dbus.Error {
	if s.tray.callbacks != nil && s.tray.callbacks.OnClick != nil {
		s.tray.callbacks.OnClick()
	}
	return nil
}

// SecondaryActivate is called on middle-click or similar secondary activation.
func (s *sniService) SecondaryActivate(x, y int32) *dbus.Error {
	if s.tray.callbacks != nil && s.tray.callbacks.OnDoubleClick != nil {
		s.tray.callbacks.OnDoubleClick()
	}
	return nil
}

// ContextMenu is called when the user right-clicks the tray icon.
// The menu is typically handled via the dbusmenu protocol, but some
// implementations also call this method.
func (s *sniService) ContextMenu(x, y int32) *dbus.Error {
	if s.tray.callbacks != nil && s.tray.callbacks.OnRightClick != nil {
		s.tray.callbacks.OnRightClick()
	}
	return nil
}

// Scroll is called when the user scrolls over the tray icon.
func (s *sniService) Scroll(delta int32, orientation string) *dbus.Error {
	// Scroll is not exposed in the public API.
	return nil
}

// --- dbusmenu D-Bus service ---

// menuLayout represents a single menu item in the dbusmenu layout tree.
// D-Bus signature: (ia{sv}av) — id, properties, children.
type menuLayout struct {
	V0 int32                   // item ID (0 = root)
	V1 map[string]dbus.Variant // properties
	V2 []dbus.Variant          // children (each is another menuLayout)
}

// menuItemProps is used for GetGroupProperties response.
// D-Bus signature: (ia{sv}).
type menuItemProps struct {
	ID    int32
	Props map[string]dbus.Variant
}

// menuEvent represents a single event in EventGroup.
// D-Bus signature: (isvu).
type menuEvent struct {
	ID        int32
	EventID   string
	Data      dbus.Variant
	Timestamp uint32
}

// dbusMenuService implements the com.canonical.dbusmenu D-Bus methods.
type dbusMenuService struct {
	tray *linuxTray
}

// GetLayout returns the menu tree starting from parentID.
func (m *dbusMenuService) GetLayout(parentID int32, recursionDepth int32, propertyNames []string) (uint32, menuLayout, *dbus.Error) {
	m.tray.mu.RLock()
	defer m.tray.mu.RUnlock()

	rev := m.tray.menuRev
	layout := m.buildLayout(parentID, recursionDepth, 0)

	return rev, layout, nil
}

// buildLayout constructs the dbusmenu layout tree recursively.
func (m *dbusMenuService) buildLayout(id int32, maxDepth int32, currentDepth int32) menuLayout {
	if id == 0 {
		// Root node.
		rootProps := map[string]dbus.Variant{
			"children-display": dbus.MakeVariant("submenu"),
		}
		var children []dbus.Variant
		if maxDepth != 0 && m.tray.menu != nil {
			children = m.buildChildren(m.tray.menu.Items, 1, maxDepth, currentDepth+1)
		}
		return menuLayout{V0: 0, V1: rootProps, V2: children}
	}

	// Non-root: find the item by ID.
	item, ok := m.tray.menuItems[id]
	if !ok {
		return menuLayout{V0: id, V1: map[string]dbus.Variant{}, V2: nil}
	}

	props := m.itemProperties(item)
	var children []dbus.Variant
	if item.Type == MenuItemSubmenu && item.Submenu != nil && maxDepth != 0 {
		// Find the starting child ID for this submenu.
		childStartID := m.findChildStartID(id)
		if childStartID > 0 {
			children = m.buildChildren(item.Submenu.Items, childStartID, maxDepth, currentDepth+1)
		}
	}

	return menuLayout{V0: id, V1: props, V2: children}
}

// buildChildren creates dbus.Variant entries for a list of menu items.
func (m *dbusMenuService) buildChildren(items []*MenuItem, startID int32, maxDepth int32, currentDepth int32) []dbus.Variant {
	children := make([]dbus.Variant, 0, len(items))
	nextID := startID

	for _, item := range items {
		id := nextID
		nextID++

		props := m.itemProperties(item)
		var subChildren []dbus.Variant

		if item.Type == MenuItemSubmenu && item.Submenu != nil {
			if maxDepth < 0 || currentDepth < maxDepth {
				subChildren = m.buildChildren(item.Submenu.Items, nextID, maxDepth, currentDepth+1)
			}
			// Advance nextID past all submenu items.
			nextID = m.advancePastSubmenu(item.Submenu, nextID)
		}

		child := menuLayout{V0: id, V1: props, V2: subChildren}
		children = append(children, dbus.MakeVariant(child))
	}

	return children
}

// advancePastSubmenu calculates the next available ID after all items in a submenu.
func (m *dbusMenuService) advancePastSubmenu(menu *Menu, startID int32) int32 {
	nextID := startID
	for _, item := range menu.Items {
		nextID++
		if item.Type == MenuItemSubmenu && item.Submenu != nil {
			nextID = m.advancePastSubmenu(item.Submenu, nextID)
		}
	}
	return nextID
}

// findChildStartID finds the start ID of children for a submenu item.
func (m *dbusMenuService) findChildStartID(parentID int32) int32 {
	// The children of item with parentID start at parentID+1 in our
	// sequential numbering scheme (same as buildMenuItemMap).
	return parentID + 1
}

// itemProperties converts a MenuItem to dbusmenu properties.
func (m *dbusMenuService) itemProperties(item *MenuItem) map[string]dbus.Variant {
	props := make(map[string]dbus.Variant)

	switch item.Type {
	case MenuItemSeparator:
		props["type"] = dbus.MakeVariant("separator")

	case MenuItemCheckbox:
		props["label"] = dbus.MakeVariant(item.Label)
		props["toggle-type"] = dbus.MakeVariant("checkmark")
		if item.Checked {
			props["toggle-state"] = dbus.MakeVariant(int32(1))
		} else {
			props["toggle-state"] = dbus.MakeVariant(int32(0))
		}
		if item.Disabled {
			props["enabled"] = dbus.MakeVariant(false)
		}

	case MenuItemSubmenu:
		props["label"] = dbus.MakeVariant(item.Label)
		props["children-display"] = dbus.MakeVariant("submenu")
		if item.Disabled {
			props["enabled"] = dbus.MakeVariant(false)
		}

	default: // MenuItemNormal
		props["label"] = dbus.MakeVariant(item.Label)
		if item.Disabled {
			props["enabled"] = dbus.MakeVariant(false)
		}
		if len(item.Icon) > 0 {
			props["icon-data"] = dbus.MakeVariant(item.Icon)
		}
	}

	return props
}

// GetGroupProperties returns properties for a set of menu items.
func (m *dbusMenuService) GetGroupProperties(ids []int32, propertyNames []string) ([]menuItemProps, *dbus.Error) {
	m.tray.mu.RLock()
	defer m.tray.mu.RUnlock()

	result := make([]menuItemProps, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			result = append(result, menuItemProps{
				ID: 0,
				Props: map[string]dbus.Variant{
					"children-display": dbus.MakeVariant("submenu"),
				},
			})
			continue
		}
		item, ok := m.tray.menuItems[id]
		if !ok {
			continue
		}
		result = append(result, menuItemProps{
			ID:    id,
			Props: m.itemProperties(item),
		})
	}

	return result, nil
}

// Event handles a menu item activation event from the DE.
func (m *dbusMenuService) Event(id int32, eventID string, data dbus.Variant, timestamp uint32) *dbus.Error {
	if eventID != "clicked" {
		return nil
	}

	m.tray.mu.RLock()
	item, ok := m.tray.menuItems[id]
	m.tray.mu.RUnlock()

	if !ok {
		return nil
	}

	if item.OnClick != nil {
		item.OnClick()
	}

	return nil
}

// AboutToShow is called before a submenu is shown. Returns whether the
// layout needs updating.
func (m *dbusMenuService) AboutToShow(id int32) (bool, *dbus.Error) {
	return false, nil
}

// EventGroup handles a batch of menu events.
func (m *dbusMenuService) EventGroup(events []menuEvent) ([]int32, *dbus.Error) {
	var idErrors []int32
	for _, ev := range events {
		if err := m.Event(ev.ID, ev.EventID, ev.Data, ev.Timestamp); err != nil {
			idErrors = append(idErrors, ev.ID)
		}
	}
	if idErrors == nil {
		idErrors = []int32{}
	}
	return idErrors, nil
}

// AboutToShowGroup handles a batch of AboutToShow calls.
func (m *dbusMenuService) AboutToShowGroup(ids []int32) ([]int32, []int32, *dbus.Error) {
	// No updates needed for any items.
	return []int32{}, []int32{}, nil
}

// --- PNG to ARGB conversion ---

// pngToARGB decodes a PNG image and converts it to ARGB32 big-endian pixel
// data, as required by the StatusNotifierItem IconPixmap property.
// Each pixel is 4 bytes: A, R, G, B (network byte order / big-endian).
func pngToARGB(data []byte) (w, h int, argb []byte, err error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return 0, 0, nil, fmt.Errorf("decode PNG: %w", err)
	}

	bounds := img.Bounds()
	w = bounds.Dx()
	h = bounds.Dy()
	argb = make([]byte, w*h*4)

	idx := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			// RGBA() returns pre-multiplied 16-bit values.
			// Convert to 8-bit and write ARGB big-endian.
			binary.BigEndian.PutUint32(argb[idx:], (a>>8)<<24|(r>>8)<<16|(g>>8)<<8|(b>>8))
			idx += 4
		}
	}

	return w, h, argb, nil
}
