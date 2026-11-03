//go:build darwin

package internal

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/go-webgpu/goffi/ffi"

	"github.com/gogpu/systray/internal/darwin"
)

// NSVariableStatusItemLength tells NSStatusBar to size the item to fit its content.
const nsVariableStatusItemLength = -1.0

// NSApplicationActivationPolicyAccessory creates an app without dock icon.
// Systray apps typically run as accessories, not as regular dock-visible apps.
const nsApplicationActivationPolicyAccessory = 1

// Selectors used by the darwin tray implementation.
// Registered lazily on first use.
var darwinSels struct {
	once sync.Once

	// NSObject
	alloc   darwin.SEL
	init    darwin.SEL
	release darwin.SEL

	// NSApplication
	sharedApplication     darwin.SEL
	setActivationPolicy   darwin.SEL
	run                   darwin.SEL
	stop                  darwin.SEL
	finishLaunching       darwin.SEL
	nextEventMatchingMask darwin.SEL // nextEventMatchingMask:untilDate:inMode:dequeue:
	sendEvent             darwin.SEL

	// NSStatusBar
	systemStatusBar   darwin.SEL
	statusItemWithLen darwin.SEL // statusItemWithLength:
	removeStatusItem  darwin.SEL // removeStatusItem:

	// NSStatusItem
	button  darwin.SEL
	setMenu darwin.SEL // setMenu:

	// NSStatusBarButton (NSButton subclass)
	setImage   darwin.SEL // setImage:
	setToolTip darwin.SEL // setToolTip:
	setTarget  darwin.SEL // setTarget:
	setAction  darwin.SEL // setAction:

	// NSImage
	initWithData darwin.SEL // initWithData:
	setSize      darwin.SEL // setSize:
	setTemplate  darwin.SEL // setTemplate:

	// NSMenu
	initWithTitle               darwin.SEL
	addItem                     darwin.SEL // addItem:
	separatorItem               darwin.SEL
	setSubmenu                  darwin.SEL // setSubmenu:
	initWithTitleActionKeyEquiv darwin.SEL // initWithTitle:action:keyEquivalent:
	setState                    darwin.SEL // setState:
	setEnabled                  darwin.SEL // setEnabled:
	setDelegate                 darwin.SEL // setDelegate:
	menuWillOpen                darwin.SEL // NSMenuDelegate
	menuDidClose                darwin.SEL // NSMenuDelegate

	// NSMenuItem
	menu darwin.SEL // menu (the NSMenu the item belongs to)

	// NSObject main-thread dispatch
	performSelectorOnMainThread darwin.SEL // performSelectorOnMainThread:withObject:waitUntilDone:

	// NSDate
	distantPast   darwin.SEL
	distantFuture darwin.SEL

	// NSUserNotificationCenter
	defaultUserNotificationCenter darwin.SEL
	deliverNotification           darwin.SEL // deliverNotification:

	// NSUserNotification
	setTitle           darwin.SEL // setTitle:
	setInformativeText darwin.SEL // setInformativeText:
}

// Classes used by the darwin tray implementation.
var darwinClasses struct {
	once sync.Once

	NSApplication            darwin.Class
	NSStatusBar              darwin.Class
	NSImage                  darwin.Class
	NSMenu                   darwin.Class
	NSMenuItem               darwin.Class
	NSDate                   darwin.Class
	NSAutoreleasePool        darwin.Class
	NSUserNotificationCenter darwin.Class
	NSUserNotification       darwin.Class
}

func initDarwinSels() {
	darwinSels.once.Do(func() {
		// NSObject
		darwinSels.alloc = darwin.RegisterSelector("alloc")
		darwinSels.init = darwin.RegisterSelector("init")
		darwinSels.release = darwin.RegisterSelector("release")

		// NSApplication
		darwinSels.sharedApplication = darwin.RegisterSelector("sharedApplication")
		darwinSels.setActivationPolicy = darwin.RegisterSelector("setActivationPolicy:")
		darwinSels.run = darwin.RegisterSelector("run")
		darwinSels.stop = darwin.RegisterSelector("stop:")
		darwinSels.finishLaunching = darwin.RegisterSelector("finishLaunching")
		darwinSels.nextEventMatchingMask = darwin.RegisterSelector(
			"nextEventMatchingMask:untilDate:inMode:dequeue:")
		darwinSels.sendEvent = darwin.RegisterSelector("sendEvent:")

		// NSStatusBar
		darwinSels.systemStatusBar = darwin.RegisterSelector("systemStatusBar")
		darwinSels.statusItemWithLen = darwin.RegisterSelector("statusItemWithLength:")
		darwinSels.removeStatusItem = darwin.RegisterSelector("removeStatusItem:")

		// NSStatusItem
		darwinSels.button = darwin.RegisterSelector("button")
		darwinSels.setMenu = darwin.RegisterSelector("setMenu:")

		// NSStatusBarButton (NSButton)
		darwinSels.setImage = darwin.RegisterSelector("setImage:")
		darwinSels.setToolTip = darwin.RegisterSelector("setToolTip:")
		darwinSels.setTarget = darwin.RegisterSelector("setTarget:")
		darwinSels.setAction = darwin.RegisterSelector("setAction:")

		// NSImage
		darwinSels.initWithData = darwin.RegisterSelector("initWithData:")
		darwinSels.setSize = darwin.RegisterSelector("setSize:")
		darwinSels.setTemplate = darwin.RegisterSelector("setTemplate:")

		// NSMenu
		darwinSels.initWithTitle = darwin.RegisterSelector("initWithTitle:")
		darwinSels.addItem = darwin.RegisterSelector("addItem:")
		darwinSels.separatorItem = darwin.RegisterSelector("separatorItem")
		darwinSels.setSubmenu = darwin.RegisterSelector("setSubmenu:")
		darwinSels.initWithTitleActionKeyEquiv = darwin.RegisterSelector(
			"initWithTitle:action:keyEquivalent:")
		darwinSels.setState = darwin.RegisterSelector("setState:")
		darwinSels.setEnabled = darwin.RegisterSelector("setEnabled:")
		darwinSels.setDelegate = darwin.RegisterSelector("setDelegate:")
		darwinSels.menuWillOpen = darwin.RegisterSelector("menuWillOpen:")
		darwinSels.menuDidClose = darwin.RegisterSelector("menuDidClose:")

		// NSMenuItem
		darwinSels.menu = darwin.RegisterSelector("menu")

		darwinSels.performSelectorOnMainThread = darwin.RegisterSelector(
			"performSelectorOnMainThread:withObject:waitUntilDone:")

		// NSDate
		darwinSels.distantPast = darwin.RegisterSelector("distantPast")
		darwinSels.distantFuture = darwin.RegisterSelector("distantFuture")

		// NSUserNotificationCenter
		darwinSels.defaultUserNotificationCenter = darwin.RegisterSelector(
			"defaultUserNotificationCenter")
		darwinSels.deliverNotification = darwin.RegisterSelector("deliverNotification:")

		// NSUserNotification
		darwinSels.setTitle = darwin.RegisterSelector("setTitle:")
		darwinSels.setInformativeText = darwin.RegisterSelector("setInformativeText:")
	})
}

