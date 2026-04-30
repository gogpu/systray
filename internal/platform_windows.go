//go:build windows

package internal

import (
	"fmt"
	"log/slog"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// Win32 Shell_NotifyIconW message constants.
const (
	nimAdd    = 0x00000000
	nimModify = 0x00000001
	nimDelete = 0x00000002

	nimSetVersion = 0x00000004
)

// NOTIFYICONDATA.uFlags constants.
const (
	nifMessage  = 0x00000001
	nifIcon     = 0x00000002
	nifTip      = 0x00000004
	nifState    = 0x00000008
	nifInfo     = 0x00000010
	nifRealtime = 0x00000040
	nifShovel   = 0x00000080
)

// NOTIFYICONDATA.dwInfoFlags constants for balloon notification icons.
const (
	niifNone    = 0x00000000 // No icon
	niifInfo    = 0x00000001 // Info icon
	niifWarning = 0x00000002 // Warning icon
	niifError   = 0x00000003 // Error icon
	niifUser    = 0x00000004 // Use hBalloonIcon
)

// NOTIFYICONDATA.dwState / dwStateMask constants are reserved for future use.

// NOTIFYICON_VERSION_4 enables rich notification area behavior (Vista+):
// lParam carries the actual event in LOWORD, coordinates in HIWORD of wParam.
const notifyIconVersion4 = 4

// WM_USER base for tray callback messages.
// Each tray instance uses wmTrayCallback + its uid.
const wmTrayCallback = 0x0400 + 100 // WM_USER + 100

// Mouse messages used in tray wndProc (NOTIFYICON_VERSION_4 behavior).
const (
	wmLButtonUp     = 0x0202
	wmRButtonUp     = 0x0205
	wmLButtonDblClk = 0x0203
	wmContextMenu   = 0x007B
	wmNull          = 0x0000
	wmCommand       = 0x0111
	wmDestroy       = 0x0002
	wmSettingChange = 0x001A
)

// TrackPopupMenu flags.
const (
	tpmLeftAlign   = 0x0000
	tpmRightButton = 0x0002
	tpmReturnCmd   = 0x0100
	tpmNoNotify    = 0x0080
)

// Menu item flags for AppendMenuW.
const (
	mfString    = 0x00000000
	mfSeparator = 0x00000800
	mfPopup     = 0x00000010
	mfChecked   = 0x00000008
	mfGrayed    = 0x00000001
)

// HWND_MESSAGE parent for message-only windows.
const hwndMessage = ^uintptr(2) // (HWND)-3 = HWND_MESSAGE

// ChangeWindowMessageFilterEx constants (Windows 7+, UIPI).
const (
	msgfltAllow = 1
)

// CreateIconFromResourceEx flags.
const (
	lrDefaultColor = 0x00000000
	lrDefaultSize  = 0x00000040
)

// System tray icon dimensions.
const (
	smCXSmIcon = 49 // SM_CXSMICON
	smCYSmIcon = 50 // SM_CYSMICON
)

// Win32 DLL and procedure handles.
var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procShellNotifyIconW            = shell32.NewProc("Shell_NotifyIconW")
	procRegisterClassExW            = user32.NewProc("RegisterClassExW")
	procCreateWindowExW             = user32.NewProc("CreateWindowExW")
	procDestroyWindow               = user32.NewProc("DestroyWindow")
	procDefWindowProcW              = user32.NewProc("DefWindowProcW")
	procSetForegroundWindow         = user32.NewProc("SetForegroundWindow")
	procTrackPopupMenu              = user32.NewProc("TrackPopupMenu")
	procPostMessageW                = user32.NewProc("PostMessageW")
	procCreatePopupMenu             = user32.NewProc("CreatePopupMenu")
	procAppendMenuW                 = user32.NewProc("AppendMenuW")
	procDestroyMenu                 = user32.NewProc("DestroyMenu")
	procRegisterWindowMessageW      = user32.NewProc("RegisterWindowMessageW")
	procGetCursorPos                = user32.NewProc("GetCursorPos")
	procGetModuleHandleW            = kernel32.NewProc("GetModuleHandleW")
	procCreateIconFromResourceEx    = user32.NewProc("CreateIconFromResourceEx")
	procDestroyIcon                 = user32.NewProc("DestroyIcon")
	procGetSystemMetrics            = user32.NewProc("GetSystemMetrics")
	procChangeWindowMessageFilterEx = user32.NewProc("ChangeWindowMessageFilterEx")
	procGetMessageW                 = user32.NewProc("GetMessageW")
	procTranslateMessage            = user32.NewProc("TranslateMessage")
	procDispatchMessageW            = user32.NewProc("DispatchMessageW")
	procShellNotifyIconGetRect      = shell32.NewProc("Shell_NotifyIconGetRect")
)

