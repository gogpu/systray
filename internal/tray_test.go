package internal

import (
	"testing"
)

// mockPlatformTray implements PlatformTray for testing without OS dependencies.
type mockPlatformTray struct {
	created    bool
	icon       []byte
	tooltip    string
	menu       *Menu
	visible    bool
	hidden     bool
	destroyed  bool
	notifTitle string
	notifMsg   string
	runCalled  bool
	boundsX    int
	boundsY    int
	boundsW    int
	boundsH    int

	// Error injection for testing error paths.
	createErr     error
	setIconErr    error
	setTooltipErr error
	setMenuErr    error
	showErr       error
	hideErr       error
	showNotifErr  error
	runErr        error
}

func (m *mockPlatformTray) Create() error {
	if m.createErr != nil {
		return m.createErr
	}
	m.created = true
	return nil
}

func (m *mockPlatformTray) SetIcon(png []byte) error {
	if m.setIconErr != nil {
		return m.setIconErr
	}
	m.icon = png
	return nil
}

func (m *mockPlatformTray) SetTooltip(text string) error {
	if m.setTooltipErr != nil {
		return m.setTooltipErr
	}
	m.tooltip = text
	return nil
}

func (m *mockPlatformTray) SetMenu(menu *Menu) error {
	if m.setMenuErr != nil {
		return m.setMenuErr
	}
	m.menu = menu
	return nil
}

func (m *mockPlatformTray) ShowNotification(title, message string) error {
	if m.showNotifErr != nil {
		return m.showNotifErr
	}
	m.notifTitle = title
	m.notifMsg = message
	return nil
}

func (m *mockPlatformTray) Show() error {
	if m.showErr != nil {
		return m.showErr
	}
	m.visible = true
	m.hidden = false
	return nil
}

func (m *mockPlatformTray) Hide() error {
	if m.hideErr != nil {
		return m.hideErr
	}
	m.visible = false
	m.hidden = true
	return nil
}

func (m *mockPlatformTray) Bounds() (int, int, int, int) {
	return m.boundsX, m.boundsY, m.boundsW, m.boundsH
}

func (m *mockPlatformTray) Run() error {
	m.runCalled = true
	return m.runErr
}

func (m *mockPlatformTray) Destroy() {
	m.destroyed = true
}

// --- NewTrayID tests ---

func TestNewTrayID_NonZero(t *testing.T) {
	t.Parallel()

	id := NewTrayID()
	if id == 0 {
		t.Error("NewTrayID returned zero, which is invalid")
	}
}

func TestNewTrayID_Unique(t *testing.T) {
	t.Parallel()

	const count = 100
	seen := make(map[TrayID]bool, count)

	for i := 0; i < count; i++ {
		id := NewTrayID()
		if id == 0 {
			t.Fatalf("NewTrayID returned zero on iteration %d", i)
		}
		if seen[id] {
			t.Fatalf("duplicate TrayID %d on iteration %d", id, i)
		}
		seen[id] = true
	}
}

func TestNewTrayID_MonotonicallyIncreasing(t *testing.T) {
	t.Parallel()

	prev := NewTrayID()
	for i := 0; i < 10; i++ {
		next := NewTrayID()
		if next <= prev {
			t.Errorf("NewTrayID not monotonically increasing: prev=%d, next=%d", prev, next)
		}
		prev = next
	}
}

// --- NewTray tests ---

func TestNewTray_Defaults(t *testing.T) {
	t.Parallel()

	mock := &mockPlatformTray{}
	tray := NewTray(mock)

	if tray.ID == 0 {
		t.Error("NewTray should assign a non-zero ID")
	}
	if tray.Platform != mock {
		t.Error("NewTray should store the provided platform")
	}
	if tray.Visible {
		t.Error("new tray should not be visible by default")
	}
	if tray.Tooltip != "" {
		t.Errorf("new tray should have empty tooltip, got %q", tray.Tooltip)
	}
	if tray.Icon != nil {
		t.Error("new tray should have nil icon")
	}
	if tray.Menu != nil {
		t.Error("new tray should have nil menu")
	}
}

// --- Tray.SetIcon tests ---

