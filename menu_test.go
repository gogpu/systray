package systray

import (
	"testing"
)

func TestNewMenu_Creates(t *testing.T) {
	t.Parallel()

	m := NewMenu()
	if m == nil {
		t.Fatal("NewMenu returned nil")
	}
	if m.impl == nil {
		t.Fatal("NewMenu().impl is nil")
	}
}

func TestMenu_Add(t *testing.T) {
	t.Parallel()

	called := false
	m := NewMenu().Add("Open", func() { called = true })

	if m == nil {
		t.Fatal("Add returned nil")
	}
	if len(m.impl.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(m.impl.Items))
	}

	item := m.impl.Items[0]
	if item.Label != "Open" {
		t.Errorf("label = %q, want %q", item.Label, "Open")
	}
	if item.Type != MenuItemNormal {
		t.Errorf("type = %d, want MenuItemNormal (%d)", item.Type, MenuItemNormal)
	}

	// Verify callback works.
	item.OnClick()
	if !called {
		t.Error("OnClick callback was not invoked")
	}
}

func TestMenu_AddCheckbox(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		label   string
		checked bool
	}{
		{"checked", "Enable", true},
		{"unchecked", "Disable", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := NewMenu().AddCheckbox(tt.label, tt.checked, nil)

			if len(m.impl.Items) != 1 {
				t.Fatalf("expected 1 item, got %d", len(m.impl.Items))
			}

			item := m.impl.Items[0]
			if item.Label != tt.label {
				t.Errorf("label = %q, want %q", item.Label, tt.label)
			}
			if item.Type != MenuItemCheckbox {
				t.Errorf("type = %d, want MenuItemCheckbox", item.Type)
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

	if len(m.impl.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(m.impl.Items))
	}

	if m.impl.Items[0].Type != MenuItemSeparator {
		t.Errorf("type = %d, want MenuItemSeparator", m.impl.Items[0].Type)
	}
}

func TestMenu_AddSubmenu(t *testing.T) {
	t.Parallel()

	sub := NewMenu().Add("SubItem", nil)
	m := NewMenu().AddSubmenu("More", sub)

	if len(m.impl.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(m.impl.Items))
	}

	item := m.impl.Items[0]
	if item.Label != "More" {
		t.Errorf("label = %q, want %q", item.Label, "More")
	}
	if item.Type != MenuItemSubmenu {
		t.Errorf("type = %d, want MenuItemSubmenu", item.Type)
	}
	if item.Submenu == nil {
		t.Fatal("submenu is nil")
	}
	if len(item.Submenu.Items) != 1 {
		t.Errorf("submenu has %d items, want 1", len(item.Submenu.Items))
	}
}

func TestMenu_AddWithIcon(t *testing.T) {
	t.Parallel()

	icon := []byte{0x89, 0x50, 0x4E, 0x47}
	m := NewMenu().AddWithIcon("Paste", icon, nil)

	if len(m.impl.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(m.impl.Items))
	}

	item := m.impl.Items[0]
	if item.Label != "Paste" {
		t.Errorf("label = %q, want %q", item.Label, "Paste")
	}
	if item.Type != MenuItemNormal {
		t.Errorf("type = %d, want MenuItemNormal", item.Type)
	}
	if len(item.Icon) != 4 {
		t.Errorf("icon length = %d, want 4", len(item.Icon))
	}
}

func TestMenu_Chaining(t *testing.T) {
	t.Parallel()

	sub := NewMenu().Add("Recent1", nil).Add("Recent2", nil)

	m := NewMenu().
		Add("Open", nil).
		AddSeparator().
		AddCheckbox("Auto-save", true, nil).
		AddSubmenu("Recent", sub).
		AddWithIcon("Copy", []byte{0xFF}, nil).
		Add("Quit", nil)

	if len(m.impl.Items) != 6 {
		t.Fatalf("expected 6 items, got %d", len(m.impl.Items))
	}

	expectedTypes := []MenuItemType{
		MenuItemNormal,    // Open
		MenuItemSeparator, // ---
		MenuItemCheckbox,  // Auto-save
		MenuItemSubmenu,   // Recent
		MenuItemNormal,    // Copy (with icon)
		MenuItemNormal,    // Quit
	}

	expectedLabels := []string{
		"Open",
		"",
		"Auto-save",
		"Recent",
		"Copy",
		"Quit",
	}

	for i := range expectedTypes {
		if m.impl.Items[i].Type != expectedTypes[i] {
			t.Errorf("item[%d].Type = %d, want %d", i, m.impl.Items[i].Type, expectedTypes[i])
		}
		if m.impl.Items[i].Label != expectedLabels[i] {
			t.Errorf("item[%d].Label = %q, want %q", i, m.impl.Items[i].Label, expectedLabels[i])
		}
	}
}

func TestMenu_Chaining_ReturnsSamePointer(t *testing.T) {
	t.Parallel()

	m := NewMenu()
	m2 := m.Add("A", nil)
	m3 := m2.AddSeparator()
	m4 := m3.AddCheckbox("B", false, nil)
	m5 := m4.AddSubmenu("C", NewMenu())
	m6 := m5.AddWithIcon("D", []byte{0x01}, nil)

	if m != m2 || m2 != m3 || m3 != m4 || m4 != m5 || m5 != m6 {
		t.Error("all chaining methods should return the same *Menu pointer")
	}
}

func TestMenu_NestedSubmenus(t *testing.T) {
	t.Parallel()

	level3 := NewMenu().Add("Leaf", nil)
	level2 := NewMenu().AddSubmenu("Level3", level3)
	level1 := NewMenu().AddSubmenu("Level2", level2)
	root := NewMenu().AddSubmenu("Level1", level1)

	// Navigate 3 levels deep.
	item := root.impl.Items[0]
	if item.Type != MenuItemSubmenu || item.Label != "Level1" {
		t.Fatalf("level 1: type=%d label=%q", item.Type, item.Label)
	}

	item = item.Submenu.Items[0]
	if item.Type != MenuItemSubmenu || item.Label != "Level2" {
		t.Fatalf("level 2: type=%d label=%q", item.Type, item.Label)
	}

	item = item.Submenu.Items[0]
	if item.Type != MenuItemSubmenu || item.Label != "Level3" {
		t.Fatalf("level 3: type=%d label=%q", item.Type, item.Label)
	}

	leaf := item.Submenu.Items[0]
	if leaf.Type != MenuItemNormal || leaf.Label != "Leaf" {
		t.Errorf("leaf: type=%d label=%q, want Normal/%q", leaf.Type, leaf.Label, "Leaf")
	}
}

func TestMenuItemType_Constants(t *testing.T) {
	t.Parallel()

	// Verify public type aliases match internal values.
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
