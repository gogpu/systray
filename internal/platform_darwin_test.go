//go:build darwin

package internal

import (
	"testing"

	"github.com/gogpu/systray/internal/darwin"
)

func TestShouldStopDarwinApplication(t *testing.T) {
	tests := []struct {
		name                string
		remainingTrays      int
		wantApplicationStop bool
	}{
		{name: "last tray removed", remainingTrays: 0, wantApplicationStop: true},
		{name: "one tray remains", remainingTrays: 1, wantApplicationStop: false},
		{name: "multiple trays remain", remainingTrays: 2, wantApplicationStop: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldStopDarwinApplication(test.remainingTrays); got != test.wantApplicationStop {
				t.Errorf("shouldStopDarwinApplication(%d) = %v, want %v", test.remainingTrays, got, test.wantApplicationStop)
			}
		})
	}
}

// newTestDarwinTray creates a tray whose click target is registered, without a
// status item: menu trees can be built and clicked without touching NSStatusBar.
func newTestDarwinTray(t *testing.T) *darwinTray {
	t.Helper()

	initDarwinSels()
	initDarwinClasses()

	cls, err := registerGoSystrayTarget()
	if err != nil {
		t.Fatalf("register target class: %v", err)
	}

	target := darwin.ID(cls).Send(darwinSels.alloc)
	target = target.Send(darwinSels.init)
	if target.IsNil() {
		t.Fatal("failed to create GoSystrayTarget")
	}

	tray := NewPlatformTray(nil).(*darwinTray)
	tray.target = target

	trayRegistryMu.Lock()
	trayRegistryMap[target.Ptr()] = tray
	trayRegistryMu.Unlock()

	t.Cleanup(func() {
		trayRegistryMu.Lock()
		delete(trayRegistryMap, target.Ptr())
		trayRegistryMu.Unlock()
	})

	return tray
}

// click delivers an NSMenuItem action the way AppKit does, so the whole
// menuItemClicked: path (tag lookup, owning NSMenu, callback) is exercised.
func click(t *testing.T, tray *darwinTray, sender darwin.ID) {
	t.Helper()
	tray.target.SendPtr(darwin.RegisterSelector("menuItemClicked:"), sender.Ptr())
}

// TestClickAfterPublishResolvesClickedMenu is the macOS regression for the
// reviewed ordering problem: AppKit sends menuDidClose: before the clicked
// item's action, so the rebuild is published first. The click must still reach
// the item of the menu that was on screen, not be dropped by the new build.
func TestClickAfterPublishResolvesClickedMenu(t *testing.T) {
	tray := newTestDarwinTray(t)

	var fired []string
	menuA := NewMenu()
	menuA.Add("Kept", func() { fired = append(fired, "kept") })
	vanishing := menuA.Add("View Logs", func() { fired = append(fired, "view logs") })

	displayed, err := tray.buildMenuTree(menuA)
	if err != nil {
		t.Fatalf("build menu A: %v", err)
	}
	tray.menuMu.Lock()
	tray.adoptBuildLocked(displayed)
	tray.menuOpen = true
	tray.menuMu.Unlock()

	// A rebuild arrives while the menu is on screen; it creates new items, so
	// the displayed item's tag exists only in the old build.
	menuB := NewMenu()
	menuB.Add("Kept", func() { fired = append(fired, "kept again") })
	menuB.Add("Other", func() { fired = append(fired, "other") })
	queued, err := tray.buildMenuTree(menuB)
	if err != nil {
		t.Fatalf("build menu B: %v", err)
	}
	tray.menuMu.Lock()
	tray.pending = queued
	tray.menuMu.Unlock()

	// AppKit order: the menu closes (which publishes the rebuild), then the
	// chosen item's action is delivered.
	tray.menuDidClose()

	if got := tray.currentMenu(); got.Ptr() != queued.menu.Ptr() {
		t.Fatalf("queued menu not published: %#x", got.Ptr())
	}

	click(t, tray, displayed.items[vanishing.ID()])
	if len(fired) != 1 || fired[0] != "view logs" {
		t.Fatalf("fired %v, want [view logs]", fired)
	}

	// A click on the newly published menu reaches its own callback.
	other := menuB.Items[1]
	click(t, tray, queued.items[other.ID()])
	if len(fired) != 2 || fired[1] != "other" {
		t.Fatalf("fired %v, want [view logs other]", fired)
	}

	// A tag from a build nobody owns resolves to nothing.
	if fn := tray.callbackForMenu(displayed.menu.Ptr(), 0xFFFF); fn != nil {
		t.Error("unknown tag resolved to a callback")
	}
}