func initDarwinClasses() {
	darwinClasses.once.Do(func() {
		darwinClasses.NSApplication = darwin.GetClass("NSApplication")
		darwinClasses.NSStatusBar = darwin.GetClass("NSStatusBar")
		darwinClasses.NSImage = darwin.GetClass("NSImage")
		darwinClasses.NSMenu = darwin.GetClass("NSMenu")
		darwinClasses.NSMenuItem = darwin.GetClass("NSMenuItem")
		darwinClasses.NSDate = darwin.GetClass("NSDate")
		darwinClasses.NSAutoreleasePool = darwin.GetClass("NSAutoreleasePool")
		darwinClasses.NSUserNotificationCenter = darwin.GetClass("NSUserNotificationCenter")
		darwinClasses.NSUserNotification = darwin.GetClass("NSUserNotification")
	})
}

// darwinTray implements PlatformTray using NSStatusBar/NSStatusItem.
type darwinTray struct {
	statusBar  darwin.ID // [NSStatusBar systemStatusBar]
	statusItem darwin.ID // NSStatusItem
	btn        darwin.ID // NSStatusBarButton (from [statusItem button])
	target     darwin.ID // GoSystrayTarget instance for click action routing

	callbacks *Callbacks
	iconData  []byte // stored PNG for recovery after Hide/Show

	// menuMu guards the menu bookkeeping below.
	//
	// builds holds the menu trees built by SetMenu, newest first. A build is
	// immutable once published and stays registered for as long as AppKit may
	// still deliver a click from it: builds[0] is the menu attached to the
	// status item, and builds[1] is the one it replaced, which is still needed
	// while the action of a click on it is being delivered. A click is resolved
	// against the build that owns the clicked NSMenuItem, never against "the
	// current menu", so a rebuild can never renumber the menu on screen.
	//
	// While a menu is on screen a rebuild is queued in pending and published by
	// menuDidClose:. That keeps the menu attached to the status item stable for
	// the whole interaction; dispatch does not depend on it.
	menuMu   sync.Mutex
	builds   []*menuBuild
	menuOpen bool       // an NSMenu is on screen right now
	pending  *menuBuild // newest SetMenu queued behind the open menu

	pendingUpdates chan menuItemSnapshot // buffered channel for main-thread dispatch
}

// menuBuild is one built NSMenu tree together with the lookup tables that
// describe it. A build with a nil menu is a SetMenu(nil): it carries no menu
// and no items, and publishing it detaches the menu from the status item.
type menuBuild struct {
	menu    darwin.ID            // root NSMenu
	actions map[uint32]func()    // item.ID() -> callback
	items   map[uint32]darwin.ID // item.ID() -> NSMenuItem handle (incl. submenus)
	menus   map[uintptr]struct{} // every NSMenu handle of this build, root and submenus
}

// menuBuildRetention is how many built menus stay registered for click
// resolution: the menu on screen and the one that was on screen when it was
// replaced, whose action AppKit may deliver after it closed.
const menuBuildRetention = 2

func newMenuBuild() *menuBuild {
	return &menuBuild{
		actions: make(map[uint32]func()),
		items:   make(map[uint32]darwin.ID),
		menus:   make(map[uintptr]struct{}),
	}
}

// goSystrayTargetClass is the custom ObjC class registered once for click handling.
var (
	goSystrayTargetClass     darwin.Class
	goSystrayTargetClassOnce sync.Once
	errGoSystrayTargetClass  error
)

// trayRegistry maps GoSystrayTarget ObjC instance pointer to the owning darwinTray.
// In ObjC callbacks, the `self` parameter identifies which target fired,
// allowing correct routing when multiple trays exist.
var (
	trayRegistryMu  sync.RWMutex
	trayRegistryMap = make(map[uintptr]*darwinTray)
	runningNSApp    darwin.ID
)

