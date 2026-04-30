package systray

import (
	"testing"

	"github.com/gogpu/systray/internal"
)

// newTestTray creates a SystemTray backed by a mock platform for testing.
// This avoids calling New() which requires actual OS resources.
func newTestTray(t *testing.T) (*SystemTray, *testPlatform) {
	t.Helper()

	mock := &testPlatform{}
	impl := &internal.Tray{
		ID:       internal.NewTrayID(),
		Platform: mock,
	}
	tray := &SystemTray{impl: impl}
	return tray, mock
}

// testPlatform implements internal.PlatformTray for testing.
type testPlatform struct {
	created    bool
	icon       []byte
	tooltip    string
	menu       *internal.Menu
	visible    bool
	destroyed  bool
	notifTitle string
	notifMsg   string
}

func (m *testPlatform) Create() error                     { m.created = true; return nil }
func (m *testPlatform) SetIcon(png []byte) error          { m.icon = png; return nil }
func (m *testPlatform) SetTooltip(text string) error      { m.tooltip = text; return nil }
func (m *testPlatform) SetMenu(menu *internal.Menu) error { m.menu = menu; return nil }
func (m *testPlatform) ShowNotification(title, message string) error {
	m.notifTitle = title
	m.notifMsg = message
	return nil
}
func (m *testPlatform) Show() error                  { m.visible = true; return nil }
func (m *testPlatform) Hide() error                  { m.visible = false; return nil }
func (m *testPlatform) Bounds() (int, int, int, int) { return 10, 20, 24, 24 }
func (m *testPlatform) Run() error                   { return nil }
func (m *testPlatform) Destroy()                     { m.destroyed = true }

// --- ID tests ---

func TestSystemTray_ID_NonZero(t *testing.T) {
	t.Parallel()

	tray, _ := newTestTray(t)
	if tray.ID() == 0 {
		t.Error("ID() returned zero, which is invalid")
	}
}

func TestSystemTray_ID_Unique(t *testing.T) {
	t.Parallel()

	tray1, _ := newTestTray(t)
	tray2, _ := newTestTray(t)

	if tray1.ID() == tray2.ID() {
		t.Errorf("two trays have same ID: %d", tray1.ID())
	}
}

func TestSystemTray_ID_MultipleUnique(t *testing.T) {
	t.Parallel()

	const count = 20
	seen := make(map[TrayID]bool, count)

	for i := 0; i < count; i++ {
		tray, _ := newTestTray(t)
		id := tray.ID()
		if id == 0 {
			t.Fatalf("ID() returned zero on iteration %d", i)
		}
		if seen[id] {
			t.Fatalf("duplicate ID %d on iteration %d", id, i)
		}
		seen[id] = true
	}
}

// --- SetIcon tests ---

func TestSystemTray_SetIcon(t *testing.T) {
	t.Parallel()

	tray, mock := newTestTray(t)
	icon := []byte{0x89, 0x50, 0x4E, 0x47}

	result := tray.SetIcon(icon)

	if result != tray {
		t.Error("SetIcon should return the same *SystemTray for chaining")
	}
	if len(mock.icon) != 4 {
		t.Errorf("platform received icon length = %d, want 4", len(mock.icon))
	}
}

// --- SetDarkModeIcon tests ---

func TestSystemTray_SetDarkModeIcon(t *testing.T) {
	t.Parallel()

	tray, _ := newTestTray(t)
	dark := []byte{0xDE, 0xAD}

	result := tray.SetDarkModeIcon(dark)

	if result != tray {
		t.Error("SetDarkModeIcon should return the same *SystemTray for chaining")
	}
	if len(tray.impl.DarkModeIcon) != 2 {
		t.Errorf("dark mode icon length = %d, want 2", len(tray.impl.DarkModeIcon))
	}
}

// --- SetTemplateIcon tests ---

func TestSystemTray_SetTemplateIcon(t *testing.T) {
	t.Parallel()

	tray, _ := newTestTray(t)
	tmpl := []byte{0xCA, 0xFE}

	result := tray.SetTemplateIcon(tmpl)

	if result != tray {
		t.Error("SetTemplateIcon should return the same *SystemTray for chaining")
	}
	if len(tray.impl.TemplateIcon) != 2 {
		t.Errorf("template icon length = %d, want 2", len(tray.impl.TemplateIcon))
	}
}