func TestTray_SetIcon(t *testing.T) {
	t.Parallel()

	mock := &mockPlatformTray{}
	tray := NewTray(mock)

	icon := []byte{0x89, 0x50, 0x4E, 0x47}
	err := tray.SetIcon(icon)
	if err != nil {
		t.Fatalf("SetIcon returned error: %v", err)
	}

	// Verify icon stored in tray state.
	if len(tray.Icon) != 4 {
		t.Errorf("tray.Icon length = %d, want 4", len(tray.Icon))
	}

	// Verify icon forwarded to platform.
	if len(mock.icon) != 4 {
		t.Errorf("platform received icon length = %d, want 4", len(mock.icon))
	}
}

// --- Tray.SetDarkModeIcon tests ---

func TestTray_SetDarkModeIcon_StoresData(t *testing.T) {
	t.Parallel()

	mock := &mockPlatformTray{}
	tray := NewTray(mock)

	dark := []byte{0x01, 0x02, 0x03}
	err := tray.SetDarkModeIcon(dark)
	if err != nil {
		t.Fatalf("SetDarkModeIcon returned error: %v", err)
	}

	if len(tray.DarkModeIcon) != 3 {
		t.Errorf("tray.DarkModeIcon length = %d, want 3", len(tray.DarkModeIcon))
	}
}

// --- Tray.SetTemplateIcon tests ---

func TestTray_SetTemplateIcon_StoresData(t *testing.T) {
	t.Parallel()

	mock := &mockPlatformTray{}
	tray := NewTray(mock)

	tmpl := []byte{0xAA, 0xBB}
	tray.SetTemplateIcon(tmpl)

	if len(tray.TemplateIcon) != 2 {
		t.Errorf("tray.TemplateIcon length = %d, want 2", len(tray.TemplateIcon))
	}
}

// --- Tray.SetTooltip tests ---

func TestTray_SetTooltip(t *testing.T) {
	t.Parallel()

	mock := &mockPlatformTray{}
	tray := NewTray(mock)

	err := tray.SetTooltip("My App v1.0")
	if err != nil {
		t.Fatalf("SetTooltip returned error: %v", err)
	}

	if tray.Tooltip != "My App v1.0" {
		t.Errorf("tray.Tooltip = %q, want %q", tray.Tooltip, "My App v1.0")
	}
	if mock.tooltip != "My App v1.0" {
		t.Errorf("platform.tooltip = %q, want %q", mock.tooltip, "My App v1.0")
	}
}

func TestTray_SetTooltip_Empty(t *testing.T) {
	t.Parallel()

	mock := &mockPlatformTray{}
	tray := NewTray(mock)

	err := tray.SetTooltip("")
	if err != nil {
		t.Fatalf("SetTooltip returned error: %v", err)
	}

	if tray.Tooltip != "" {
		t.Errorf("tray.Tooltip = %q, want empty", tray.Tooltip)
	}
}

// --- Tray.SetMenu tests ---

func TestTray_SetMenu(t *testing.T) {
	t.Parallel()

	mock := &mockPlatformTray{}
	tray := NewTray(mock)
	menu := NewMenu().Add("Item1", nil).Add("Item2", nil)

	err := tray.SetMenu(menu)
	if err != nil {
		t.Fatalf("SetMenu returned error: %v", err)
	}

	if tray.Menu != menu {
		t.Error("tray.Menu should reference the provided menu")
	}
	if mock.menu != menu {
		t.Error("platform.menu should reference the provided menu")
	}
}

// --- Tray.Show / Hide tests ---

func TestTray_Show(t *testing.T) {
	t.Parallel()

	mock := &mockPlatformTray{}
	tray := NewTray(mock)

	err := tray.Show()
	if err != nil {
		t.Fatalf("Show returned error: %v", err)
	}

	if !tray.Visible {
		t.Error("tray.Visible should be true after Show()")
	}
	if !mock.visible {
		t.Error("platform.Show should have been called")
	}
}

func TestTray_Hide(t *testing.T) {
	t.Parallel()

	mock := &mockPlatformTray{}
	tray := NewTray(mock)

	// Show first, then hide.
	_ = tray.Show()
	err := tray.Hide()
	if err != nil {
		t.Fatalf("Hide returned error: %v", err)
	}

	if tray.Visible {
		t.Error("tray.Visible should be false after Hide()")
	}
	if !mock.hidden {
		t.Error("platform.Hide should have been called")
	}
}

