//go:build windows

package internal

import (
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

// procGetMenuItemInfoW is used only by tests to verify that SetMenuItemInfoW
// updates landed on the native Win32 menu.
var procGetMenuItemInfoW = windows.NewLazySystemDLL("user32.dll").NewProc("GetMenuItemInfoW")

// getMenuItemInfo reads back a native menu item's string and state via
// GetMenuItemInfoW, mirroring the menuItemInfoW layout used by SetMenuItemInfoW.
func getMenuItemInfo(t *testing.T, hmenu uintptr, uItem uintptr, byPosition bool) (string, uint32) {
	t.Helper()

	buf := make([]uint16, 256)
	mii := menuItemInfoW{
		cbSize:     uint32(unsafe.Sizeof(menuItemInfoW{})),
		fMask:      miimString | miimState,
		dwTypeData: uintptr(unsafe.Pointer(&buf[0])),
		cch:        uint32(len(buf)),
	}

	fByPosition := uintptr(0)
	if byPosition {
		fByPosition = 1
	}
	ret, _, _ := procGetMenuItemInfoW.Call(hmenu, uItem, fByPosition, uintptr(unsafe.Pointer(&mii)))
	if ret == 0 {
		t.Fatalf("GetMenuItemInfoW failed for hmenu=%#x uItem=%#x byPosition=%v", hmenu, uItem, byPosition)
	}

	return windows.UTF16ToString(buf), mii.fState
}

// newTestWin32Tray builds a real Win32 tray without creating the message-only
// window: buildHMENU/populateMenu/UpdateItem do not require an HWND.
func newTestWin32Tray() *win32Tray {
	return NewPlatformTray(nil).(*win32Tray)
}

func TestShouldStopWin32MessageLoop(t *testing.T) {
	tests := []struct {
		name                string
		remainingTrays      int
		wantMessageLoopStop bool
	}{
		{name: "last tray removed", remainingTrays: 0, wantMessageLoopStop: true},
		{name: "one tray remains", remainingTrays: 1, wantMessageLoopStop: false},
		{name: "multiple trays remain", remainingTrays: 2, wantMessageLoopStop: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldStopWin32MessageLoop(test.remainingTrays); got != test.wantMessageLoopStop {
				t.Errorf("shouldStopWin32MessageLoop(%d) = %v, want %v", test.remainingTrays, got, test.wantMessageLoopStop)
			}
		})
	}
}

func TestUpdateItem_SubmenuContainer_Label(t *testing.T) {
	tray := newTestWin32Tray()

	sub := NewMenu()
	sub.Add("SubItem", nil)

	root := NewMenu()
	root.Add("Top", nil)
	root.AddSeparator()
	container := root.AddSubmenu("Options", sub)
	root.Add("Bottom", nil)

	if err := tray.SetMenu(root); err != nil {
		t.Fatalf("SetMenu: %v", err)
	}
	defer procDestroyMenu.Call(tray.hmenu)

	// Native layout: Top(0), separator(1), Options(2), Bottom(3).
	pos, ok := tray.menuState.pos[container.ID()]
	if !ok {
		t.Fatal("submenu container not registered in pos map")
	}
	if pos != 2 {
		t.Errorf("container position = %d, want 2", pos)
	}

	container.Label = "Settings"
	if err := tray.UpdateItem(container); err != nil {
		t.Fatalf("UpdateItem: %v", err)
	}

	got, _ := getMenuItemInfo(t, tray.hmenu, uintptr(pos), true)
	if got != "Settings" {
		t.Errorf("container label = %q, want %q", got, "Settings")
	}
}

func TestUpdateItem_SubmenuContainer_Disabled(t *testing.T) {
	tray := newTestWin32Tray()

	sub := NewMenu()
	sub.Add("SubItem", nil)

	root := NewMenu()
	container := root.AddSubmenu("Options", sub)

	if err := tray.SetMenu(root); err != nil {
		t.Fatalf("SetMenu: %v", err)
	}
	defer procDestroyMenu.Call(tray.hmenu)

	pos, ok := tray.menuState.pos[container.ID()]
	if !ok {
		t.Fatal("submenu container not registered in pos map")
	}

	container.Disabled = true
	if err := tray.UpdateItem(container); err != nil {
		t.Fatalf("UpdateItem: %v", err)
	}
	_, state := getMenuItemInfo(t, tray.hmenu, uintptr(pos), true)
	if state&mfsDisabled == 0 {
		t.Errorf("container state = %#x, want MFS_DISABLED set", state)
	}

	container.Disabled = false
	if err := tray.UpdateItem(container); err != nil {
		t.Fatalf("UpdateItem: %v", err)
	}
	_, state = getMenuItemInfo(t, tray.hmenu, uintptr(pos), true)
	if state&mfsDisabled != 0 {
		t.Errorf("container state = %#x, want MFS_DISABLED clear", state)
	}
}