// msg is the Win32 MSG structure for the message loop.
type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

// notifyIconData is the Win32 NOTIFYICONDATAW structure (Shell32 v6.0, Vista+).
// The struct layout must match the C definition exactly for Shell_NotifyIconW.
type notifyIconData struct {
	cbSize           uint32
	hWnd             uintptr
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	hIcon            uintptr
	szTip            [128]uint16
	dwState          uint32
	dwStateMask      uint32
	szInfo           [256]uint16
	uVersion         uint32 // union with uTimeout; NOTIFYICON_VERSION_4
	szInfoTitle      [64]uint16
	dwInfoFlags      uint32
	guidItem         [16]byte // GUID, unused
	hBalloonIcon     uintptr
}

// wndClassExW is the Win32 WNDCLASSEXW structure.
type wndClassExW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

// point is the Win32 POINT structure.
type point struct {
	x, y int32
}

// rect is the Win32 RECT structure.
type rect struct {
	left, top, right, bottom int32
}

// notifyIconIdentifier is the NOTIFYICONIDENTIFIER structure
// for Shell_NotifyIconGetRect (Windows 7+).
type notifyIconIdentifier struct {
	cbSize   uint32
	hWnd     uintptr
	uID      uint32
	guidItem [16]byte // GUID, unused (zero)
}

// trayRegistry maps HWND to tray instance for wndProc routing.
// Multiple tray icons each have their own message-only window.
var (
	trayMu       sync.RWMutex
	trayRegistry = make(map[uintptr]*win32Tray)
)

// classRegistered tracks whether the window class has been registered.
var (
	classOnce        sync.Once
	errClassRegister error
)

// taskbarCreatedMsg is the registered "TaskbarCreated" message ID.
// When explorer.exe crashes and restarts, it broadcasts this message
// so tray icons can re-add themselves.
var taskbarCreatedMsg uint32

// win32Tray implements PlatformTray using Shell_NotifyIconW.
type win32Tray struct {
	hwnd     uintptr // message-only window for tray callbacks
	uid      uint32  // icon ID passed to Shell_NotifyIconW
	hicon    uintptr // current HICON handle
	hmenu    uintptr // current HMENU handle for context menu
	visible  bool    // whether icon has been added to tray
	iconData []byte  // stored PNG for explorer crash recovery (light mode icon)
	iconDark []byte  // dark mode icon PNG for automatic theme switching
	tooltip  string  // stored tooltip for explorer crash recovery

	callbacks *Callbacks
	menu      *Menu // stored menu for rebuilding HMENU and dispatch
}

// NewPlatformTray creates a Win32 system tray implementation.
func NewPlatformTray(callbacks *Callbacks) PlatformTray {
	return &win32Tray{
		callbacks: callbacks,
	}
}

// Create initializes the Win32 tray: registers the window class (once),
// creates a message-only window, and registers the TaskbarCreated message.
func (t *win32Tray) Create() error {
	classOnce.Do(func() {
		errClassRegister = registerTrayWindowClass()
	})
	if errClassRegister != nil {
		return fmt.Errorf("register tray window class: %w", errClassRegister)
	}

	hwnd, err := createMessageWindow(t)
	if err != nil {
		return fmt.Errorf("create message window: %w", err)
	}
	t.hwnd = hwnd

	// Generate a unique icon ID from the atomic counter.
	t.uid = uint32(NewTrayID())

	// Allow the TaskbarCreated message through UIPI for elevated processes.
	if taskbarCreatedMsg != 0 {
		if err := procChangeWindowMessageFilterEx.Find(); err == nil {
			ret, _, _ := procChangeWindowMessageFilterEx.Call(
				t.hwnd,
				uintptr(taskbarCreatedMsg),
				uintptr(msgfltAllow),
				0,
			)
			if ret == 0 {
				slog.Warn("systray: ChangeWindowMessageFilterEx failed for TaskbarCreated")
			}
		}
	}

	return nil
}

