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
	pos, ok := tray.itemPos[container.ID()]
	if !ok {
		t.Fatal("submenu container not registered in itemPos")
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

	pos, ok := tray.itemPos[container.ID()]
	if !ok {
		t.Fatal("submenu container not registered in itemPos")
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
	pos1, ok := tray.itemPos[container1.ID()]
	if !ok {
		t.Fatal("nested container Level1 not registered in itemPos")
	}
	if pos1 != 0 {
		t.Errorf("Level1 position = %d, want 0", pos1)
	}
	hmenu1, ok := tray.itemHMenus[container1.ID()]
	if !ok {
		t.Fatal("nested container Level1 not registered in itemHMenus")
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
	pos2, ok := tray.itemPos[container2.ID()]
	if !ok {
		t.Fatal("nested container Level2 not registered in itemPos")
	}
	if pos2 != 0 {
		t.Errorf("Level2 position = %d, want 0", pos2)
	}
	hmenu2, ok := tray.itemHMenus[container2.ID()]
	if !ok {
		t.Fatal("nested container Level2 not registered in itemHMenus")
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
	got, _ := getMenuItemInfo(t, tray.hmenu, uintptr(tray.itemIDs[top.ID()]), false)
	if got != "Tops" {
		t.Errorf("top label = %q, want %q", got, "Tops")
	}

	// Item inside a submenu resolves by command ID on the submenu's HMENU.
	subItem.Label = "Subbed"
	if err := tray.UpdateItem(subItem); err != nil {
		t.Fatalf("UpdateItem subItem: %v", err)
	}
	got, _ = getMenuItemInfo(t, tray.itemHMenus[subItem.ID()], uintptr(tray.itemIDs[subItem.ID()]), false)
	if got != "Subbed" {
		t.Errorf("sub item label = %q, want %q", got, "Subbed")
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

	if _, ok := tray.itemPos[broken.ID()]; ok {
		t.Error("nil-submenu item should not be registered in itemPos")
	}
	if _, ok := tray.itemHMenus[broken.ID()]; ok {
		t.Error("nil-submenu item should not be registered in itemHMenus")
	}

	// Native layout: First(0), Options(1). The nil submenu appended nothing,
	// so later items must not have shifted positions.
	pos, ok := tray.itemPos[container.ID()]
	if !ok {
		t.Fatal("submenu container not registered in itemPos")
	}
	if pos != 1 {
		t.Errorf("container position = %d, want 1", pos)
	}
}