// NewPlatformTray creates a macOS system tray implementation.
func NewPlatformTray(callbacks *Callbacks) PlatformTray {
	return &darwinTray{
		callbacks:      callbacks,
		pendingUpdates: make(chan menuItemSnapshot, 64),
	}
}

// ensureNSApplicationLaunched initializes the shared NSApplication and
// finishes launching it before any NSStatusItem is created. Creating a
// status item before the app launch completes crashes with a
// "CGSConnectionByID" assertion (known macOS pattern: status items must not
// be created before applicationDidFinishLaunching). All steps are idempotent,
// safe to call from Create() and Run().
func ensureNSApplicationLaunched() {
	initDarwinSels()
	initDarwinClasses()

	// Get or create the shared NSApplication.
	nsApp := darwinClasses.NSApplication.Send(darwinSels.sharedApplication)
	if nsApp.IsNil() {
		return
	}

	// Set activation policy to accessory (no dock icon for tray-only apps).
	nsApp.SendInt(darwinSels.setActivationPolicy, nsApplicationActivationPolicyAccessory)

	// Finish launching is required before the event loop can process events
	// and before the window-server connection is available for UI objects.
	nsApp.Send(darwinSels.finishLaunching)
}

// Create initializes the NSStatusBar item and sets up click handling.
func (t *darwinTray) Create() error {
	// NSApplication must be launched before creating the status item;
	// otherwise AppKit aborts in CGSConnectionByID when the process has no
	// window-server connection yet.
	ensureNSApplicationLaunched()

	// Get the system status bar.
	t.statusBar = darwinClasses.NSStatusBar.Send(darwinSels.systemStatusBar)
	if t.statusBar.IsNil() {
		return errors.New("darwin: failed to get NSStatusBar")
	}

	// Create status item with variable length.
	// [statusBar statusItemWithLength:NSVariableStatusItemLength]
	t.statusItem = t.statusBar.SendDouble(darwinSels.statusItemWithLen, nsVariableStatusItemLength)
	if t.statusItem.IsNil() {
		return errors.New("darwin: failed to create NSStatusItem")
	}

	// Get the button associated with the status item.
	t.btn = t.statusItem.Send(darwinSels.button)
	if t.btn.IsNil() {
		return errors.New("darwin: NSStatusItem has no button")
	}

	// Register custom ObjC target class for click action routing.
	targetClass, err := registerGoSystrayTarget()
	if err != nil {
		return fmt.Errorf("darwin: register target class: %w", err)
	}

	// Create an instance of the target and set it on the button.
	t.target = darwin.ID(targetClass).Send(darwinSels.alloc)
	t.target = t.target.Send(darwinSels.init)
	if t.target.IsNil() {
		return errors.New("darwin: failed to create GoSystrayTarget")
	}

	// Register in tray registry so ObjC callbacks can route to this instance.
	trayRegistryMu.Lock()
	trayRegistryMap[t.target.Ptr()] = t
	trayRegistryMu.Unlock()

	// Set the button's target to our GoSystrayTarget instance and its action
	// to the trayClicked: selector. When the user clicks the status item
	// button, the ObjC runtime sends trayClicked: to our target.
	t.btn.SendPtr(darwinSels.setTarget, t.target.Ptr())
	trayClickedSel := darwin.RegisterSelector("trayClicked:")
	t.btn.SendPtr(darwinSels.setAction, uintptr(trayClickedSel))

	return nil
}