// registerTrayWindowClass registers the window class used by all tray
// message-only windows. Called once via sync.Once.
func registerTrayWindowClass() error {
	// Register TaskbarCreated for explorer crash recovery.
	msgName, err := windows.UTF16PtrFromString("TaskbarCreated")
	if err != nil {
		return fmt.Errorf("utf16 TaskbarCreated: %w", err)
	}
	ret, _, _ := procRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(msgName)))
	if ret != 0 {
		taskbarCreatedMsg = uint32(ret)
	}

	className, err := windows.UTF16PtrFromString("GoGPUSystrayMsg")
	if err != nil {
		return fmt.Errorf("utf16 class name: %w", err)
	}

	hinstance, _, _ := procGetModuleHandleW.Call(0)

	wc := wndClassExW{
		cbSize:        uint32(unsafe.Sizeof(wndClassExW{})),
		lpfnWndProc:   syscall.NewCallback(trayWndProc),
		hInstance:     hinstance,
		lpszClassName: className,
	}

	ret, _, _ = procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if ret == 0 {
		return fmt.Errorf("RegisterClassExW failed for GoGPUSystrayMsg")
	}

	return nil
}

// createMessageWindow creates an HWND_MESSAGE window for tray callbacks
// and registers the tray instance in the global registry.
func createMessageWindow(t *win32Tray) (uintptr, error) {
	className, err := windows.UTF16PtrFromString("GoGPUSystrayMsg")
	if err != nil {
		return 0, fmt.Errorf("utf16 class name: %w", err)
	}

	hinstance, _, _ := procGetModuleHandleW.Call(0)

	hwnd, _, _ := procCreateWindowExW.Call(
		0,                                  // dwExStyle
		uintptr(unsafe.Pointer(className)), // lpClassName
		0,                                  // lpWindowName (none needed)
		0,                                  // dwStyle
		0, 0, 0, 0,                         // x, y, w, h
		hwndMessage, // hWndParent = HWND_MESSAGE
		0,           // hMenu
		hinstance,   // hInstance
		0,           // lpParam
	)
	if hwnd == 0 {
		return 0, fmt.Errorf("CreateWindowExW HWND_MESSAGE failed")
	}

	trayMu.Lock()
	trayRegistry[hwnd] = t
	trayMu.Unlock()

	return hwnd, nil
}

// SetIcon converts PNG bytes to HICON and updates the tray icon.
// This sets the light mode (default) icon. If a dark mode icon is also set,
// the tray automatically switches between them based on the system theme.
func (t *win32Tray) SetIcon(png []byte) error {
	if len(png) == 0 {
		return fmt.Errorf("empty icon data")
	}

	// Store PNG for explorer crash recovery and theme switching.
	t.iconData = png

	return t.applyIcon(png)
}

// SetDarkModeIcon sets an alternative icon displayed when Windows is in dark mode.
// If the system is currently in dark mode and the tray is visible, the icon
// switches immediately.
func (t *win32Tray) SetDarkModeIcon(png []byte) error {
	t.iconDark = png

	// If currently in dark mode and visible, switch immediately.
	if t.visible && len(png) > 0 && isSystemDarkMode() {
		return t.applyIcon(png)
	}

	return nil
}

// applyIcon converts PNG bytes to HICON and sets it as the current tray icon.
// Reused by SetIcon, SetDarkModeIcon, and theme switching logic.
func (t *win32Tray) applyIcon(png []byte) error {
	if len(png) == 0 {
		return fmt.Errorf("empty icon data")
	}

	hicon, err := pngToHICON(png)
	if err != nil {
		return fmt.Errorf("convert PNG to HICON: %w", err)
	}

	// Destroy previous icon if any.
	if t.hicon != 0 {
		if ret, _, _ := procDestroyIcon.Call(t.hicon); ret == 0 {
			slog.Warn("systray: DestroyIcon failed during icon replacement")
		}
	}
	t.hicon = hicon

	// If already visible, update the icon in the tray.
	if t.visible {
		return t.modifyIcon()
	}

	return nil
}