// --- SetTooltip tests ---

func TestSystemTray_SetTooltip(t *testing.T) {
	t.Parallel()

	tray, mock := newTestTray(t)

	result := tray.SetTooltip("GoGPU App")

	if result != tray {
		t.Error("SetTooltip should return the same *SystemTray for chaining")
	}
	if mock.tooltip != "GoGPU App" {
		t.Errorf("platform tooltip = %q, want %q", mock.tooltip, "GoGPU App")
	}
}

// --- SetMenu tests ---

func TestSystemTray_SetMenu(t *testing.T) {
	t.Parallel()

	tray, mock := newTestTray(t)
	menu := NewMenu().Add("Open", nil).AddSeparator().Add("Quit", nil)

	result := tray.SetMenu(menu)

	if result != tray {
		t.Error("SetMenu should return the same *SystemTray for chaining")
	}
	if mock.menu == nil {
		t.Fatal("platform menu is nil")
	}
	if len(mock.menu.Items) != 3 {
		t.Errorf("platform menu has %d items, want 3", len(mock.menu.Items))
	}
}

// --- OnClick / OnDoubleClick / OnRightClick tests ---

func TestSystemTray_OnClick(t *testing.T) {
	t.Parallel()

	tray, _ := newTestTray(t)

	called := false
	result := tray.OnClick(func() { called = true })

	if result != tray {
		t.Error("OnClick should return the same *SystemTray for chaining")
	}
	if tray.impl.Callbacks.OnClick == nil {
		t.Fatal("OnClick callback is nil")
	}

	tray.impl.Callbacks.OnClick()
	if !called {
		t.Error("OnClick callback was not invoked")
	}
}

func TestSystemTray_OnDoubleClick(t *testing.T) {
	t.Parallel()

	tray, _ := newTestTray(t)

	called := false
	result := tray.OnDoubleClick(func() { called = true })

	if result != tray {
		t.Error("OnDoubleClick should return the same *SystemTray for chaining")
	}
	if tray.impl.Callbacks.OnDoubleClick == nil {
		t.Fatal("OnDoubleClick callback is nil")
	}

	tray.impl.Callbacks.OnDoubleClick()
	if !called {
		t.Error("OnDoubleClick callback was not invoked")
	}
}

func TestSystemTray_OnRightClick(t *testing.T) {
	t.Parallel()

	tray, _ := newTestTray(t)

	called := false
	result := tray.OnRightClick(func() { called = true })

	if result != tray {
		t.Error("OnRightClick should return the same *SystemTray for chaining")
	}
	if tray.impl.Callbacks.OnRightClick == nil {
		t.Fatal("OnRightClick callback is nil")
	}

	tray.impl.Callbacks.OnRightClick()
	if !called {
		t.Error("OnRightClick callback was not invoked")
	}
}

// --- ShowNotification tests ---

func TestSystemTray_ShowNotification(t *testing.T) {
	t.Parallel()

	tray, mock := newTestTray(t)

	result := tray.ShowNotification("Update", "Version 2.0 available")

	if result != tray {
		t.Error("ShowNotification should return the same *SystemTray for chaining")
	}
	if mock.notifTitle != "Update" {
		t.Errorf("notification title = %q, want %q", mock.notifTitle, "Update")
	}
	if mock.notifMsg != "Version 2.0 available" {
		t.Errorf("notification message = %q, want %q", mock.notifMsg, "Version 2.0 available")
	}
}

// --- Show / Hide tests ---

func TestSystemTray_Show(t *testing.T) {
	t.Parallel()

	tray, mock := newTestTray(t)

	result := tray.Show()

	if result != tray {
		t.Error("Show should return the same *SystemTray for chaining")
	}
	if !mock.visible {
		t.Error("platform should be visible after Show()")
	}
}

