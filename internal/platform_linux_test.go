//go:build linux

package internal

import (
	"testing"

	"github.com/godbus/dbus/v5"
)

// newTestLinuxTray builds a linuxTray with no D-Bus connection. SetMenu skips
// the LayoutUpdated emit when conn is nil, so the static menu bookkeeping can
// be exercised in isolation.
func newTestLinuxTray() *linuxTray {
	return &linuxTray{
		menuItems:  make(map[int32]*MenuItem),
		itemIDs:    make(map[uint32]int32),
		nextMenuID: 1,
	}
}

func TestLinuxMenuIDsStableAcrossRebuilds(t *testing.T) {
	tray := newTestLinuxTray()

	shared := NewMenu().Add("Shared", func() {})
	removed := NewMenu().Add("Removed", func() {})
	sub := NewMenu()
	subChild := sub.Add("Child", func() {})

	menuA := NewMenu()
	menuA.Items = append(menuA.Items, shared, removed)
	container := menuA.AddSubmenu("Sub", sub)

	if err := tray.SetMenu(menuA); err != nil {
		t.Fatalf("SetMenu A: %v", err)
	}

	sharedID := tray.itemIDs[shared.ID()]
	removedID := tray.itemIDs[removed.ID()]
	containerID := tray.itemIDs[container.ID()]
	childID := tray.itemIDs[subChild.ID()]
	if sharedID == 0 || removedID == 0 || containerID == 0 || childID == 0 {
		t.Fatalf("menu IDs not assigned: shared=%d removed=%d container=%d child=%d",
			sharedID, removedID, containerID, childID)
	}

	// Rebuild reusing the same MenuItem objects, dropping one and adding one.
	fresh := NewMenu().Add("Fresh", func() {})
	menuB := NewMenu()
	menuB.Items = append(menuB.Items, shared, container, fresh)

	if err := tray.SetMenu(menuB); err != nil {
		t.Fatalf("SetMenu B: %v", err)
	}

	if got := tray.itemIDs[shared.ID()]; got != sharedID {
		t.Errorf("shared item ID changed across rebuild: %d -> %d", sharedID, got)
	}
	if got := tray.itemIDs[container.ID()]; got != containerID {
		t.Errorf("submenu container ID changed across rebuild: %d -> %d", containerID, got)
	}
	if got := tray.itemIDs[subChild.ID()]; got != childID {
		t.Errorf("submenu child ID changed across rebuild: %d -> %d", childID, got)
	}
	if _, ok := tray.menuItems[removedID]; ok {
		t.Error("removed item is still dispatchable after rebuild")
	}

	// Core invariant: every current ID maps back to the item carrying it, so a
	// stale ID can never activate a different item.
	for id, item := range tray.menuItems {
		if tray.itemIDs[item.ID()] != id {
			t.Errorf("menuItems[%d] does not map back to item %d (aliasing)", id, item.ID())
		}
	}
}

// TestLinuxMenuEventStaleIDDoesNotAlias covers the reported bug shape: a
// rebuild replaces an item at the same position. Positional IDs would let the
// stale ID fire the replacement's callback; stable per-item IDs must drop it.
func TestLinuxMenuEventStaleIDDoesNotAlias(t *testing.T) {
	tray := newTestLinuxTray()
	svc := &dbusMenuService{tray: tray}

	var fired []string
	alpha := NewMenu().Add("Alpha", func() { fired = append(fired, "alpha") })
	beta := NewMenu().Add("Beta", func() { fired = append(fired, "beta") })
	gamma := NewMenu().Add("Gamma", func() { fired = append(fired, "gamma") })
	delta := NewMenu().Add("Delta", func() { fired = append(fired, "delta") })

	menuA := NewMenu()
	menuA.Items = append(menuA.Items, alpha, beta, gamma)
	if err := tray.SetMenu(menuA); err != nil {
		t.Fatalf("SetMenu A: %v", err)
	}
	staleID := tray.itemIDs[gamma.ID()]

	// "Gamma" is replaced by "Delta" in the same slot.
	menuB := NewMenu()
	menuB.Items = append(menuB.Items, alpha, beta, delta)
	if err := tray.SetMenu(menuB); err != nil {
		t.Fatalf("SetMenu B: %v", err)
	}

	if deltaID := tray.itemIDs[delta.ID()]; deltaID == staleID {
		t.Fatalf("new item reused the removed item's ID %d", staleID)
	}

	// A click resolved against the pre-rebuild menu must not fire Delta.
	if err := svc.Event(staleID, "clicked", dbus.Variant{}, 0); err != nil {
		t.Fatalf("Event(stale): %v", err)
	}
	if len(fired) != 0 {
		t.Fatalf("stale ID %d dispatched %v; want no dispatch", staleID, fired)
	}

	// A current ID still dispatches normally.
	if err := svc.Event(tray.itemIDs[alpha.ID()], "clicked", dbus.Variant{}, 0); err != nil {
		t.Fatalf("Event(current): %v", err)
	}
	if len(fired) != 1 || fired[0] != "alpha" {
		t.Fatalf("current ID dispatched %v, want [alpha]", fired)
	}
}
