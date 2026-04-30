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

// menuItemCallbackID is the base for menu item command IDs.
// Each menu item gets baseID + index to route action callbacks.
const menuItemCallbackBaseID = 1000

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
	nsMenu     darwin.ID // NSMenu attached to the status item
	target     darwin.ID // GoSystrayTarget instance for click action routing
	nsApp      darwin.ID // NSApplication shared instance

	callbacks *Callbacks
	iconData  []byte // stored PNG for recovery after Hide/Show

	// menuActions maps menu item indices to their callbacks.
	// Populated when SetMenu builds the NSMenu hierarchy.
	menuActions map[int]func()
	menuMu      sync.Mutex
}

// goSystrayTargetClass is the custom ObjC class registered once for click handling.
var (
	goSystrayTargetClass     darwin.Class
	goSystrayTargetClassOnce sync.Once
	errGoSystrayTargetClass  error
)

// activeTray is the tray instance receiving callbacks from the ObjC target.
// Only one systray instance per process on macOS (there is one status bar).
var activeTray *darwinTray

// NewPlatformTray creates a macOS system tray implementation.
func NewPlatformTray(callbacks *Callbacks) PlatformTray {
	return &darwinTray{
		callbacks:   callbacks,
		menuActions: make(map[int]func()),
	}
}

// Create initializes the NSStatusBar item and sets up click handling.
func (t *darwinTray) Create() error {
	initDarwinSels()
	initDarwinClasses()

	// Store as active tray for ObjC callbacks.
	activeTray = t

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
			if activeTray != nil && activeTray.callbacks != nil {
				if fn := activeTray.callbacks.OnClick; fn != nil {
					fn()
				}
			}
			return 0
		})
		darwin.ClassAddMethod(cls, darwin.RegisterSelector("trayClicked:"), trayClickedIMP, "v@:@")

		// Add menuItemClicked: method — called when a menu item is clicked.
		// We use the sender's tag to look up the Go callback.
		// ObjC signature: -(void)menuItemClicked:(NSMenuItem*)sender → "v@:@"
		menuClickedIMP := ffi.NewCallback(func(self, sel, sender uintptr) uintptr {
			if activeTray == nil {
				return 0
			}
			// Get the tag from the sender: [sender tag]
			tagSel := darwin.RegisterSelector("tag")
			tag := darwin.ID(sender).Send(tagSel)
			idx := int(tag)

			activeTray.menuMu.Lock()
			fn := activeTray.menuActions[idx]
			activeTray.menuMu.Unlock()

			if fn != nil {
				fn()
			}
			return 0
		})
		darwin.ClassAddMethod(cls, darwin.RegisterSelector("menuItemClicked:"), menuClickedIMP, "v@:@")

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

// SetMenu builds an NSMenu from our Menu struct and attaches it to the status item.
func (t *darwinTray) SetMenu(menu *Menu) error {
	if t.statusItem.IsNil() {
		return errors.New("darwin: tray not created")
	}

	if menu == nil {
		// Remove the menu. When no menu is set, the button action (trayClicked:)
		// fires on click.
		t.statusItem.SendPtr(darwinSels.setMenu, 0)
		t.nsMenu = 0
		return nil
	}

	// Build the NSMenu hierarchy.
	t.menuMu.Lock()
	// Clear old actions.
	t.menuActions = make(map[int]func())
	t.menuMu.Unlock()

	counter := menuItemCallbackBaseID
	nsMenu := t.buildNSMenu("", menu, &counter)
	if nsMenu.IsNil() {
		return errors.New("darwin: failed to build NSMenu")
	}

	t.nsMenu = nsMenu

	// [statusItem setMenu:nsMenu]
	t.statusItem.SendPtr(darwinSels.setMenu, nsMenu.Ptr())

	return nil
}