func TestTray_ShowHideToggle(t *testing.T) {
	t.Parallel()

	mock := &mockPlatformTray{}
	tray := NewTray(mock)

	_ = tray.Show()
	if !tray.Visible {
		t.Error("should be visible after Show()")
	}

	_ = tray.Hide()
	if tray.Visible {
		t.Error("should not be visible after Hide()")
	}

	_ = tray.Show()
	if !tray.Visible {
		t.Error("should be visible after second Show()")
	}
}

// --- Tray.ShowNotification tests ---

func TestTray_ShowNotification(t *testing.T) {
	t.Parallel()

	mock := &mockPlatformTray{}
	tray := NewTray(mock)

	err := tray.ShowNotification("Alert", "Something happened")
	if err != nil {
		t.Fatalf("ShowNotification returned error: %v", err)
	}

	if mock.notifTitle != "Alert" {
		t.Errorf("notification title = %q, want %q", mock.notifTitle, "Alert")
	}
	if mock.notifMsg != "Something happened" {
		t.Errorf("notification message = %q, want %q", mock.notifMsg, "Something happened")
	}
}

// --- Tray.Bounds tests ---

func TestTray_Bounds(t *testing.T) {
	t.Parallel()

	mock := &mockPlatformTray{
		boundsX: 100,
		boundsY: 200,
		boundsW: 24,
		boundsH: 24,
	}
	tray := NewTray(mock)

	x, y, w, h := tray.Bounds()
	if x != 100 || y != 200 || w != 24 || h != 24 {
		t.Errorf("Bounds() = (%d, %d, %d, %d), want (100, 200, 24, 24)", x, y, w, h)
	}
}

// --- Tray.Remove tests ---

func TestTray_Remove(t *testing.T) {
	t.Parallel()

	mock := &mockPlatformTray{}
	tray := NewTray(mock)

	_ = tray.Show()
	tray.Remove()

	if tray.Visible {
		t.Error("tray.Visible should be false after Remove()")
	}
	if !mock.destroyed {
		t.Error("platform.Destroy should have been called")
	}
}

func TestTray_Remove_WhenNotVisible(t *testing.T) {
	t.Parallel()

	mock := &mockPlatformTray{}
	tray := NewTray(mock)

	tray.Remove()

	if tray.Visible {
		t.Error("tray.Visible should be false after Remove()")
	}
	if !mock.destroyed {
		t.Error("platform.Destroy should have been called even when not visible")
	}
}

// --- Tray.Run tests ---

func TestTray_Run(t *testing.T) {
	t.Parallel()

	mock := &mockPlatformTray{}
	tray := NewTray(mock)

	err := tray.Run()
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !mock.runCalled {
		t.Error("platform.Run should have been called")
	}
}

// --- Callbacks tests ---

func TestTray_Callbacks_OnClick(t *testing.T) {
	t.Parallel()

	mock := &mockPlatformTray{}
	tray := NewTray(mock)

	called := false
	tray.Callbacks.OnClick = func() { called = true }

	if tray.Callbacks.OnClick == nil {
		t.Fatal("OnClick should be set")
	}

	tray.Callbacks.OnClick()
	if !called {
		t.Error("OnClick callback was not invoked")
	}
}

func TestTray_Callbacks_OnDoubleClick(t *testing.T) {
	t.Parallel()

	mock := &mockPlatformTray{}
	tray := NewTray(mock)

	called := false
	tray.Callbacks.OnDoubleClick = func() { called = true }

	if tray.Callbacks.OnDoubleClick == nil {
		t.Fatal("OnDoubleClick should be set")
	}

	tray.Callbacks.OnDoubleClick()
	if !called {
		t.Error("OnDoubleClick callback was not invoked")
	}
}

func TestTray_Callbacks_OnRightClick(t *testing.T) {
	t.Parallel()

	mock := &mockPlatformTray{}
	tray := NewTray(mock)

	called := false
	tray.Callbacks.OnRightClick = func() { called = true }

	if tray.Callbacks.OnRightClick == nil {
		t.Fatal("OnRightClick should be set")
	}

	tray.Callbacks.OnRightClick()
	if !called {
		t.Error("OnRightClick callback was not invoked")
	}
}