// registerGoSystrayTarget creates a custom ObjC class "GoSystrayTarget" that
// handles button click actions. The class is created once and reused.
func registerGoSystrayTarget() (darwin.Class, error) {
	goSystrayTargetClassOnce.Do(func() {
		nsObjectClass := darwin.GetClass("NSObject")
		if nsObjectClass == 0 {
			errGoSystrayTargetClass = darwin.ErrClassNotFound
			return
		}

		cls := darwin.AllocateClassPair(nsObjectClass, "GoSystrayTarget")
		if cls == 0 {
			errGoSystrayTargetClass = errors.New("darwin: failed to allocate GoSystrayTarget class")
			return
		}

		// Add trayClicked: method — called when the status bar button is clicked.
		// ObjC signature: -(void)trayClicked:(id)sender → "v@:@"
		trayClickedIMP := ffi.NewCallback(func(self, sel, sender uintptr) uintptr {
			trayRegistryMu.RLock()
			t := trayRegistryMap[self]
			trayRegistryMu.RUnlock()
			if t != nil && t.callbacks != nil {
				if fn := t.callbacks.OnClick; fn != nil {
					fn()
				}
			}
			return 0
		})
		darwin.ClassAddMethod(cls, darwin.RegisterSelector("trayClicked:"), trayClickedIMP, "v@:@")

		// Add menuItemClicked: method — called when a menu item is clicked.
		// The sender's tag is the item's stable MenuItem.ID() and its NSMenu
		// identifies the build it was created in, so the click is resolved
		// against the menu the item actually belongs to. AppKit sends
		// menuDidClose: before the item's action, so resolving against "the
		// current menu" would drop clicks that a rebuild replaced in between.
		// ObjC signature: -(void)menuItemClicked:(NSMenuItem*)sender → "v@:@"
		menuClickedIMP := ffi.NewCallback(func(self, sel, sender uintptr) uintptr {
			trayRegistryMu.RLock()
			t := trayRegistryMap[self]
			trayRegistryMu.RUnlock()
			if t == nil {
				return 0
			}
			tagSel := darwin.RegisterSelector("tag")
			tag := darwin.ID(sender).Send(tagSel)
			parent := darwin.ID(sender).Send(darwinSels.menu)

			if fn := t.callbackForMenu(parent.Ptr(), uint32(tag)); fn != nil {
				fn()
			}
			return 0
		})
		darwin.ClassAddMethod(cls, darwin.RegisterSelector("menuItemClicked:"), menuClickedIMP, "v@:@")

		// Add menuWillOpen:/menuDidClose: — NSMenuDelegate hooks that mark when a
		// menu is on screen. SetMenu queues a rebuild while a menu is open and
		// publishes it here, so the menu the user is interacting with is not
		// swapped underneath the pointer.
		// ObjC signature: -(void)menuWillOpen:(NSMenu*)menu → "v@:@"
		menuWillOpenIMP := ffi.NewCallback(func(self, sel, sender uintptr) uintptr {
			trayRegistryMu.RLock()
			t := trayRegistryMap[self]
			trayRegistryMu.RUnlock()
			if t != nil {
				t.menuWillOpen()
			}
			return 0
		})
		darwin.ClassAddMethod(cls, darwin.RegisterSelector("menuWillOpen:"), menuWillOpenIMP, "v@:@")

		// ObjC signature: -(void)menuDidClose:(NSMenu*)menu → "v@:@"
		menuDidCloseIMP := ffi.NewCallback(func(self, sel, sender uintptr) uintptr {
			trayRegistryMu.RLock()
			t := trayRegistryMap[self]
			trayRegistryMu.RUnlock()
			if t != nil {
				t.menuDidClose()
			}
			return 0
		})
		darwin.ClassAddMethod(cls, darwin.RegisterSelector("menuDidClose:"), menuDidCloseIMP, "v@:@")

		// Add drainUpdates: method — called on main thread via performSelectorOnMainThread.
		// Drains the pendingUpdates channel and applies AppKit changes safely.
		drainUpdatesIMP := ffi.NewCallback(func(self, sel, sender uintptr) uintptr {
			trayRegistryMu.RLock()
			t := trayRegistryMap[self]
			trayRegistryMu.RUnlock()
			if t != nil {
				t.applyPendingUpdates()
			}
			return 0
		})
		darwin.ClassAddMethod(cls, darwin.RegisterSelector("drainUpdates:"), drainUpdatesIMP, "v@:@")

		darwin.RegisterClassPair(cls)
		goSystrayTargetClass = cls
	})

	return goSystrayTargetClass, errGoSystrayTargetClass
}

// SetIcon sets the tray icon from PNG bytes.
// The image is resized to 22x22 points, the standard macOS menu bar icon size.
func (t *darwinTray) SetIcon(png []byte) error {
	if t.statusItem.IsNil() || t.btn.IsNil() {
		return errors.New("darwin: tray not created")
	}

	t.iconData = png

	nsImage := createNSImage(png, false)
	if nsImage.IsNil() {
		return errors.New("darwin: failed to create NSImage from PNG data")
	}

	// [button setImage:nsImage]
	t.btn.SendPtr(darwinSels.setImage, nsImage.Ptr())

	return nil
}

// SetTemplateIcon sets a macOS template image. Template images are monochrome
// and the system automatically adjusts their appearance for the current menu
// bar style (light/dark).
func (t *darwinTray) SetTemplateIcon(png []byte) error {
	if t.statusItem.IsNil() || t.btn.IsNil() {
		return errors.New("darwin: tray not created")
	}

	t.iconData = png

	nsImage := createNSImage(png, true)
	if nsImage.IsNil() {
		return errors.New("darwin: failed to create template NSImage")
	}

	// [button setImage:nsImage]
	t.btn.SendPtr(darwinSels.setImage, nsImage.Ptr())

	return nil
}

// createNSImage creates an NSImage from PNG data, optionally marking it as
// a template image. The image is resized to 22x22 points.
func createNSImage(png []byte, template bool) darwin.ID {
	initDarwinSels()
	initDarwinClasses()

	// Create NSData from the raw PNG bytes.
	nsData := darwin.NewNSData(png)
	if nsData.IsNil() {
		return 0
	}

	// [[NSImage alloc] initWithData:nsData]
	nsImage := darwinClasses.NSImage.Send(darwinSels.alloc)
	if nsImage.IsNil() {
		return 0
	}
	nsImage = nsImage.SendPtr(darwinSels.initWithData, nsData.Ptr())
	if nsImage.IsNil() {
		return 0
	}

	// [nsImage setSize:NSMakeSize(22, 22)] — standard menu bar icon size
	nsImage.SendSize(darwinSels.setSize, darwin.NSSize{Width: 22, Height: 22})

	// [nsImage setTemplate:YES] if requested
	if template {
		nsImage.SendBool(darwinSels.setTemplate, true)
	}

	return nsImage
}

// SetTooltip sets the hover tooltip text.
func (t *darwinTray) SetTooltip(text string) error {
	if t.btn.IsNil() {
		return errors.New("darwin: tray not created")
	}

	nsStr := darwin.NewNSString(text)
	if nsStr.IsNil() {
		return errors.New("darwin: failed to create NSString for tooltip")
	}

	// [button setToolTip:nsString]
	t.btn.SendPtr(darwinSels.setToolTip, nsStr.Ptr())

	return nil
}