// buildNSMenu recursively converts a Menu into an NSMenu.
// counter is incremented per item and used as the tag for callback routing.
func (t *darwinTray) buildNSMenu(title string, menu *Menu, counter *int) darwin.ID {
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

	menuClickedSel := darwin.RegisterSelector("menuItemClicked:")

	for _, item := range menu.Items {
		switch item.Type {
		case MenuItemSeparator:
			sep := darwinClasses.NSMenuItem.Send(darwinSels.separatorItem)
			if !sep.IsNil() {
				nsMenu.SendPtr(darwinSels.addItem, sep.Ptr())
			}

		case MenuItemSubmenu:
			// Create a placeholder NSMenuItem for the submenu.
			nsLabel := darwin.NewNSString(item.Label)
			emptyKey := darwin.NewNSString("")
			nsItem := darwinClasses.NSMenuItem.Send(darwinSels.alloc)
			nsItem = darwin.MsgSend3Ptr(nsItem, darwinSels.initWithTitleActionKeyEquiv,
				nsLabel.Ptr(), 0, emptyKey.Ptr())
			if nsItem.IsNil() {
				continue
			}

			// Build the submenu recursively.
			subMenu := t.buildNSMenu(item.Label, item.Submenu, counter)
			if !subMenu.IsNil() {
				nsItem.SendPtr(darwinSels.setSubmenu, subMenu.Ptr())
			}

			nsMenu.SendPtr(darwinSels.addItem, nsItem.Ptr())

		default:
			// Normal or checkbox item.
			idx := *counter
			*counter++

			nsLabel := darwin.NewNSString(item.Label)
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

			// Set tag for callback routing.
			// [nsItem setTag:idx]
			setTagSel := darwin.RegisterSelector("setTag:")
			nsItem.SendInt(setTagSel, int64(idx))

			// Set checked state for checkbox items.
			// NSControlStateValueOn = 1, NSControlStateValueOff = 0
			if item.Type == MenuItemCheckbox && item.Checked {
				nsItem.SendInt(darwinSels.setState, 1)
			}

			// Set icon if provided.
			if len(item.Icon) > 0 {
				nsImage := createNSImage(item.Icon, false)
				if !nsImage.IsNil() {
					nsItem.SendPtr(darwinSels.setImage, nsImage.Ptr())
				}
			}

			// Register Go callback.
			if item.OnClick != nil {
				t.menuMu.Lock()
				t.menuActions[idx] = item.OnClick
				t.menuMu.Unlock()
			}

			nsMenu.SendPtr(darwinSels.addItem, nsItem.Ptr())
		}
	}

	return nsMenu
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
	if !t.nsMenu.IsNil() {
		t.statusItem.SendPtr(darwinSels.setMenu, t.nsMenu.Ptr())
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

// Run blocks the calling goroutine, running the Cocoa event loop.
// If NSApplication is already initialized (e.g., systray is used inside a gogpu
// app), this runs a local event polling loop instead of calling [NSApp run].
func (t *darwinTray) Run() error {
	initDarwinSels()
	initDarwinClasses()

	// Get or create the shared NSApplication.
	t.nsApp = darwinClasses.NSApplication.Send(darwinSels.sharedApplication)
	if t.nsApp.IsNil() {
		return errors.New("darwin: failed to get NSApplication")
	}

	// Set activation policy to accessory (no dock icon for tray-only apps).
	t.nsApp.SendInt(darwinSels.setActivationPolicy, nsApplicationActivationPolicyAccessory)

	// Finish launching is required before the event loop can process events.
	t.nsApp.Send(darwinSels.finishLaunching)

	// Run the Cocoa event loop. This blocks until the app terminates.
	// [NSApp run]
	t.nsApp.Send(darwinSels.run)

	return nil
}

// Destroy releases all resources associated with the tray icon.
func (t *darwinTray) Destroy() {
	// Remove the status item from the menu bar.
	if !t.statusBar.IsNil() && !t.statusItem.IsNil() {
		t.statusBar.SendPtr(darwinSels.removeStatusItem, t.statusItem.Ptr())
	}

	// Release ObjC objects.
	if !t.target.IsNil() {
		t.target.Send(darwinSels.release)
		t.target = 0
	}

	t.statusItem = 0
	t.btn = 0
	t.nsMenu = 0

	// Stop the run loop if we started it.
	if !t.nsApp.IsNil() {
		t.nsApp.SendPtr(darwinSels.stop, 0)
	}

	activeTray = nil
}