func TestUpdateItem_SubmenuContainer_Nested(t *testing.T) {
	tray := newTestWin32Tray()

	level3 := NewMenu()
	level3.Add("Deep", nil)

	level2 := NewMenu()
	container2 := level2.AddSubmenu("Level2", level3)

	level1 := NewMenu()
	container1 := level1.AddSubmenu("Level1", level2)

	root := NewMenu()
	root.AddSubmenu("RootSub", level1)

	if err := tray.SetMenu(root); err != nil {
		t.Fatalf("SetMenu: %v", err)
	}
	defer procDestroyMenu.Call(tray.hmenu)

	// container1 lives inside the root submenu's HMENU at position 0.
	pos1, ok := tray.menuState.pos[container1.ID()]
	if !ok {
		t.Fatal("nested container Level1 not registered in pos map")
	}
	if pos1 != 0 {
		t.Errorf("Level1 position = %d, want 0", pos1)
	}
	hmenu1, ok := tray.menuState.owners[container1.ID()]
	if !ok {
		t.Fatal("nested container Level1 not registered in owners map")
	}
	if hmenu1 == tray.hmenu {
		t.Error("Level1 owning HMENU should be the root submenu, not the root menu")
	}

	container1.Label = "Renamed"
	if err := tray.UpdateItem(container1); err != nil {
		t.Fatalf("UpdateItem Level1: %v", err)
	}
	got, _ := getMenuItemInfo(t, hmenu1, uintptr(pos1), true)
	if got != "Renamed" {
		t.Errorf("Level1 label = %q, want %q", got, "Renamed")
	}

	// container2 lives inside level2's HMENU at position 0.
	pos2, ok := tray.menuState.pos[container2.ID()]
	if !ok {
		t.Fatal("nested container Level2 not registered in pos map")
	}
	if pos2 != 0 {
		t.Errorf("Level2 position = %d, want 0", pos2)
	}
	hmenu2, ok := tray.menuState.owners[container2.ID()]
	if !ok {
		t.Fatal("nested container Level2 not registered in owners map")
	}

	container2.Label = "Renamed2"
	if err := tray.UpdateItem(container2); err != nil {
		t.Fatalf("UpdateItem Level2: %v", err)
	}
	got, _ = getMenuItemInfo(t, hmenu2, uintptr(pos2), true)
	if got != "Renamed2" {
		t.Errorf("Level2 label = %q, want %q", got, "Renamed2")
	}
}

func TestUpdateItem_NormalItem_Regression(t *testing.T) {
	tray := newTestWin32Tray()

	sub := NewMenu()
	subItem := sub.Add("SubItem", nil)

	root := NewMenu()
	top := root.Add("Top", nil)
	root.AddSubmenu("Options", sub)

	if err := tray.SetMenu(root); err != nil {
		t.Fatalf("SetMenu: %v", err)
	}
	defer procDestroyMenu.Call(tray.hmenu)

	// Root-level item resolves by command ID.
	top.Label = "Tops"
	if err := tray.UpdateItem(top); err != nil {
		t.Fatalf("UpdateItem top: %v", err)
	}
	got, _ := getMenuItemInfo(t, tray.hmenu, uintptr(top.ID()), false)
	if got != "Tops" {
		t.Errorf("top label = %q, want %q", got, "Tops")
	}

	// Item inside a submenu resolves by command ID on the submenu's HMENU.
	subItem.Label = "Subbed"
	if err := tray.UpdateItem(subItem); err != nil {
		t.Fatalf("UpdateItem subItem: %v", err)
	}
	got, _ = getMenuItemInfo(t, tray.menuState.owners[subItem.ID()], uintptr(subItem.ID()), false)
	if got != "Subbed" {
		t.Errorf("sub item label = %q, want %q", got, "Subbed")
	}
}