// SetTooltip sets the hover tooltip text.
func (t *win32Tray) SetTooltip(text string) error {
	t.tooltip = text

	if t.visible {
		return t.modifyIcon()
	}

	return nil
}

// SetMenu stores the menu for context menu display on right-click.
func (t *win32Tray) SetMenu(menu *Menu) error {
	t.menu = menu

	// Destroy old HMENU if any.
	if t.hmenu != 0 {
		if ret, _, _ := procDestroyMenu.Call(t.hmenu); ret == 0 {
			slog.Warn("systray: DestroyMenu failed during menu replacement")
		}
		t.hmenu = 0
	}

	if menu != nil && len(menu.Items) > 0 {
		hmenu, err := buildHMENU(menu)
		if err != nil {
			return fmt.Errorf("build HMENU: %w", err)
		}
		t.hmenu = hmenu
	}

	return nil
}

// ShowNotification displays a balloon notification from the tray icon.
// Uses Shell_NotifyIconW with NIM_MODIFY and NIF_INFO flag.
// Title is limited to 63 characters, message to 255 characters (Win32 limits).
func (t *win32Tray) ShowNotification(title, message string) error {
	if !t.visible {
		return fmt.Errorf("tray icon not visible, call Show() first")
	}

	nid := notifyIconData{
		cbSize:      uint32(unsafe.Sizeof(notifyIconData{})),
		hWnd:        t.hwnd,
		uID:         t.uid,
		uFlags:      nifInfo,
		dwInfoFlags: niifInfo,
	}

	// Copy title to szInfoTitle (max 63 chars + null terminator).
	if title != "" {
		titleUTF16, err := windows.UTF16FromString(title)
		if err == nil {
			maxLen := len(nid.szInfoTitle) - 1
			if len(titleUTF16) > maxLen {
				titleUTF16 = titleUTF16[:maxLen]
			}
			copy(nid.szInfoTitle[:], titleUTF16)
		}
	}

	// Copy message to szInfo (max 255 chars + null terminator).
	if message != "" {
		msgUTF16, err := windows.UTF16FromString(message)
		if err == nil {
			maxLen := len(nid.szInfo) - 1
			if len(msgUTF16) > maxLen {
				msgUTF16 = msgUTF16[:maxLen]
			}
			copy(nid.szInfo[:], msgUTF16)
		}
	}

	ret, _, _ := procShellNotifyIconW.Call(nimModify, uintptr(unsafe.Pointer(&nid)))
	if ret == 0 {
		return fmt.Errorf("Shell_NotifyIconW NIM_MODIFY (notification) failed")
	}

	return nil
}

// Show adds the icon to the system tray notification area.
// If a dark mode icon is set, the initial icon is chosen based on the
// current system theme (dark or light).
func (t *win32Tray) Show() error {
	if t.visible {
		return nil
	}

	nid := t.makeNID()
	nid.uFlags = nifMessage | nifIcon | nifTip

	ret, _, _ := procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
	if ret == 0 {
		return fmt.Errorf("Shell_NotifyIconW NIM_ADD failed")
	}

	// Set NOTIFYICON_VERSION_4 for proper event behavior.
	nid.uVersion = notifyIconVersion4
	ret, _, _ = procShellNotifyIconW.Call(nimSetVersion, uintptr(unsafe.Pointer(&nid)))
	if ret == 0 {
		slog.Warn("Shell_NotifyIconW NIM_SETVERSION failed, falling back to legacy behavior")
	}

	t.visible = true

	// Apply the correct icon for the current system theme.
	// This handles the case where dark mode icon was set before Show().
	t.updateIconForTheme()

	return nil
}

// Hide removes the icon from the tray without destroying the window.
func (t *win32Tray) Hide() error {
	if !t.visible {
		return nil
	}

	nid := notifyIconData{
		cbSize: uint32(unsafe.Sizeof(notifyIconData{})),
		hWnd:   t.hwnd,
		uID:    t.uid,
	}

	ret, _, _ := procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
	if ret == 0 {
		return fmt.Errorf("Shell_NotifyIconW NIM_DELETE failed")
	}

	t.visible = false
	return nil
}