func TestSystemTray_Hide(t *testing.T) {
	t.Parallel()

	tray, mock := newTestTray(t)

	_ = tray.Show()
	result := tray.Hide()

	if result != tray {
		t.Error("Hide should return the same *SystemTray for chaining")
	}
	if mock.visible {
		t.Error("platform should not be visible after Hide()")
	}
}

// --- Bounds tests ---

func TestSystemTray_Bounds(t *testing.T) {
	t.Parallel()

	tray, _ := newTestTray(t)

	x, y, w, h := tray.Bounds()
	if x != 10 || y != 20 || w != 24 || h != 24 {
		t.Errorf("Bounds() = (%d, %d, %d, %d), want (10, 20, 24, 24)", x, y, w, h)
	}
}

// --- Run tests ---

func TestSystemTray_Run(t *testing.T) {
	t.Parallel()

	tray, _ := newTestTray(t)

	err := tray.Run()
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

// --- Remove tests ---

func TestSystemTray_Remove(t *testing.T) {
	t.Parallel()

	tray, mock := newTestTray(t)

	_ = tray.Show()
	tray.Remove()

	if !mock.destroyed {
		t.Error("platform.Destroy should have been called")
	}
}

// --- Builder pattern chaining tests ---

func TestSystemTray_BuilderChaining(t *testing.T) {
	t.Parallel()

	tray, mock := newTestTray(t)
	menu := NewMenu().Add("Quit", nil)

	// Full builder chain.
	result := tray.
		SetIcon([]byte{0x01}).
		SetDarkModeIcon([]byte{0x02}).
		SetTemplateIcon([]byte{0x03}).
		SetTooltip("Builder Test").
		SetMenu(menu).
		OnClick(func() {}).
		OnDoubleClick(func() {}).
		OnRightClick(func() {}).
		ShowNotification("Title", "Body").
		Show()

	if result != tray {
		t.Error("full builder chain should return the same *SystemTray")
	}
	if len(mock.icon) != 1 {
		t.Errorf("icon not set via builder chain")
	}
	if mock.tooltip != "Builder Test" {
		t.Errorf("tooltip not set via builder chain: %q", mock.tooltip)
	}
	if mock.menu == nil {
		t.Error("menu not set via builder chain")
	}
	if !mock.visible {
		t.Error("not visible after builder chain")
	}
	if mock.notifTitle != "Title" {
		t.Errorf("notification not sent via builder chain")
	}
}

func TestSystemTray_BuilderChaining_MinimalQuickStart(t *testing.T) {
	t.Parallel()

	// Replicate the quick start from doc.go:
	//   tray := systray.New()
	//   tray.SetIcon(iconPNG).SetTooltip("My App").SetMenu(menu).Show()
	tray, mock := newTestTray(t)
	menu := NewMenu().Add("Quit", nil)

	tray.SetIcon([]byte{0xFF}).SetTooltip("My App").SetMenu(menu).Show()

	if !mock.visible {
		t.Error("quick start builder pattern did not make tray visible")
	}
}

// --- Callback pointer sharing test ---

func TestSystemTray_CallbackPointerSharing(t *testing.T) {
	t.Parallel()

	// Verify that callbacks set via OnClick/OnDoubleClick/OnRightClick on
	// the public SystemTray are visible through impl.Callbacks. This is the
	// core of the pointer sharing design: platform code holds a pointer to
	// impl.Callbacks and sees updates made after creation.
	tray, _ := newTestTray(t)

	clickCalled := false
	dblClickCalled := false
	rightClickCalled := false

	tray.OnClick(func() { clickCalled = true })
	tray.OnDoubleClick(func() { dblClickCalled = true })
	tray.OnRightClick(func() { rightClickCalled = true })

	// Platform would call these via the Callbacks pointer.
	tray.impl.Callbacks.OnClick()
	tray.impl.Callbacks.OnDoubleClick()
	tray.impl.Callbacks.OnRightClick()

	if !clickCalled {
		t.Error("OnClick not callable through impl.Callbacks")
	}
	if !dblClickCalled {
		t.Error("OnDoubleClick not callable through impl.Callbacks")
	}
	if !rightClickCalled {
		t.Error("OnRightClick not callable through impl.Callbacks")
	}
}
