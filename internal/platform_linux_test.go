//go:build linux

package internal

import (
	"testing"

	"github.com/godbus/dbus/v5"
)

// testTray returns a linuxTray without a D-Bus session: conn stays nil,
// so SetMenu only builds its ID maps and never touches the bus.
func testTray(t *testing.T) *linuxTray {
	t.Helper()
	tr, ok := NewPlatformTray(&Callbacks{}).(*linuxTray)
	if !ok {
		t.Fatal("NewPlatformTray did not return a *linuxTray")
	}
	return tr
}

func layoutIDs(t *testing.T, l menuLayout) []int32 {
	t.Helper()
	ids := []int32{l.V0}
	for _, c := range l.V2 {
		var child menuLayout
		if err := c.Store(&child); err != nil {
			t.Fatalf("decoding child of item %d: %v", l.V0, err)
		}
		ids = append(ids, layoutIDs(t, child)...)
	}
	return ids
}

func TestSetMenuIDsAreMonotonic(t *testing.T) {
	tr := testTray(t)
	svc := &dbusMenuService{tray: tr}

	m1 := NewMenu()
	m1.Add("Header", nil).SetDisabled(true)
	m1.AddSeparator()
	m1.Add("Quit", nil)
	if err := tr.SetMenu(m1); err != nil {
		t.Fatal(err)
	}
	tr.mu.RLock()
	_, firstLayout, _ := svc.GetLayout(0, -1, nil)
	tr.mu.RUnlock()

	m2 := NewMenu()
	m2.Add("Header", nil).SetDisabled(true)
	m2.Add("Clickable", nil)
	m2.AddSeparator()
	m2.Add("Quit", nil)
	if err := tr.SetMenu(m2); err != nil {
		t.Fatal(err)
	}
	tr.mu.RLock()
	_, secondLayout, _ := svc.GetLayout(0, -1, nil)
	tr.mu.RUnlock()

	first, second := layoutIDs(t, firstLayout), layoutIDs(t, secondLayout)
	if len(first) != 4 {
		t.Fatalf("first layout has %d ids %v, want root plus 3 items", len(first), first)
	}
	if len(second) != 5 {
		t.Fatalf("second layout has %d ids %v, want root plus 4 items", len(second), second)
	}
	firstIDs := map[int32]bool{}
	for _, id := range first {
		firstIDs[id] = true
	}
	for _, id := range second {
		if id != 0 && firstIDs[id] {
			t.Fatalf("id %d reused across SetMenu; first=%v second=%v", id, first, second)
		}
	}
}

func TestItemPropertiesAreExplicit(t *testing.T) {
	tr := testTray(t)
	svc := &dbusMenuService{tray: tr}

	normal := svc.itemProperties(menuItemSnapshot{itemType: MenuItemNormal, label: "row"})
	if v, ok := normal["enabled"].Value().(bool); !ok || !v {
		t.Fatalf("normal item lacks explicit enabled=true: %v", normal)
	}
	if v, ok := normal["type"].Value().(string); !ok || v != "standard" {
		t.Fatalf("normal item lacks explicit type=standard: %v", normal)
	}
	sep := svc.itemProperties(menuItemSnapshot{itemType: MenuItemSeparator})
	if v, ok := sep["type"].Value().(string); !ok || v != "separator" {
		t.Fatalf("separator lacks type=separator: %v", sep)
	}
	if _, ok := sep["enabled"]; ok {
		t.Fatal("separator should not carry enabled")
	}
	disabled := svc.itemProperties(menuItemSnapshot{itemType: MenuItemNormal, label: "row", disabled: true})
	if v, ok := disabled["enabled"].Value().(bool); !ok || v {
		t.Fatalf("disabled item should export enabled=false: %v", disabled)
	}
}

func TestEventDispatchUsesAssignedIDs(t *testing.T) {
	tr := testTray(t)
	svc := &dbusMenuService{tray: tr}

	clicked := false
	sub := NewMenu()
	sub.Add("Child", func() { clicked = true })
	menu := NewMenu()
	menu.AddSubmenu("Parent", sub)
	if err := tr.SetMenu(menu); err != nil {
		t.Fatal(err)
	}

	tr.mu.RLock()
	parentID := int32(-1)
	for id, item := range tr.menuItems {
		if item.Submenu != nil {
			parentID = id
		}
	}
	childID := svc.findChildStartID(parentID)
	tr.mu.RUnlock()
	if childID <= 0 {
		t.Fatalf("findChildStartID(%d)=%d, want assigned child id", parentID, childID)
	}
	if err := svc.Event(childID, "clicked", dbus.Variant{}, 0); err != nil {
		t.Fatal(err)
	}
	if !clicked {
		t.Fatal("dispatching the child id did not reach the submenu item")
	}
}