// SetMenu builds an NSMenu from our Menu struct and attaches it to the status
// item. A nil (or empty) menu removes the menu, so a click on the status item
// fires trayClicked: instead.
//
// If a menu is currently on screen the new tree is built and queued instead of
// attached, and menuDidClose: publishes it: the menu the user is interacting
// with keeps its own NSMenu for the whole interaction. Click dispatch does not
// depend on that deferral, because a click is resolved through the NSMenuItem
// AppKit sends (see menuItemClicked:).
func (t *darwinTray) SetMenu(menu *Menu) error {
	if t.statusItem.IsNil() {
		return errors.New("darwin: tray not created")
	}

	// Build the tree before publishing it, so an in-flight click can never
	// observe a half-built build.
	build, err := t.buildMenuTree(menu)
	if err != nil {
		return err
	}

	t.menuMu.Lock()
	if t.menuOpen {
		// Publishing now would replace the menu the user is interacting with.
		// Keep it queued; the newest rebuild wins.
		t.pending = build
		t.menuMu.Unlock()
		return nil
	}
	t.adoptBuildLocked(build)
	t.menuMu.Unlock()

	t.attachMenu(build)

	return nil
}

// buildMenuTree builds the native menu tree for menu. A nil (or empty) menu
// builds a tree that carries no NSMenu: publishing it detaches the menu.
func (t *darwinTray) buildMenuTree(menu *Menu) (*menuBuild, error) {
	build := newMenuBuild()
	if menu == nil || len(menu.Items) == 0 {
		return build, nil
	}

	nsMenu := t.buildNSMenu("", menu, build)
	if nsMenu.IsNil() {
		return nil, errors.New("darwin: failed to build NSMenu")
	}
	build.menu = nsMenu

	// menuWillOpen:/menuDidClose: are delivered to the target.
	nsMenu.SendPtr(darwinSels.setDelegate, t.target.Ptr())

	return build, nil
}

// adoptBuildLocked makes build the newest published menu, keeping just enough
// history for clicks that AppKit has not delivered yet. Caller must hold
// menuMu.
func (t *darwinTray) adoptBuildLocked(build *menuBuild) {
	t.builds = append(t.builds, nil)
	copy(t.builds[1:], t.builds[:len(t.builds)-1])
	t.builds[0] = build
	if len(t.builds) > menuBuildRetention {
		t.builds = t.builds[:menuBuildRetention]
	}
}

// currentBuildLocked returns the newest published build, or nil when no menu
// was ever set. Caller must hold menuMu.
func (t *darwinTray) currentBuildLocked() *menuBuild {
	if len(t.builds) == 0 {
		return nil
	}
	return t.builds[0]
}

// currentMenu returns the NSMenu attached to the status item, or 0 when the
// current build carries no menu.
func (t *darwinTray) currentMenu() darwin.ID {
	t.menuMu.Lock()
	defer t.menuMu.Unlock()

	build := t.currentBuildLocked()
	if build == nil {
		return 0
	}
	return build.menu
}

// attachMenu makes build the menu the status item shows. Caller must not hold
// menuMu: this calls into AppKit.
func (t *darwinTray) attachMenu(build *menuBuild) {
	if t.statusItem.IsNil() || build == nil {
		return
	}
	if build.menu.IsNil() {
		// [statusItem setMenu:nil] — no menu, clicks fall through to the button.
		t.statusItem.SendPtr(darwinSels.setMenu, 0)
		return
	}
	// [statusItem setMenu:nsMenu]
	t.statusItem.SendPtr(darwinSels.setMenu, build.menu.Ptr())
}

// callbackForMenu returns the callback tagged on menuH for tag, resolved in the
// build that owns the NSMenu. It returns nil when no retained build owns the
// menu or the tag.
func (t *darwinTray) callbackForMenu(menuH uintptr, tag uint32) func() {
	t.menuMu.Lock()
	defer t.menuMu.Unlock()

	for _, build := range t.builds {
		if _, ok := build.menus[menuH]; !ok {
			continue
		}
		return build.actions[tag]
	}

	return nil
}

// menuWillOpen records that a menu is on screen, so a rebuild arriving from
// another goroutine is queued instead of replacing the menu under the pointer.
//
// A pending build at this point means menuDidClose: was not delivered for the
// previous menu. It is applied now, otherwise a missed callback would freeze
// the menu for the rest of the process. Clicks keep working either way, because
// they resolve against the build that owns the clicked item.
func (t *darwinTray) menuWillOpen() {
	t.menuMu.Lock()
	t.menuOpen = true
	stuck := t.pending
	t.pending = nil
	if stuck != nil {
		t.adoptBuildLocked(stuck)
	}
	t.menuMu.Unlock()

	if stuck != nil {
		t.attachMenu(stuck)
	}
}

// menuDidClose publishes a rebuild that was queued while the menu was open.
//
// AppKit sends menuDidClose: before the clicked item's action, so a build
// published here is newer than the menu the click came from. That is harmless:
// menuItemClicked: resolves the click in the build that owns the NSMenuItem,
// and the replaced build stays registered for exactly that action.
func (t *darwinTray) menuDidClose() {
	t.menuMu.Lock()
	t.menuOpen = false
	next := t.pending
	t.pending = nil
	if next != nil {
		t.adoptBuildLocked(next)
	}
	t.menuMu.Unlock()

	if next != nil {
		t.attachMenu(next)
	}
}