// Bounds returns the tray icon's screen rectangle (x, y, width, height).
// Uses Shell_NotifyIconGetRect (Windows 7+). Returns (0,0,0,0) if the
// function is unavailable or the icon position cannot be determined.
func (t *win32Tray) Bounds() (int, int, int, int) {
	if !t.visible {
		return 0, 0, 0, 0
	}

	// Shell_NotifyIconGetRect may not exist on older systems.
	if err := procShellNotifyIconGetRect.Find(); err != nil {
		return 0, 0, 0, 0
	}

	nii := notifyIconIdentifier{
		cbSize: uint32(unsafe.Sizeof(notifyIconIdentifier{})),
		hWnd:   t.hwnd,
		uID:    t.uid,
	}

	var r rect
	ret, _, _ := procShellNotifyIconGetRect.Call(
		uintptr(unsafe.Pointer(&nii)),
		uintptr(unsafe.Pointer(&r)),
	)
	// Shell_NotifyIconGetRect returns S_OK (0) on success.
	if ret != 0 {
		return 0, 0, 0, 0
	}

	return int(r.left), int(r.top), int(r.right - r.left), int(r.bottom - r.top)
}

// Run blocks the calling goroutine, pumping the Win32 message loop.
// Returns when PostQuitMessage is called (via Quit or WM_DESTROY).
// All enterprise references (Qt6, getlantern/systray, fyne-io/systray)
// use GetMessage — 0% CPU when idle, correct WM_QUIT semantics.
func (t *win32Tray) Run() error {
	var m msg
	for {
		ret, _, err := procGetMessageW.Call(
			uintptr(unsafe.Pointer(&m)), 0, 0, 0,
		)
		switch int32(ret) {
		case -1:
			return fmt.Errorf("GetMessage failed: %w", err)
		case 0:
			return nil
		default:
			_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
			_, _, _ = procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
		}
	}
}

// Destroy removes the tray icon, destroys the window, and frees resources.
func (t *win32Tray) Destroy() {
	// Remove icon from tray.
	if t.visible {
		if err := t.Hide(); err != nil {
			slog.Warn("systray: Hide during Destroy failed", "err", err)
		}
	}

	// Unregister from the global registry.
	if t.hwnd != 0 {
		trayMu.Lock()
		delete(trayRegistry, t.hwnd)
		trayMu.Unlock()

		if ret, _, _ := procDestroyWindow.Call(t.hwnd); ret == 0 {
			slog.Warn("systray: DestroyWindow failed during cleanup")
		}
		t.hwnd = 0
	}

	// Free HICON.
	if t.hicon != 0 {
		if ret, _, _ := procDestroyIcon.Call(t.hicon); ret == 0 {
			slog.Warn("systray: DestroyIcon failed during cleanup")
		}
		t.hicon = 0
	}

	// Free HMENU.
	if t.hmenu != 0 {
		if ret, _, _ := procDestroyMenu.Call(t.hmenu); ret == 0 {
			slog.Warn("systray: DestroyMenu failed during cleanup")
		}
		t.hmenu = 0
	}
}

// makeNID builds a NOTIFYICONDATAW with current state.
func (t *win32Tray) makeNID() notifyIconData {
	nid := notifyIconData{
		cbSize:           uint32(unsafe.Sizeof(notifyIconData{})),
		hWnd:             t.hwnd,
		uID:              t.uid,
		uCallbackMessage: uint32(wmTrayCallback),
		hIcon:            t.hicon,
	}

	// Copy tooltip (UTF-16, max 127 chars + null terminator).
	if t.tooltip != "" {
		tip, err := windows.UTF16FromString(t.tooltip)
		if err == nil {
			maxLen := len(nid.szTip) - 1 // reserve space for null terminator
			if len(tip) > maxLen {
				tip = tip[:maxLen]
			}
			copy(nid.szTip[:], tip)
		}
	}

	return nid
}

