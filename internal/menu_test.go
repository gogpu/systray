package internal

import (
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
	m := NewMenu().Add("Open", func() { called = true })

	if len(m.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(m.Items))
	}

	item := m.Items[0]
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

	m := NewMenu().Add("NoOp", nil)

	if len(m.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(m.Items))
	}
	if m.Items[0].OnClick != nil {
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

			m := NewMenu().AddCheckbox(tt.label, tt.checked, nil)
			if len(m.Items) != 1 {
				t.Fatalf("expected 1 item, got %d", len(m.Items))
			}

			item := m.Items[0]
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

	m := NewMenu().AddSeparator()

	if len(m.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(m.Items))
	}

	item := m.Items[0]
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

	sub := NewMenu().Add("SubItem", nil)
	m := NewMenu().AddSubmenu("More", sub)

	if len(m.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(m.Items))
	}

	item := m.Items[0]
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
	innermost := NewMenu().Add("Leaf", nil)
	current := innermost
	for i := depth - 1; i > 0; i-- {
		parent := NewMenu().AddSubmenu("Level", current)
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
	m := NewMenu().AddWithIcon("Copy", icon, nil)

	if len(m.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(m.Items))
	}

	item := m.Items[0]
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

func TestMenu_Chaining(t *testing.T) {
	t.Parallel()

	m := NewMenu().
		Add("File", nil).
		AddSeparator().
		AddCheckbox("Auto-save", true, nil).
		AddSeparator().
		AddSubmenu("Recent", NewMenu().Add("doc1.txt", nil).Add("doc2.txt", nil)).
		AddWithIcon("Paste", []byte{0xFF}, nil).
		Add("Exit", nil)

	if len(m.Items) != 7 {
		t.Fatalf("expected 7 items after chaining, got %d", len(m.Items))
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

func TestMenu_Chaining_ReturnsSamePointer(t *testing.T) {
	t.Parallel()

	m := NewMenu()
	m2 := m.Add("A", nil)
	m3 := m2.AddSeparator()
	m4 := m3.Add("B", nil)

	if m != m2 || m2 != m3 || m3 != m4 {
		t.Error("chaining methods should return the same *Menu pointer")
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

	m := NewMenu().Add("Test", nil)
	item := m.Items[0]

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

	// Chaining on empty menu should still work.
	result := m.AddSeparator()
	if result != m {
		t.Error("AddSeparator on empty menu should return same pointer")
	}
	if len(m.Items) != 1 {
		t.Errorf("expected 1 item after AddSeparator, got %d", len(m.Items))
	}
}