// TestClickAfterMissedCloseKeepsDispatching covers a delegate that never sent
// menuDidClose: the queued rebuild is applied on the next menuWillOpen, and
// clicks from the menu that was on screen still dispatch.
func TestClickAfterMissedCloseKeepsDispatching(t *testing.T) {
	tray := newTestDarwinTray(t)

	var fired []string
	menuA := NewMenu()
	stale := menuA.Add("Stale", func() { fired = append(fired, "stale") })

	displayed, err := tray.buildMenuTree(menuA)
	if err != nil {
		t.Fatalf("build menu A: %v", err)
	}
	tray.menuMu.Lock()
	tray.adoptBuildLocked(displayed)
	tray.menuOpen = true
	tray.menuMu.Unlock()

	menuB := NewMenu()
	menuB.Add("Fresh", nil)
	queued, err := tray.buildMenuTree(menuB)
	if err != nil {
		t.Fatalf("build menu B: %v", err)
	}
	tray.menuMu.Lock()
	tray.pending = queued
	tray.menuMu.Unlock()

	// The next open recovers the queued build and leaves the old one
	// dispatchable for the click that is still in flight.
	tray.menuWillOpen()

	if got := tray.currentMenu(); got.Ptr() != queued.menu.Ptr() {
		t.Fatalf("queued menu not recovered: %#x", got.Ptr())
	}
	click(t, tray, displayed.items[stale.ID()])
	if len(fired) != 1 || fired[0] != "stale" {
		t.Fatalf("fired %v, want [stale]", fired)
	}
}

// TestUpdateReachesQueuedMenu covers an update that arrives between a queued
// SetMenu and its publish: the queued NSMenu is not on screen yet, but it is
// the one that will be, so it is updated as well.
func TestUpdateReachesQueuedMenu(t *testing.T) {
	tray := newTestDarwinTray(t)

	menuA := NewMenu()
	menuA.Add("Shown", nil)
	displayed, err := tray.buildMenuTree(menuA)
	if err != nil {
		t.Fatalf("build menu A: %v", err)
	}
	tray.menuMu.Lock()
	tray.adoptBuildLocked(displayed)
	tray.menuOpen = true
	tray.menuMu.Unlock()

	shared := NewMenu().Add("Shared", nil)
	sharedCheckbox := NewMenu().AddCheckbox("Check", true, nil)
	menuB := NewMenu()
	menuB.Items = append(menuB.Items, shared, sharedCheckbox)
	queued, err := tray.buildMenuTree(menuB)
	if err != nil {
		t.Fatalf("build menu B: %v", err)
	}
	tray.menuMu.Lock()
	tray.pending = queued
	tray.menuMu.Unlock()

	if !tray.tracksItem(shared.ID()) {
		t.Fatal("item of the queued menu is not tracked")
	}
	if handles := tray.itemHandles(shared.ID()); len(handles) != 1 || handles[0].Ptr() != queued.items[shared.ID()].Ptr() {
		t.Fatalf("handles = %v, want the queued menu item", handles)
	}

	// Applying the update natively touches the queued NSMenuItem only.
	shared.Label = "Renamed"
	sharedCheckbox.Checked = false
	tray.applyItemUpdate(shared.snapshot())
	tray.applyItemUpdate(sharedCheckbox.snapshot())

	title := queued.items[shared.ID()].Send(darwin.RegisterSelector("title"))
	if title.IsNil() {
		t.Fatal("queued menu item has no title")
	}
	// isEqualToString: returns a BOOL, which SendPtr surfaces as 0 or 1.
	want := darwin.NewNSString("Renamed")
	if title.SendPtr(darwin.RegisterSelector("isEqualToString:"), want.Ptr()).IsNil() {
		t.Error("queued menu item title was not updated")
	}
	// NSControlStateValueOff is 0.
	if state := queued.items[sharedCheckbox.ID()].Send(darwin.RegisterSelector("state")); !state.IsNil() {
		t.Errorf("queued checkbox state = %v, want off", state.Ptr())
	}
}

// TestMenuBuildRetention keeps history bounded: the menu on screen and the one
// it replaced stay addressable, older ones are dropped.
func TestMenuBuildRetention(t *testing.T) {
	tray := &darwinTray{}

	for i := 0; i < menuBuildRetention+3; i++ {
		build := newMenuBuild()
		tray.menuMu.Lock()
		tray.adoptBuildLocked(build)
		tray.menuMu.Unlock()
	}

	tray.menuMu.Lock()
	n := len(tray.builds)
	tray.menuMu.Unlock()
	if n != menuBuildRetention {
		t.Fatalf("retained builds = %d, want %d", n, menuBuildRetention)
	}
}