func TestTray_Callbacks_SharedPointer(t *testing.T) {
	t.Parallel()

	// Verify that Callbacks is stored by value in Tray, but the public API
	// in systray.go passes a pointer to impl.Callbacks to NewPlatformTray.
	// This test verifies the basic struct behavior.
	tray := &Tray{
		ID: NewTrayID(),
	}

	callbacksPtr := &tray.Callbacks

	called := false
	callbacksPtr.OnClick = func() { called = true }

	if tray.Callbacks.OnClick == nil {
		t.Fatal("setting via pointer should update the struct field")
	}

	tray.Callbacks.OnClick()
	if !called {
		t.Error("callback set via pointer should be callable via struct")
	}
}

func TestTray_Callbacks_DefaultNil(t *testing.T) {
	t.Parallel()

	tray := &Tray{ID: NewTrayID()}

	if tray.Callbacks.OnClick != nil {
		t.Error("default OnClick should be nil")
	}
	if tray.Callbacks.OnDoubleClick != nil {
		t.Error("default OnDoubleClick should be nil")
	}
	if tray.Callbacks.OnRightClick != nil {
		t.Error("default OnRightClick should be nil")
	}
}

// --- SetDarkModeIcon with optional platform interface ---

type mockDarkModePlatform struct {
	mockPlatformTray
	darkIcon []byte
}

func (m *mockDarkModePlatform) SetDarkModeIcon(png []byte) error {
	m.darkIcon = png
	return nil
}

func TestTray_SetDarkModeIcon_WithSupportingPlatform(t *testing.T) {
	t.Parallel()

	mock := &mockDarkModePlatform{}
	tray := &Tray{
		ID:       NewTrayID(),
		Platform: mock,
	}

	dark := []byte{0xDE, 0xAD}
	err := tray.SetDarkModeIcon(dark)
	if err != nil {
		t.Fatalf("SetDarkModeIcon returned error: %v", err)
	}

	if len(tray.DarkModeIcon) != 2 {
		t.Errorf("tray.DarkModeIcon length = %d, want 2", len(tray.DarkModeIcon))
	}
	if len(mock.darkIcon) != 2 {
		t.Errorf("platform.darkIcon length = %d, want 2", len(mock.darkIcon))
	}
}

func TestTray_SetDarkModeIcon_WithoutSupportingPlatform(t *testing.T) {
	t.Parallel()

	// Basic mockPlatformTray does NOT implement SetDarkModeIcon interface.
	mock := &mockPlatformTray{}
	tray := &Tray{
		ID:       NewTrayID(),
		Platform: mock,
	}

	dark := []byte{0xDE, 0xAD}
	err := tray.SetDarkModeIcon(dark)
	if err != nil {
		t.Fatalf("SetDarkModeIcon should succeed even without platform support: %v", err)
	}

	// Data should still be stored in tray state.
	if len(tray.DarkModeIcon) != 2 {
		t.Errorf("tray.DarkModeIcon length = %d, want 2", len(tray.DarkModeIcon))
	}
}

// --- SetTemplateIcon with optional platform interface ---

type mockTemplateIconPlatform struct {
	mockPlatformTray
	templateIcon []byte
}

func (m *mockTemplateIconPlatform) SetTemplateIcon(png []byte) error {
	m.templateIcon = png
	return nil
}

func TestTray_SetTemplateIcon_WithSupportingPlatform(t *testing.T) {
	t.Parallel()

	mock := &mockTemplateIconPlatform{}
	tray := &Tray{
		ID:       NewTrayID(),
		Platform: mock,
	}

	tmpl := []byte{0xCA, 0xFE}
	tray.SetTemplateIcon(tmpl)

	if len(tray.TemplateIcon) != 2 {
		t.Errorf("tray.TemplateIcon length = %d, want 2", len(tray.TemplateIcon))
	}
	if len(mock.templateIcon) != 2 {
		t.Errorf("platform.templateIcon length = %d, want 2", len(mock.templateIcon))
	}
}

func TestTray_SetTemplateIcon_WithoutSupportingPlatform(t *testing.T) {
	t.Parallel()

	mock := &mockPlatformTray{}
	tray := &Tray{
		ID:       NewTrayID(),
		Platform: mock,
	}

	tmpl := []byte{0xCA, 0xFE}
	tray.SetTemplateIcon(tmpl)

	// Should still store data even without platform support.
	if len(tray.TemplateIcon) != 2 {
		t.Errorf("tray.TemplateIcon length = %d, want 2", len(tray.TemplateIcon))
	}
}
