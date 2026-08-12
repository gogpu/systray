package internal

import (
	"bytes"
	"sync/atomic"
	"testing"
)

func TestNewMenu_CreatesEmpty(t *testing.T) {
	t.Parallel()

	m := NewMenu()
	if m == nil {
		t.Fatal("NewMenu returned nil")
	}
	if len(m.Items) != 0 {
		t.Errorf("expected empty menu, got %d items", len(m.Items))
	}
}

func TestMenu_Add(t *testing.T) {
	t.Parallel()

	called := false
	m := NewMenu()
	item := m.Add("Open", func() { called = true })

	if len(m.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(m.Items))
	}
	if item != m.Items[0] {
		t.Error("returned item should be the same as the one appended to menu")
	}
	if item.Label != "Open" {
		t.Errorf("label = %q, want %q", item.Label, "Open")
	}
	if item.Type != MenuItemNormal {
		t.Errorf("type = %d, want MenuItemNormal (%d)", item.Type, MenuItemNormal)
	}
	if item.OnClick == nil {
		t.Fatal("OnClick is nil")
	}

	item.OnClick()
	if !called {
		t.Error("OnClick callback was not invoked")
	}
}

func TestMenu_Add_NilCallback(t *testing.T) {
	t.Parallel()

	m := NewMenu()
	item := m.Add("NoOp", nil)

	if len(m.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(m.Items))
	}
	if item.OnClick != nil {
		t.Error("expected nil OnClick for nil callback")
	}
}

func TestMenu_AddCheckbox(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		label   string
		checked bool
	}{
		{"checked", "Enable Feature", true},
		{"unchecked", "Disable Feature", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := NewMenu()
			item := m.AddCheckbox(tt.label, tt.checked, nil)
			if len(m.Items) != 1 {
				t.Fatalf("expected 1 item, got %d", len(m.Items))
			}
			if item.Label != tt.label {
				t.Errorf("label = %q, want %q", item.Label, tt.label)
			}
			if item.Type != MenuItemCheckbox {
				t.Errorf("type = %d, want MenuItemCheckbox (%d)", item.Type, MenuItemCheckbox)
			}
			if item.Checked != tt.checked {
				t.Errorf("checked = %v, want %v", item.Checked, tt.checked)
			}
		})
	}
}

func TestMenu_AddSeparator(t *testing.T) {
	t.Parallel()

	m := NewMenu()
	item := m.AddSeparator()

	if len(m.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(m.Items))
	}
	if item != m.Items[0] {
		t.Error("returned item should be the same as the one appended to menu")
	}
	if item.Type != MenuItemSeparator {
		t.Errorf("type = %d, want MenuItemSeparator (%d)", item.Type, MenuItemSeparator)
	}
	if item.Label != "" {
		t.Errorf("separator should have empty label, got %q", item.Label)
	}
	if item.OnClick != nil {
		t.Error("separator should have nil OnClick")
	}
}

func TestMenu_AddSubmenu(t *testing.T) {
	t.Parallel()

	sub := NewMenu()
	sub.Add("SubItem", nil)

	m := NewMenu()
	item := m.AddSubmenu("More", sub)

	if len(m.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(m.Items))
	}
	if item.Label != "More" {
		t.Errorf("label = %q, want %q", item.Label, "More")
	}
	if item.Type != MenuItemSubmenu {
		t.Errorf("type = %d, want MenuItemSubmenu (%d)", item.Type, MenuItemSubmenu)
	}
	if item.Submenu == nil {
		t.Fatal("submenu is nil")
	}
	if len(item.Submenu.Items) != 1 {
		t.Errorf("submenu has %d items, want 1", len(item.Submenu.Items))
	}
	if item.Submenu.Items[0].Label != "SubItem" {
		t.Errorf("submenu item label = %q, want %q", item.Submenu.Items[0].Label, "SubItem")
	}
}