// modifyIcon sends NIM_MODIFY to update the icon/tooltip in the tray.
func (t *win32Tray) modifyIcon() error {
	nid := t.makeNID()
	nid.uFlags = nifMessage | nifIcon | nifTip

	ret, _, _ := procShellNotifyIconW.Call(nimModify, uintptr(unsafe.Pointer(&nid)))
	if ret == 0 {
		return fmt.Errorf("Shell_NotifyIconW NIM_MODIFY failed")
	}

	return nil
}

// reAddIcon re-creates the tray icon after explorer.exe crash/restart.
// Selects the correct icon (dark or light) based on the current theme.
func (t *win32Tray) reAddIcon() {
	if !t.visible {
		return
	}

	// Re-create HICON from the theme-appropriate stored PNG.
	if t.hicon == 0 {
		iconPNG := t.themeIcon()
		if len(iconPNG) == 0 {
			return
		}
		hicon, err := pngToHICON(iconPNG)
		if err != nil {
			slog.Warn("systray: re-create HICON after explorer restart failed", "err", err)
			return
		}
		t.hicon = hicon
	}

	// Force visible=false so Show() will NIM_ADD again.
	t.visible = false
	if err := t.Show(); err != nil {
		slog.Warn("systray: re-add icon after explorer restart failed", "err", err)
	}
}

// --- Menu construction ---

// buildHMENU creates a Win32 HMENU from the internal Menu structure.
// Menu item IDs are 1-based sequential (0 is reserved for "no selection"
// in TrackPopupMenu with TPM_RETURNCMD).
func buildHMENU(menu *Menu) (uintptr, error) {
	hmenu, _, _ := procCreatePopupMenu.Call()
	if hmenu == 0 {
		return 0, fmt.Errorf("CreatePopupMenu failed")
	}

	if err := populateMenu(hmenu, menu); err != nil {
		if ret, _, _ := procDestroyMenu.Call(hmenu); ret == 0 {
			slog.Warn("systray: DestroyMenu failed during error cleanup")
		}
		return 0, err
	}

	return hmenu, nil
}

// populateMenu recursively adds items to an HMENU.
// Item IDs are assigned sequentially using a global counter that resets
// when buildHMENU is called. For submenus, items continue the sequence.
func populateMenu(hmenu uintptr, menu *Menu) error {
	for i, item := range menu.Items {
		switch item.Type {
		case MenuItemSeparator:
			ret, _, _ := procAppendMenuW.Call(
				hmenu,
				uintptr(mfSeparator),
				0,
				0,
			)
			if ret == 0 {
				return fmt.Errorf("AppendMenuW separator failed at index %d", i)
			}

		case MenuItemSubmenu:
			if item.Submenu == nil {
				continue
			}
			subHMenu, err := buildHMENU(item.Submenu)
			if err != nil {
				return fmt.Errorf("build submenu %q: %w", item.Label, err)
			}
			label, err := windows.UTF16PtrFromString(item.Label)
			if err != nil {
				// Best-effort cleanup of already-created submenu handle.
				_, _, _ = procDestroyMenu.Call(subHMenu)
				return fmt.Errorf("utf16 submenu label %q: %w", item.Label, err)
			}
			flags := uintptr(mfString | mfPopup)
			if item.Disabled {
				flags |= mfGrayed
			}
			ret, _, _ := procAppendMenuW.Call(
				hmenu,
				flags,
				subHMenu,
				uintptr(unsafe.Pointer(label)),
			)
			if ret == 0 {
				// Best-effort cleanup of already-created submenu handle.
				_, _, _ = procDestroyMenu.Call(subHMenu)
				return fmt.Errorf("AppendMenuW submenu %q failed", item.Label)
			}

		default: // MenuItemNormal, MenuItemCheckbox
			label, err := windows.UTF16PtrFromString(item.Label)
			if err != nil {
				return fmt.Errorf("utf16 menu label %q: %w", item.Label, err)
			}
			flags := uintptr(mfString)
			if item.Checked {
				flags |= mfChecked
			}
			if item.Disabled {
				flags |= mfGrayed
			}
			// Item ID is 1-based index for flat lookup via collectItems.
			itemID := uintptr(i + 1)
			ret, _, _ := procAppendMenuW.Call(
				hmenu,
				flags,
				itemID,
				uintptr(unsafe.Pointer(label)),
			)
			if ret == 0 {
				return fmt.Errorf("AppendMenuW item %q failed", item.Label)
			}
		}
	}
	return nil
}