func TestSetMenu_CommandIDsAreStablePerItem(t *testing.T) {
	tray := newTestWin32Tray()

	shared := NewMenu().Add("Shared", nil)
	removed := NewMenu().Add("Removed", nil)

	menuA := NewMenu()
	menuA.Items = append(menuA.Items, shared, removed)

	if err := tray.SetMenu(menuA); err != nil {
		t.Fatalf("SetMenu A: %v", err)
	}
	if _, ok := tray.menuState.cmd[shared.ID()]; !ok {
		t.Fatal("shared item missing from cmd map after first SetMenu")
	}

	// Rebuild with the same *MenuItem (shared) plus a brand new one; the
	// removed item is gone from the new menu.
	fresh := NewMenu().Add("Fresh", nil)
	menuB := NewMenu()
	menuB.Items = append(menuB.Items, shared, fresh)

	if err := tray.SetMenu(menuB); err != nil {
		t.Fatalf("SetMenu B: %v", err)
	}
	defer func() { destroyMenu(tray.hmenu) }()

	// Native menu entries are addressed by the stable item ID.
	if got, _ := getMenuItemInfo(t, tray.hmenu, uintptr(shared.ID()), false); got != "Shared" {
		t.Errorf("shared item native label = %q, want %q", got, "Shared")
	}
	if got, _ := getMenuItemInfo(t, tray.hmenu, uintptr(fresh.ID()), false); got != "Fresh" {
		t.Errorf("fresh item native label = %q, want %q", got, "Fresh")
	}

	// A stale ID for an item that was removed must not dispatch after rebuild.
	if _, ok := tray.menuState.cmd[removed.ID()]; ok {
		t.Error("removed item's command ID still dispatches after rebuild")
	}

	// Core invariant: every command ID maps to the item carrying that same
	// stable ID, so a stale command ID can never fire a different item.
	for cmdID, item := range tray.menuState.cmd {
		if item.ID() != cmdID {
			t.Errorf("cmd[%d] maps to item %d (aliasing)", cmdID, item.ID())
		}
	}
}

// openPopupState mimics what showContextMenu does before it calls
// TrackPopupMenu: it snapshots the menu that is about to be displayed.
func openPopupState(tray *win32Tray) *menuItemMap {
	tray.menuMu.Lock()
	defer tray.menuMu.Unlock()
	state := tray.menuState
	tray.popupOpen = true
	tray.popupHMenu = tray.hmenu
	return state
}

// TestPopup_ClickUsesDisplayedMenu_AfterRebuild is the regression for the
// reported bug: a background SetMenu rebuilds the menu while the popup is open.
// Clicking "View Logs" on the displayed menu must still dispatch that item and
// never a different one.
func TestPopup_ClickUsesDisplayedMenu_AfterRebuild(t *testing.T) {
	tray := newTestWin32Tray()

	var fired []string
	build := func(big bool) *Menu {
		sub := NewMenu()
		sub.Add("Sub item 1", func() { fired = append(fired, "Sub item 1") })
		sub.Add("Sub item 2", func() { fired = append(fired, "Sub item 2") })
		if big {
			sub.Add("Sub item 3", func() { fired = append(fired, "Sub item 3") })
		}
		root := NewMenu()
		root.AddSubmenu("Submenu", sub)
		root.Add("View Logs", func() { fired = append(fired, "View Logs") })
		if !big {
			root.Add("Noop", func() { fired = append(fired, "Noop") })
		}
		root.Add("Quit", func() { fired = append(fired, "Quit") })
		return root
	}

	small := build(false)
	if err := tray.SetMenu(small); err != nil {
		t.Fatalf("SetMenu: %v", err)
	}
	viewLogs := small.Items[1]
	displayed := openPopupState(tray)
	displayedHMenu := displayed.hmenu

	// A fresh tree of the opposite shape is published while the popup is open.
	if err := tray.SetMenu(build(true)); err != nil {
		t.Fatalf("background SetMenu: %v", err)
	}

	// The menu on screen must not have been renumbered out from under the user.
	tray.menuMu.Lock()
	if tray.menuState != displayed {
		tray.menuMu.Unlock()
		t.Fatal("menu was republished while a popup was open")
	}
	if tray.hmenu != displayedHMenu {
		tray.menuMu.Unlock()
		t.Fatalf("displayed HMENU changed: %#x -> %#x", displayedHMenu, tray.hmenu)
	}
	tray.menuMu.Unlock()

	// "View Logs" is clicked on the displayed menu.
	got := tray.endPopup(viewLogs.ID(), displayed)
	if got != viewLogs {
		t.Fatalf("click resolved to %v, want %v", got, viewLogs)
	}
	if got != nil && got.OnClick != nil {
		got.OnClick()
	}

	if len(fired) != 1 || fired[0] != "View Logs" {
		t.Fatalf("fired %v, want [View Logs]", fired)
	}

	// The queued menu became current, and the menu the popup displayed is gone.
	tray.menuMu.Lock()
	nowHMenu, nowState := tray.hmenu, tray.menuState
	popupHMenu, pending := tray.popupHMenu, tray.pendingHMenu
	tray.menuMu.Unlock()

	if popupHMenu != 0 {
		t.Errorf("popup handle not cleared: %#x", popupHMenu)
	}
	if pending != 0 {
		t.Errorf("pending handle not cleared: %#x", pending)
	}
	if nowHMenu == displayedHMenu || nowHMenu == 0 {
		t.Errorf("queued menu not published: hmenu=%#x displayed=%#x", nowHMenu, displayedHMenu)
	}
	if nowState == displayed {
		t.Error("dispatch map still points at the menu the popup displayed")
	}
	// The rebuilt tree reuses no item IDs from the old one, so its own IDs must
	// resolve to its own items only.
	for cmdID, item := range nowState.cmd {
		if item.ID() != cmdID {
			t.Errorf("cmd[%d] maps to item %d (aliasing)", cmdID, item.ID())
		}
	}
	destroyMenu(nowHMenu)
}