// buildNSMenu recursively converts a Menu into an NSMenu, filling in the build
// that describes it.
// Each actionable item is tagged with its stable MenuItem.ID() for callback
// routing, so tags are never reused for a different item across rebuilds. Every
// NSMenu handle of the tree is recorded in the build, which is what lets
// menuItemClicked: resolve the click against the menu the item belongs to.
func (t *darwinTray) buildNSMenu(title string, menu *Menu, build *menuBuild) darwin.ID {
	initDarwinSels()
	initDarwinClasses()

	// Create NSMenu.
	nsMenu := darwinClasses.NSMenu.Send(darwinSels.alloc)
	if nsMenu.IsNil() {
		return 0
	}

	if title != "" {
		nsTitle := darwin.NewNSString(title)
		nsMenu = nsMenu.SendPtr(darwinSels.initWithTitle, nsTitle.Ptr())
	} else {
		nsMenu = nsMenu.Send(darwinSels.init)
	}
	if nsMenu.IsNil() {
		return 0
	}

	build.menus[nsMenu.Ptr()] = struct{}{}

	menuClickedSel := darwin.RegisterSelector("menuItemClicked:")

	for _, item := range menu.Items {
		snapshot := item.snapshot()
		switch snapshot.itemType {
		case MenuItemSeparator:
			sep := darwinClasses.NSMenuItem.Send(darwinSels.separatorItem)
			if !sep.IsNil() {
				nsMenu.SendPtr(darwinSels.addItem, sep.Ptr())
			}

		case MenuItemSubmenu:
			// Create a placeholder NSMenuItem for the submenu.
			nsLabel := darwin.NewNSString(snapshot.label)
			emptyKey := darwin.NewNSString("")
			nsItem := darwinClasses.NSMenuItem.Send(darwinSels.alloc)
			nsItem = darwin.MsgSend3Ptr(nsItem, darwinSels.initWithTitleActionKeyEquiv,
				nsLabel.Ptr(), 0, emptyKey.Ptr())
			if nsItem.IsNil() {
				continue
			}

			// Build the submenu recursively.
			subMenu := t.buildNSMenu(snapshot.label, item.Submenu, build)
			if !subMenu.IsNil() {
				nsItem.SendPtr(darwinSels.setSubmenu, subMenu.Ptr())
			}

			nsMenu.SendPtr(darwinSels.addItem, nsItem.Ptr())

			// Store the submenu container handle for dynamic updates.
			// Submenu containers are NSMenuItems like any other — users can
			// call SetLabel/SetDisabled on the *MenuItem returned by
			// AddSubmenu, so it must be resolvable via build.items.
			build.items[snapshot.id] = nsItem

		default:
			// Normal or checkbox item.
			nsLabel := darwin.NewNSString(snapshot.label)
			emptyKey := darwin.NewNSString("")
			nsItem := darwinClasses.NSMenuItem.Send(darwinSels.alloc)

			// Set action to menuItemClicked: on our target.
			nsItem = darwin.MsgSend3Ptr(nsItem, darwinSels.initWithTitleActionKeyEquiv,
				nsLabel.Ptr(), uintptr(menuClickedSel), emptyKey.Ptr())
			if nsItem.IsNil() {
				continue
			}

			// Set the target so Cocoa sends the action to our GoSystrayTarget.
			nsItem.SendPtr(darwinSels.setTarget, t.target.Ptr())

			// Set tag for callback routing to the item's stable ID.
			// [nsItem setTag:snapshot.id]
			setTagSel := darwin.RegisterSelector("setTag:")
			nsItem.SendInt(setTagSel, int64(snapshot.id))

			// Set checked state for checkbox items.
			// NSControlStateValueOn = 1, NSControlStateValueOff = 0
			if snapshot.itemType == MenuItemCheckbox && snapshot.checked {
				nsItem.SendInt(darwinSels.setState, 1)
			}

			// Set icon if provided.
			if len(snapshot.icon) > 0 {
				nsImage := createNSImage(snapshot.icon, false)
				if !nsImage.IsNil() {
					nsItem.SendPtr(darwinSels.setImage, nsImage.Ptr())
				}
			}

			// Register Go callback and map item ID to NSMenuItem handle for
			// UpdateItem lookup. itemWithTag: only searches the root menu, so
			// items inside submenus must be resolved via this map.
			if item.OnClick != nil {
				build.actions[snapshot.id] = item.OnClick
			}
			build.items[snapshot.id] = nsItem

			nsMenu.SendPtr(darwinSels.addItem, nsItem.Ptr())
		}
	}

	return nsMenu
}

// UpdateItem dispatches a menu item update to the main thread.
// AppKit requires all UI mutations on the main thread. We enqueue the snapshot
// and call performSelectorOnMainThread to drain the queue safely.
func (t *darwinTray) UpdateItem(item *MenuItem) error {
	return t.updateItem(item.snapshot())
}

func (t *darwinTray) updateItem(item menuItemSnapshot) error {
	if t.target.IsNil() {
		return nil
	}

	// The item can belong to the menu on screen and to a menu queued behind an
	// open popup at the same time. Both are updated: the queued menu is not on
	// screen yet, and applying the change to it now keeps it from being
	// published with the state it was built from.
	if !t.tracksItem(item.id) {
		return nil
	}

	// Queue the captured state for main-thread processing.
	t.pendingUpdates <- item

	// Dispatch to main thread: [target performSelectorOnMainThread:@selector(drainUpdates:) withObject:nil waitUntilDone:YES]
	drainSel := darwin.RegisterSelector("drainUpdates:")
	darwin.MsgSend3Ptr(t.target, darwinSels.performSelectorOnMainThread,
		uintptr(drainSel), 0, 1) // waitUntilDone:YES

	return nil
}