// --- Window procedure ---

// trayWndProc handles messages for tray message-only windows.
func trayWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	// Look up the tray instance for this HWND.
	trayMu.RLock()
	t, ok := trayRegistry[hwnd]
	trayMu.RUnlock()

	if !ok {
		ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return ret
	}

	// Handle TaskbarCreated (explorer.exe restart).
	if taskbarCreatedMsg != 0 && msg == taskbarCreatedMsg {
		t.reAddIcon()
		return 0
	}

	switch msg {
	case wmTrayCallback:
		return t.handleTrayMessage(lParam)

	case wmCommand:
		// Menu item selected via WM_COMMAND (non-TPM_RETURNCMD path).
		// We use TPM_RETURNCMD, so this is a fallback.
		return 0

	case wmSettingChange:
		// Windows broadcasts WM_SETTINGCHANGE with lParam pointing to the
		// UTF-16 string "ImmersiveColorSet" when the system theme changes
		// (dark/light mode toggle in Settings > Personalization > Colors).
		if isImmersiveColorSet(lParam) {
			t.updateIconForTheme()
		}
		ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return ret

	case wmDestroy:
		return 0

	default:
		ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return ret
	}
}

// handleTrayMessage processes the tray callback message.
// With NOTIFYICON_VERSION_4, LOWORD(lParam) contains the actual event.
func (t *win32Tray) handleTrayMessage(lParam uintptr) uintptr {
	event := lParam & 0xFFFF // LOWORD

	switch event {
	case wmLButtonUp:
		if t.callbacks != nil && t.callbacks.OnClick != nil {
			t.callbacks.OnClick()
		}

	case wmLButtonDblClk:
		if t.callbacks != nil && t.callbacks.OnDoubleClick != nil {
			t.callbacks.OnDoubleClick()
		}

	case wmRButtonUp:
		if t.callbacks != nil && t.callbacks.OnRightClick != nil {
			t.callbacks.OnRightClick()
		}
		t.showContextMenu()

	case wmContextMenu:
		t.showContextMenu()
	}

	return 0
}

// showContextMenu displays the context menu at the current cursor position.
// Implements the required SetForegroundWindow + PostMessage(WM_NULL) pattern
// to ensure the menu dismisses properly when clicking outside.
func (t *win32Tray) showContextMenu() {
	if t.hmenu == 0 || t.menu == nil {
		return
	}

	// Get cursor position for menu placement.
	var pt point
	// GetCursorPos failure is non-fatal; menu appears at (0,0) as fallback.
	_, _, _ = procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	// Required: SetForegroundWindow before TrackPopupMenu.
	// Without this, the menu won't dismiss when clicking outside.
	// Return value is non-fatal (window may already be foreground).
	_, _, _ = procSetForegroundWindow.Call(t.hwnd)

	// Show menu and wait for selection. TPM_RETURNCMD makes TrackPopupMenu
	// return the selected item ID instead of posting WM_COMMAND.
	flags := uintptr(tpmLeftAlign | tpmRightButton | tpmReturnCmd | tpmNoNotify)
	ret, _, _ := procTrackPopupMenu.Call(
		t.hmenu,
		flags,
		uintptr(pt.x),
		uintptr(pt.y),
		0,
		t.hwnd,
		0,
	)

	// Required: PostMessage WM_NULL after TrackPopupMenu.
	// Fixes the dismiss-on-second-click problem.
	// Return value is non-fatal.
	_, _, _ = procPostMessageW.Call(t.hwnd, wmNull, 0, 0)

	// Dispatch the selected item (ret is 1-based index from populateMenu).
	if ret > 0 && t.menu != nil {
		idx := int(ret) - 1
		if idx >= 0 && idx < len(t.menu.Items) {
			item := t.menu.Items[idx]
			if item.OnClick != nil {
				item.OnClick()
			}
		}
	}
}

// --- Dark/light mode detection ---