// TestPopup_StaleIDFromDisplayedMenuNeverAliases covers the reverse direction:
// the ID the popup reports is looked up only in the menu that was displayed.
// A rebuild that replaced an item at the same position must not receive the
// click.
func TestPopup_StaleIDFromDisplayedMenuNeverAliases(t *testing.T) {
	tray := newTestWin32Tray()

	var fired []string
	alpha := NewMenu().Add("Alpha", func() { fired = append(fired, "alpha") })
	beta := NewMenu().Add("Beta", func() { fired = append(fired, "beta") })
	gamma := NewMenu().Add("Gamma", func() { fired = append(fired, "gamma") })
	menuA := NewMenu()
	menuA.Items = append(menuA.Items, alpha, beta, gamma)
	if err := tray.SetMenu(menuA); err != nil {
		t.Fatalf("SetMenu A: %v", err)
	}

	displayed := openPopupState(tray)

	// Gamma is replaced by a brand new Delta item.
	delta := NewMenu().Add("Delta", func() { fired = append(fired, "delta") })
	menuB := NewMenu()
	menuB.Items = append(menuB.Items, alpha, beta, delta)
	if err := tray.SetMenu(menuB); err != nil {
		t.Fatalf("SetMenu B: %v", err)
	}

	// The click is still resolved against the displayed menu, so it reaches the
	// item the user actually saw.
	got := tray.endPopup(gamma.ID(), displayed)
	if got != gamma {
		t.Fatalf("click resolved to %v, want Gamma", got)
	}
	got.OnClick()
	if len(fired) != 1 || fired[0] != "gamma" {
		t.Fatalf("fired %v, want [gamma]", fired)
	}

	// Delta's own command ID never resolves through the old menu.
	if got := tray.menuState.cmd[delta.ID()]; got != delta {
		t.Fatalf("delta not current after popup closed: %v", got)
	}
	destroyMenu(tray.hmenu)
}