func TestMenu_AddSubmenu_DeepNesting(t *testing.T) {
	t.Parallel()

	const depth = 5

	// Build nested menu 5 levels deep.
	innermost := NewMenu()
	innermost.Add("Leaf", nil)
	current := innermost
	for i := depth - 1; i > 0; i-- {
		parent := NewMenu()
		parent.AddSubmenu("Level", current)
		current = parent
	}

	// Traverse and verify depth.
	menu := current
	actualDepth := 0
	for menu != nil {
		actualDepth++
		if len(menu.Items) == 0 {
			break
		}
		item := menu.Items[0]
		if item.Type == MenuItemSubmenu {
			menu = item.Submenu
		} else {
			// Reached the leaf item.
			break
		}
	}

	if actualDepth != depth {
		t.Errorf("traversal depth = %d, want %d", actualDepth, depth)
	}
}

func TestMenu_AddWithIcon(t *testing.T) {
	t.Parallel()

	icon := []byte{0x89, 0x50, 0x4E, 0x47} // PNG magic bytes
	m := NewMenu()
	item := m.AddWithIcon("Copy", icon, nil)

	if len(m.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(m.Items))
	}
	if item.Label != "Copy" {
		t.Errorf("label = %q, want %q", item.Label, "Copy")
	}
	if item.Type != MenuItemNormal {
		t.Errorf("type = %d, want MenuItemNormal (%d)", item.Type, MenuItemNormal)
	}
	if len(item.Icon) != 4 {
		t.Errorf("icon length = %d, want 4", len(item.Icon))
	}
}

func TestMenu_MultipleItems(t *testing.T) {
	t.Parallel()

	m := NewMenu()
	m.Add("File", nil)
	m.AddSeparator()
	m.AddCheckbox("Auto-save", true, nil)
	m.AddSeparator()

	recentMenu := NewMenu()
	recentMenu.Add("doc1.txt", nil)
	recentMenu.Add("doc2.txt", nil)
	m.AddSubmenu("Recent", recentMenu)

	m.AddWithIcon("Paste", []byte{0xFF}, nil)
	m.Add("Exit", nil)

	if len(m.Items) != 7 {
		t.Fatalf("expected 7 items, got %d", len(m.Items))
	}

	expectedTypes := []MenuItemType{
		MenuItemNormal,    // File
		MenuItemSeparator, // ---
		MenuItemCheckbox,  // Auto-save
		MenuItemSeparator, // ---
		MenuItemSubmenu,   // Recent
		MenuItemNormal,    // Paste (with icon)
		MenuItemNormal,    // Exit
	}

	for i, want := range expectedTypes {
		if m.Items[i].Type != want {
			t.Errorf("item[%d].Type = %d, want %d", i, m.Items[i].Type, want)
		}
	}

	// Verify submenu at index 4 has 2 items.
	sub := m.Items[4].Submenu
	if sub == nil {
		t.Fatal("submenu at index 4 is nil")
	}
	if len(sub.Items) != 2 {
		t.Errorf("submenu has %d items, want 2", len(sub.Items))
	}
}

func TestMenuItemType_Values(t *testing.T) {
	t.Parallel()

	// Verify the iota-based constants have expected values.
	if MenuItemNormal != 0 {
		t.Errorf("MenuItemNormal = %d, want 0", MenuItemNormal)
	}
	if MenuItemCheckbox != 1 {
		t.Errorf("MenuItemCheckbox = %d, want 1", MenuItemCheckbox)
	}
	if MenuItemSeparator != 2 {
		t.Errorf("MenuItemSeparator = %d, want 2", MenuItemSeparator)
	}
	if MenuItemSubmenu != 3 {
		t.Errorf("MenuItemSubmenu = %d, want 3", MenuItemSubmenu)
	}
}

func TestMenuItem_DefaultProperties(t *testing.T) {
	t.Parallel()

	m := NewMenu()
	item := m.Add("Test", nil)

	if item.Checked {
		t.Error("default item should not be checked")
	}
	if item.Disabled {
		t.Error("default item should not be disabled")
	}
	if item.Submenu != nil {
		t.Error("default item should have nil submenu")
	}
	if item.Icon != nil {
		t.Error("default item should have nil icon")
	}
}

func TestMenu_EmptyMenu(t *testing.T) {
	t.Parallel()

	m := NewMenu()

	// AddSeparator on empty menu should work.
	item := m.AddSeparator()
	if item == nil {
		t.Error("AddSeparator should return non-nil item")
	}
	if len(m.Items) != 1 {
		t.Errorf("expected 1 item after AddSeparator, got %d", len(m.Items))
	}
}

