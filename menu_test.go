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
	m := NewMenu()
	item := m.Add("Open", func() { called = true })

	if item == nil {
		t.Fatal("Add returned nil")
	}
	if len(m.impl.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(m.impl.Items))
	}
	if item.impl != m.impl.Items[0] {
		t.Error("returned MenuItem should wrap the internal item appended to menu")
	}
	if item.impl.Label != "Open" {
		t.Errorf("label = %q, want %q", item.impl.Label, "Open")
	}
	if item.impl.Type != MenuItemNormal {
		t.Errorf("type = %d, want MenuItemNormal (%d)", item.impl.Type, MenuItemNormal)
	}

	// Verify callback works.
	item.impl.OnClick()
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

			m := NewMenu()
			item := m.AddCheckbox(tt.label, tt.checked, nil)

			if len(m.impl.Items) != 1 {
				t.Fatalf("expected 1 item, got %d", len(m.impl.Items))
			}
			if item.impl.Label != tt.label {
				t.Errorf("label = %q, want %q", item.impl.Label, tt.label)
			}
			if item.impl.Type != MenuItemCheckbox {
				t.Errorf("type = %d, want MenuItemCheckbox", item.impl.Type)
			}
			if item.impl.Checked != tt.checked {
				t.Errorf("checked = %v, want %v", item.impl.Checked, tt.checked)
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

func TestMenu_AddSeparator_Chaining(t *testing.T) {
	t.Parallel()

	m := NewMenu()
	result := m.AddSeparator()

	if result != m {
		t.Error("AddSeparator should return the same *Menu for chaining")
	}
}

func TestMenu_AddSubmenu(t *testing.T) {
	t.Parallel()

	sub := NewMenu()
	sub.Add("SubItem", nil)

	m := NewMenu()
	item := m.AddSubmenu("More", sub)

	if len(m.impl.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(m.impl.Items))
	}
	if item.impl.Label != "More" {
		t.Errorf("label = %q, want %q", item.impl.Label, "More")
	}
	if item.impl.Type != MenuItemSubmenu {
		t.Errorf("type = %d, want MenuItemSubmenu", item.impl.Type)
	}
	if item.impl.Submenu == nil {
		t.Fatal("submenu is nil")
	}
	if len(item.impl.Submenu.Items) != 1 {
		t.Errorf("submenu has %d items, want 1", len(item.impl.Submenu.Items))
	}
}

func TestMenu_AddWithIcon(t *testing.T) {
	t.Parallel()

	icon := []byte{0x89, 0x50, 0x4E, 0x47}
	m := NewMenu()
	item := m.AddWithIcon("Paste", icon, nil)

	if len(m.impl.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(m.impl.Items))
	}
	if item.impl.Label != "Paste" {
		t.Errorf("label = %q, want %q", item.impl.Label, "Paste")
	}
	if item.impl.Type != MenuItemNormal {
		t.Errorf("type = %d, want MenuItemNormal", item.impl.Type)
	}
	if len(item.impl.Icon) != 4 {
		t.Errorf("icon length = %d, want 4", len(item.impl.Icon))
	}
}

func TestMenu_MultipleItems(t *testing.T) {
	t.Parallel()

	recentMenu := NewMenu()
	recentMenu.Add("Recent1", nil)
	recentMenu.Add("Recent2", nil)

	m := NewMenu()
	m.Add("Open", nil)
	m.AddSeparator()
	m.AddCheckbox("Auto-save", true, nil)
	m.AddSubmenu("Recent", recentMenu)
	m.AddWithIcon("Copy", []byte{0xFF}, nil)
	m.Add("Quit", nil)

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

func TestMenu_NestedSubmenus(t *testing.T) {
	t.Parallel()

	level3 := NewMenu()
	level3.Add("Leaf", nil)

	level2 := NewMenu()
	level2.AddSubmenu("Level3", level3)

	level1 := NewMenu()
	level1.AddSubmenu("Level2", level2)

	root := NewMenu()
	root.AddSubmenu("Level1", level1)

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

func TestMenuItem_SetLabel_Public(t *testing.T) {
	t.Parallel()

	m := NewMenu()
	item := m.Add("Original", nil)

	item.SetLabel("Updated")

	if item.impl.Label != "Updated" {
		t.Errorf("label = %q, want %q", item.impl.Label, "Updated")
	}
}

func TestMenuItem_SetChecked_Public(t *testing.T) {
	t.Parallel()

	m := NewMenu()
	item := m.AddCheckbox("Toggle", false, nil)

	item.SetChecked(true)

	if !item.impl.Checked {
		t.Error("checked should be true")
	}
}

func TestMenuItem_SetDisabled_Public(t *testing.T) {
	t.Parallel()

	m := NewMenu()
	item := m.Add("Action", nil)

	item.SetDisabled(true)

	if !item.impl.Disabled {
		t.Error("disabled should be true")
	}
}

func TestMenuItem_SetIcon_Public(t *testing.T) {
	t.Parallel()

	m := NewMenu()
	item := m.Add("Item", nil)

	icon := []byte{0x89, 0x50}
	item.SetIcon(icon)

	if len(item.impl.Icon) != 2 {
		t.Errorf("icon length = %d, want 2", len(item.impl.Icon))
	}
}