// tracksItem reports whether id belongs to the menu on screen or to a menu
// queued behind it. Items of older builds are gone from the UI and are left
// untouched.
func (t *darwinTray) tracksItem(id uint32) bool {
	t.menuMu.Lock()
	defer t.menuMu.Unlock()

	for _, build := range []*menuBuild{t.currentBuildLocked(), t.pending} {
		if build == nil {
			continue
		}
		if _, ok := build.items[id]; ok {
			return true
		}
	}

	return false
}

// itemHandles returns the NSMenuItem handles of the item in the menu on screen
// and in the menu queued behind it. Both are updated in place, so publishing a
// queued menu cannot undo an update that arrived while it was queued.
func (t *darwinTray) itemHandles(id uint32) []darwin.ID {
	t.menuMu.Lock()
	defer t.menuMu.Unlock()

	handles := make([]darwin.ID, 0, 2)
	for _, build := range []*menuBuild{t.currentBuildLocked(), t.pending} {
		if build == nil {
			continue
		}
		if nsItem, ok := build.items[id]; ok && !nsItem.IsNil() {
			handles = append(handles, nsItem)
		}
	}

	return handles
}

// applyPendingUpdates drains the pendingUpdates channel and applies AppKit
// changes. MUST be called on the main thread (via drainUpdates: ObjC callback).
// A snapshot with a zero ID signals Destroy.
func (t *darwinTray) applyPendingUpdates() {
	for {
		select {
		case item := <-t.pendingUpdates:
			if item.id == 0 {
				t.destroyOnMainThread()
				return
			}
			t.applyItemUpdate(item)
		default:
			return
		}
	}
}

// applyItemUpdate applies a single snapshot to the item's NSMenuItem in the
// menu on screen and, if a rebuild is queued behind it, to that one as well.
// MUST be called on the main thread.
func (t *darwinTray) applyItemUpdate(item menuItemSnapshot) {
	handles := t.itemHandles(item.id)
	if len(handles) == 0 {
		return
	}

	nsTitle := darwin.NewNSString(item.label)
	var nsImage darwin.ID
	if len(item.icon) > 0 {
		nsImage = createNSImage(item.icon, false)
	}

	for _, nsItem := range handles {
		if !nsTitle.IsNil() {
			nsItem.SendPtr(darwinSels.setTitle, nsTitle.Ptr())
		}

		if item.itemType == MenuItemCheckbox {
			state := int64(0)
			if item.checked {
				state = 1
			}
			nsItem.SendInt(darwinSels.setState, state)
		}

		nsItem.SendBool(darwinSels.setEnabled, !item.disabled)

		if !nsImage.IsNil() {
			nsItem.SendPtr(darwinSels.setImage, nsImage.Ptr())
		}
	}
}

// ShowNotification displays an OS-level notification using NSUserNotification.
// NSUserNotification was deprecated in macOS 10.14 in favor of UNUserNotification,
// but remains functional through at least macOS 14. A future version may migrate
// to UNUserNotificationCenter.
func (t *darwinTray) ShowNotification(title, message string) error {
	initDarwinSels()
	initDarwinClasses()

	// Create NSUserNotification.
	notification := darwinClasses.NSUserNotification.Send(darwinSels.alloc)
	notification = notification.Send(darwinSels.init)
	if notification.IsNil() {
		return errors.New("darwin: failed to create NSUserNotification")
	}

	// Set title.
	nsTitle := darwin.NewNSString(title)
	if !nsTitle.IsNil() {
		notification.SendPtr(darwinSels.setTitle, nsTitle.Ptr())
	}

	// Set informative text (body).
	nsMessage := darwin.NewNSString(message)
	if !nsMessage.IsNil() {
		notification.SendPtr(darwinSels.setInformativeText, nsMessage.Ptr())
	}

	// Deliver via the default notification center.
	center := darwinClasses.NSUserNotificationCenter.Send(darwinSels.defaultUserNotificationCenter)
	if center.IsNil() {
		return errors.New("darwin: failed to get NSUserNotificationCenter")
	}

	center.SendPtr(darwinSels.deliverNotification, notification.Ptr())

	return nil
}

// Show makes the tray icon visible. The status item is visible immediately
// after Create(), so this is effectively a no-op unless Hide() was called.
func (t *darwinTray) Show() error {
	if !t.statusItem.IsNil() {
		// Already visible.
		return nil
	}

	// Re-create the status item if it was removed by Hide().
	if t.statusBar.IsNil() {
		return errors.New("darwin: tray not created")
	}

	t.statusItem = t.statusBar.SendDouble(darwinSels.statusItemWithLen, nsVariableStatusItemLength)
	if t.statusItem.IsNil() {
		return errors.New("darwin: failed to re-create NSStatusItem")
	}

	t.btn = t.statusItem.Send(darwinSels.button)

	// Restore icon if we had one.
	if len(t.iconData) > 0 {
		if err := t.SetIcon(t.iconData); err != nil {
			slog.Warn("darwin: failed to restore icon after Show", "err", err)
		}
	}

	// Restore menu if we had one.
	if nsMenu := t.currentMenu(); !nsMenu.IsNil() {
		t.statusItem.SendPtr(darwinSels.setMenu, nsMenu.Ptr())
	}

	// Restore target/action for click handling.
	if !t.target.IsNil() && !t.btn.IsNil() {
		t.btn.SendPtr(darwinSels.setTarget, t.target.Ptr())
		trayClickedSel := darwin.RegisterSelector("trayClicked:")
		t.btn.SendPtr(darwinSels.setAction, uintptr(trayClickedSel))
	}

	return nil
}