// isSystemDarkMode checks the Windows registry to determine if the system
// is using dark mode. Returns true for dark mode, false for light mode.
// Reads HKCU\Software\Microsoft\Windows\CurrentVersion\Themes\Personalize\SystemUsesLightTheme.
// Returns false (light mode) if the registry key cannot be read (pre-Windows 10 1809).
func isSystemDarkMode() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`,
		registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer func() { _ = key.Close() }()

	val, _, err := key.GetIntegerValue("SystemUsesLightTheme")
	if err != nil {
		return false
	}

	// SystemUsesLightTheme: 0 = dark mode, 1 = light mode.
	return val == 0
}

// procLstrcmpiW is the Win32 lstrcmpiW function for case-insensitive
// null-terminated UTF-16 string comparison. Used to compare the
// WM_SETTINGCHANGE lParam string without unsafe.Pointer conversion.
var procLstrcmpiW = kernel32.NewProc("lstrcmpiW")

// immersiveColorSetPtr is a pre-allocated UTF-16 pointer to "ImmersiveColorSet"
// for comparison in isImmersiveColorSet.
var immersiveColorSetPtr *uint16

func init() {
	var err error
	immersiveColorSetPtr, err = windows.UTF16PtrFromString("ImmersiveColorSet")
	if err != nil {
		// This is a compile-time constant string, so this cannot fail.
		panic("systray: UTF16PtrFromString ImmersiveColorSet: " + err.Error())
	}
}

// isImmersiveColorSet checks whether the WM_SETTINGCHANGE lParam points to
// the UTF-16 string "ImmersiveColorSet", which Windows broadcasts when the
// system theme (dark/light) changes.
// Uses lstrcmpiW (kernel32) for comparison to avoid uintptr-to-unsafe.Pointer
// conversion that go vet flags.
func isImmersiveColorSet(lParam uintptr) bool {
	if lParam == 0 {
		return false
	}

	// lstrcmpiW returns 0 when strings are equal (case-insensitive).
	// Both lParam and immersiveColorSetPtr are valid null-terminated UTF-16 pointers.
	ret, _, _ := procLstrcmpiW.Call(lParam, uintptr(unsafe.Pointer(immersiveColorSetPtr)))
	return ret == 0
}

// updateIconForTheme switches the tray icon based on the current system theme.
// Uses the dark icon in dark mode (if set), otherwise the light (default) icon.
func (t *win32Tray) updateIconForTheme() {
	iconPNG := t.themeIcon()
	if len(iconPNG) == 0 {
		return
	}

	if err := t.applyIcon(iconPNG); err != nil {
		slog.Warn("systray: theme icon switch failed", "dark", isSystemDarkMode(), "err", err)
	}
}

// themeIcon returns the appropriate icon PNG for the current system theme.
// Returns the dark icon if dark mode is active and a dark icon is set,
// otherwise the light (default) icon.
func (t *win32Tray) themeIcon() []byte {
	if isSystemDarkMode() && len(t.iconDark) > 0 {
		return t.iconDark
	}
	return t.iconData
}

// --- PNG to HICON conversion ---

// pngToHICON converts raw PNG bytes to a Win32 HICON handle.
// Uses CreateIconFromResourceEx which accepts PNG directly on Vista+.
func pngToHICON(png []byte) (uintptr, error) {
	if len(png) == 0 {
		return 0, fmt.Errorf("empty PNG data")
	}

	// Get system tray icon size.
	cxIcon, _, _ := procGetSystemMetrics.Call(uintptr(smCXSmIcon))
	cyIcon, _, _ := procGetSystemMetrics.Call(uintptr(smCYSmIcon))
	if cxIcon == 0 {
		cxIcon = 16
	}
	if cyIcon == 0 {
		cyIcon = 16
	}

	hicon, _, _ := procCreateIconFromResourceEx.Call(
		uintptr(unsafe.Pointer(&png[0])),
		uintptr(len(png)),
		1,              // fIcon = TRUE (icon, not cursor)
		0x00030000,     // dwVer = 0x00030000 (required)
		cxIcon,         // cxDesired
		cyIcon,         // cyDesired
		lrDefaultColor, // fuLoad
	)
	if hicon == 0 {
		return 0, fmt.Errorf("CreateIconFromResourceEx failed for %d byte PNG", len(png))
	}

	return hicon, nil
}