func TestMenuItem_ID_NonZero(t *testing.T) {
	t.Parallel()

	m := NewMenu()
	item := m.Add("Test", nil)

	if item.ID() == 0 {
		t.Error("menu item ID should be non-zero")
	}
}

func TestMenuItem_ID_Unique(t *testing.T) {
	t.Parallel()

	m := NewMenu()
	item1 := m.Add("One", nil)
	item2 := m.Add("Two", nil)
	item3 := m.AddCheckbox("Three", false, nil)

	ids := map[uint32]bool{item1.ID(): true, item2.ID(): true, item3.ID(): true}
	if len(ids) != 3 {
		t.Error("menu items should have unique IDs")
	}
}

func TestMenuItem_SetLabel(t *testing.T) {
	t.Parallel()

	m := NewMenu()
	item := m.Add("Original", nil)

	item.SetLabel("Updated")

	if item.Label != "Updated" {
		t.Errorf("label = %q, want %q", item.Label, "Updated")
	}
}

func TestMenuItem_SetChecked(t *testing.T) {
	t.Parallel()

	m := NewMenu()
	item := m.AddCheckbox("Toggle", false, nil)

	item.SetChecked(true)

	if !item.Checked {
		t.Error("checked should be true")
	}
}

func TestMenuItem_SetDisabled(t *testing.T) {
	t.Parallel()

	m := NewMenu()
	item := m.Add("Action", nil)

	item.SetDisabled(true)

	if !item.Disabled {
		t.Error("disabled should be true")
	}
}

func TestMenuItem_SetIcon(t *testing.T) {
	t.Parallel()

	m := NewMenu()
	item := m.Add("Item", nil)

	icon := []byte{0x89, 0x50}
	item.SetIcon(icon)

	if len(item.Icon) != 2 {
		t.Errorf("icon length = %d, want 2", len(item.Icon))
	}
}

func TestSetMenuUpdater_Recursive(t *testing.T) {
	t.Parallel()

	calls := 0
	updater := &mockUpdater{onUpdate: func(_ *MenuItem) { calls++ }}

	sub := NewMenu()
	sub.Add("SubItem", nil)

	root := NewMenu()
	root.Add("Top", nil)
	root.AddSubmenu("More", sub)

	SetMenuUpdater(root, updater)

	// 3 items total: "Top", submenu item, "SubItem"
	// Trigger updates on all.
	root.Items[0].SetLabel("Changed1")
	root.Items[1].SetLabel("Changed2")
	sub.Items[0].SetLabel("Changed3")

	if calls != 3 {
		t.Errorf("updater called %d times, want 3", calls)
	}
}

func TestSetMenuUpdater_NilMenu(t *testing.T) {
	t.Parallel()

	// Should not panic.
	SetMenuUpdater(nil, &mockUpdater{})
}

func TestMenuItemSetUpdaterNilDetachesLegacyUpdater(t *testing.T) {
	t.Parallel()

	item := NewMenu().Add("item", nil)
	calls := 0
	item.SetUpdater(&mockUpdater{onUpdate: func(*MenuItem) { calls++ }})
	item.SetUpdater(nil)

	item.SetLabel("detached")

	if calls != 0 {
		t.Fatalf("detached updater calls = %d, want 0", calls)
	}
}

func TestMenuItemUpdateUsesMutationSnapshot(t *testing.T) {
	t.Parallel()

	item := NewMenu().Add("original", nil)
	updater := &blockingSnapshotUpdater{
		entered:  make(chan struct{}),
		release:  make(chan struct{}),
		observed: make(chan menuItemSnapshot, 2),
	}
	setMenuSnapshotUpdater(&Menu{Items: []*MenuItem{item}}, updater)

	done := make(chan struct{})
	go func() {
		item.SetLabel("queued")
		close(done)
	}()

	<-updater.entered
	item.SetDisabled(true)
	close(updater.release)
	<-done

	var labelUpdate, disabledUpdate menuItemSnapshot
	for range 2 {
		update := <-updater.observed
		if update.disabled {
			disabledUpdate = update
		} else {
			labelUpdate = update
		}
	}

	if labelUpdate.label != "queued" || labelUpdate.disabled {
		t.Fatalf("label update = {label:%q disabled:%v}, want mutation-time state", labelUpdate.label, labelUpdate.disabled)
	}
	if disabledUpdate.label != "queued" || !disabledUpdate.disabled {
		t.Fatalf("disabled update = {label:%q disabled:%v}, want coherent later state", disabledUpdate.label, disabledUpdate.disabled)
	}
}