// Hide removes the status item from the menu bar without destroying the tray.
// Call Show() to make it visible again.
func (t *darwinTray) Hide() error {
	if t.statusBar.IsNil() || t.statusItem.IsNil() {
		return nil
	}

	// [statusBar removeStatusItem:statusItem]
	t.statusBar.SendPtr(darwinSels.removeStatusItem, t.statusItem.Ptr())
	t.statusItem = 0
	t.btn = 0

	return nil
}

// Bounds returns the tray icon's screen position.
// On macOS, NSStatusItem does not provide a direct API for this.
// Returns zeros; callers should not depend on this for positioning.
func (t *darwinTray) Bounds() (int, int, int, int) {
	// NSStatusItem window frame could be queried via [[[statusItem button] window] frame],
	// but this requires NSRect return handling. For v1, return zeros.
	return 0, 0, 0, 0
}

// Run blocks the calling goroutine, running the Cocoa event loop ([NSApp run]).
// It returns when the loop is stopped after the last tray's Destroy() has run.
// Only call Run() once per process: the shared NSApplication event loop serves
// all tray icons.
func (t *darwinTray) Run() error {
	initDarwinSels()
	initDarwinClasses()

	// Get or create the shared NSApplication.
	nsApp := darwinClasses.NSApplication.Send(darwinSels.sharedApplication)
	if nsApp.IsNil() {
		return errors.New("darwin: failed to get NSApplication")
	}
	trayRegistryMu.Lock()
	runningNSApp = nsApp
	trayRegistryMu.Unlock()

	// Set activation policy to accessory (no dock icon for tray-only apps).
	nsApp.SendInt(darwinSels.setActivationPolicy, nsApplicationActivationPolicyAccessory)

	// Finish launching is required before the event loop can process events.
	nsApp.Send(darwinSels.finishLaunching)

	// Run the Cocoa event loop. This blocks until [NSApp stop:] is sent and
	// the wake event posted by Destroy() is processed.
	// [NSApp run]
	nsApp.Send(darwinSels.run)

	trayRegistryMu.Lock()
	if runningNSApp == nsApp {
		runningNSApp = 0
	}
	trayRegistryMu.Unlock()

	return nil
}

// Destroy releases all resources associated with the tray icon.
// Safe to call from any goroutine, and never blocks the caller. The AppKit
// cleanup and the event-loop stop are executed on the main thread
// (drainUpdates:), so the ordering is deterministic:
//
//	final cleanup → [NSApp stop:] → wake event → [NSApp run] returns → Run() returns
//
// The drain is dispatched with waitUntilDone:NO because Destroy must never
// block: if the shared event loop has already exited (e.g. another tray was
// removed first), the performSelector would never be serviced and a blocking
// wait would hang the caller. While the loop is running, the drain is
// serviced before the wake event it posts, so cleanup always completes before
// Run() returns.
func (t *darwinTray) Destroy() {
	if t.target.IsNil() {
		return
	}

	// Queue cleanup for main thread execution.
	t.pendingUpdates <- menuItemSnapshot{} // zero ID sentinel signals destroy

	// Dispatch cleanup to main thread. waitUntilDone:NO.
	drainSel := darwin.RegisterSelector("drainUpdates:")
	darwin.MsgSend3Ptr(t.target, darwinSels.performSelectorOnMainThread,
		uintptr(drainSel), 0, 0) // waitUntilDone:NO
}

// destroyOnMainThread performs the actual AppKit cleanup.
// MUST be called on the main thread.
func (t *darwinTray) destroyOnMainThread() {
	// Remove the status item from the menu bar.
	if !t.statusBar.IsNil() && !t.statusItem.IsNil() {
		t.statusBar.SendPtr(darwinSels.removeStatusItem, t.statusItem.Ptr())
	}

	// Unregister from tray registry before releasing the target. Only the final
	// tray owns shutdown of the shared application event loop.
	lastTray := false
	var nsApp darwin.ID
	if !t.target.IsNil() {
		trayRegistryMu.Lock()
		delete(trayRegistryMap, t.target.Ptr())
		lastTray = shouldStopDarwinApplication(len(trayRegistryMap))
		nsApp = runningNSApp
		trayRegistryMu.Unlock()
	}

	// Detach the menu delegate before the target is released: NSMenu.delegate is
	// an unretained reference, and a menu that outlives the target would call
	// back into a freed object.
	t.menuMu.Lock()
	builds := t.builds
	t.builds = nil
	t.pending = nil
	t.menuOpen = false
	t.menuMu.Unlock()
	for _, build := range builds {
		if build.menu.IsNil() {
			continue
		}
		build.menu.SendPtr(darwinSels.setDelegate, 0)
	}

	// Release ObjC objects.
	if !t.target.IsNil() {
		t.target.Send(darwinSels.release)
		t.target = 0
	}

	t.statusItem = 0
	t.btn = 0

	// Stop the shared application and wake its event loop with a real event.
	// [NSApp run] only re-checks the stop flag after processing an event; a
	// bare CFRunLoopStop wake is insufficient (verified on macOS 14-26).
	// stop: is thread-safe, and postEvent:atStart:YES delivers the wake event
	// at the head of the queue (gogpu reference pattern).
	if lastTray && !nsApp.IsNil() {
		nsApp.SendPtr(darwinSels.stop, 0)
		darwin.PostAppDefinedEvent(nsApp)
	}
}

func shouldStopDarwinApplication(remainingTrays int) bool {
	return remainingTrays == 0
}