// TestSetMenu_QueuedWhilePopupOpen_AppliedOnNextOpen covers repeated rebuilds:
// only the newest queued menu survives, and it is published when the next popup
// opens rather than being lost.
func TestSetMenu_QueuedWhilePopupOpen_AppliedOnNextOpen(t *testing.T) {
	tray := newTestWin32Tray()

	first := NewMenu().Add("First", nil)
	menuA := NewMenu()
	menuA.Items = append(menuA.Items, first)
	if err := tray.SetMenu(menuA); err != nil {
		t.Fatalf("SetMenu A: %v", err)
	}
	displayedHMenu := tray.hmenu

	openPopupState(tray)

	// Two rebuilds arrive while the popup is open; the newer one wins.
	second := NewMenu().Add("Second", nil)
	menuB := NewMenu()
	menuB.Items = append(menuB.Items, second)
	if err := tray.SetMenu(menuB); err != nil {
		t.Fatalf("SetMenu B: %v", err)
	}
	third := NewMenu().Add("Third", nil)
	menuC := NewMenu()
	menuC.Items = append(menuC.Items, third)
	if err := tray.SetMenu(menuC); err != nil {
		t.Fatalf("SetMenu C: %v", err)
	}

	// The displayed menu keeps dispatching its own item while the rebuilds are
	// queued behind it.
	tray.menuMu.Lock()
	if tray.hmenu != displayedHMenu {
		tray.menuMu.Unlock()
		t.Fatal("displayed HMENU changed while queuing rebuilds")
	}
	if _, ok := tray.menuState.cmd[first.ID()]; !ok {
		tray.menuMu.Unlock()
		t.Fatal("displayed menu no longer dispatches its own item")
	}
	tray.menuMu.Unlock()

	// Publishing the queued menu happens at the start of the next popup.
	tray.menuMu.Lock()
	tray.applyPendingMenuLocked()
	state, hmenu := tray.menuState, tray.hmenu
	tray.menuMu.Unlock()
	defer destroyMenu(hmenu)

	if _, ok := state.cmd[third.ID()]; !ok {
		t.Fatal("newest queued menu was not applied")
	}
	if _, ok := state.cmd[second.ID()]; ok {
		t.Error("superseded queued menu was applied")
	}
	if _, ok := state.cmd[first.ID()]; ok {
		t.Error("stale item still dispatches after the queued menu was applied")
	}
}

func TestUpdateItem_IgnoresItemsOutsideCurrentMenu(t *testing.T) {
	tray := newTestWin32Tray()

	shown := NewMenu().Add("Shown", nil)
	shownMenu := NewMenu()
	shownMenu.Items = append(shownMenu.Items, shown)
	if err := tray.SetMenu(shownMenu); err != nil {
		t.Fatalf("SetMenu: %v", err)
	}

	other := NewMenu().Add("Other", nil)
	otherMenu := NewMenu()
	otherMenu.Items = append(otherMenu.Items, other)
	if err := tray.SetMenu(otherMenu); err != nil {
		t.Fatalf("SetMenu 2: %v", err)
	}
	hmenu := tray.hmenu
	defer destroyMenu(hmenu)

	// An item that is not part of the current menu must not be updated on it.
	other.Label = "Renamed"
	if err := tray.UpdateItem(other); err != nil {
		t.Fatalf("UpdateItem: %v", err)
	}
	if got, _ := getMenuItemInfo(t, hmenu, uintptr(other.ID()), false); got != "Renamed" {
		t.Errorf("current menu label = %q, want %q", got, "Renamed")
	}
	// The item dropped from the menu has no native entry in the current menu.
	if _, ok := tray.menuState.owners[shown.ID()]; ok {
		t.Error("dropped item is still registered against the current menu")
	}
}

func TestUpdateItem_NoMenuRegistered_IsNoop(t *testing.T) {
	tray := newTestWin32Tray()

	if err := tray.SetMenu(nil); err != nil {
		t.Fatalf("SetMenu(nil): %v", err)
	}

	item := NewMenu().Add("Orphan", nil)
	if err := tray.UpdateItem(item); err != nil {
		t.Fatalf("UpdateItem without a menu: %v", err)
	}
}

func TestPopulateMenu_NilSubmenu_DoesNotShiftPositions(t *testing.T) {
	tray := newTestWin32Tray()

	root := NewMenu()
	root.Add("First", nil)
	broken := &MenuItem{id: newMenuItemID(), Label: "Broken", Type: MenuItemSubmenu} // Submenu == nil
	root.Items = append(root.Items, broken)
	sub := NewMenu()
	sub.Add("SubItem", nil)
	container := root.AddSubmenu("Options", sub)

	if err := tray.SetMenu(root); err != nil {
		t.Fatalf("SetMenu: %v", err)
	}
	defer procDestroyMenu.Call(tray.hmenu)

	if _, ok := tray.menuState.pos[broken.ID()]; ok {
		t.Error("nil-submenu item should not be registered in pos map")
	}
	if _, ok := tray.menuState.owners[broken.ID()]; ok {
		t.Error("nil-submenu item should not be registered in owners map")
	}

	// Native layout: First(0), Options(1). The nil submenu appended nothing,
	// so later items must not have shifted positions.
	pos, ok := tray.menuState.pos[container.ID()]
	if !ok {
		t.Fatal("submenu container not registered in pos map")
	}
	if pos != 1 {
		t.Errorf("container position = %d, want 1", pos)
	}
}