func TestMenuItemDispatchAllowsReentrantSetter(t *testing.T) {
	t.Parallel()

	item := NewMenu().Add("item", nil)
	var updates []menuItemSnapshot
	updater := snapshotUpdaterFunc(func(update menuItemSnapshot) error {
		updates = append(updates, update)
		if len(updates) == 1 {
			item.SetDisabled(true)
		}
		return nil
	})
	setMenuSnapshotUpdater(&Menu{Items: []*MenuItem{item}}, updater)

	item.SetLabel("updated")

	if len(updates) != 2 {
		t.Fatalf("updates = %d, want initial and reentrant mutation", len(updates))
	}
	if updates[0].disabled || !updates[1].disabled {
		t.Fatalf("disabled states = [%v, %v], want [false, true]", updates[0].disabled, updates[1].disabled)
	}
}

func TestLegacyUpdaterPreservesIdentityAndReentrancy(t *testing.T) {
	t.Parallel()

	item := NewMenu().Add("item", nil)
	calls := 0
	item.SetUpdater(&mockUpdater{onUpdate: func(got *MenuItem) {
		calls++
		if got != item {
			t.Errorf("UpdateItem item = %p, want canonical %p", got, item)
		}
		if calls == 1 {
			got.SetDisabled(true)
		}
	}})

	item.SetLabel("updated")

	if calls != 2 {
		t.Fatalf("updater calls = %d, want initial and reentrant mutation", calls)
	}
	if !item.IsDisabled() {
		t.Fatal("reentrant mutation was not retained")
	}
}

func TestMenuItemIconSnapshotsOwnBytes(t *testing.T) {
	t.Parallel()

	initial := []byte{1, 2, 3}
	item := NewMenu().AddWithIcon("icon", initial, nil)
	initial[0] = 9
	if got := item.snapshot().icon; !bytes.Equal(got, []byte{1, 2, 3}) {
		t.Fatalf("initial icon = %v, want owned copy", got)
	}

	updater := &blockingSnapshotUpdater{
		entered:  make(chan struct{}),
		release:  make(chan struct{}),
		observed: make(chan menuItemSnapshot, 1),
	}
	setMenuSnapshotUpdater(&Menu{Items: []*MenuItem{item}}, updater)

	icon := []byte{4, 5, 6}
	done := make(chan struct{})
	go func() {
		item.SetIcon(icon)
		close(done)
	}()

	<-updater.entered
	icon[0] = 8
	close(updater.release)
	<-done

	update := <-updater.observed
	if !bytes.Equal(update.icon, []byte{4, 5, 6}) {
		t.Fatalf("queued icon = %v, want mutation-time owned copy", update.icon)
	}
	if got := item.snapshot().icon; !bytes.Equal(got, []byte{4, 5, 6}) {
		t.Fatalf("stored icon = %v, want owned copy", got)
	}
}

type blockingSnapshotUpdater struct {
	calls    atomic.Uint32
	entered  chan struct{}
	release  chan struct{}
	observed chan menuItemSnapshot
}

type snapshotUpdaterFunc func(menuItemSnapshot) error

func (f snapshotUpdaterFunc) updateItem(item menuItemSnapshot) error {
	return f(item)
}

func (u *blockingSnapshotUpdater) updateItem(item menuItemSnapshot) error {
	if u.calls.Add(1) == 1 {
		close(u.entered)
		<-u.release
	}
	u.observed <- item
	return nil
}

// mockUpdater implements MenuItemUpdater for testing.
type mockUpdater struct {
	onUpdate func(*MenuItem)
}

func (u *mockUpdater) UpdateItem(item *MenuItem) error {
	if u.onUpdate != nil {
		u.onUpdate(item)
	}
	return nil
}
