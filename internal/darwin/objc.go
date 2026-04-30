//go:build darwin

// Package darwin provides a minimal Objective-C runtime wrapper for the systray
// package. It loads libobjc.A.dylib and AppKit via goffi, exposing the core
// primitives needed for NSStatusBar/NSStatusItem/NSMenu interaction.
//
// This is a self-contained subset of the ObjC wrapper in gogpu/gogpu, tailored
// to the systray use case. It avoids importing gogpu to keep the dependency
// graph small.
package darwin

import (
	"errors"
	"sync"
	"unsafe"

	"github.com/go-webgpu/goffi/ffi"
	"github.com/go-webgpu/goffi/types"
)

// Errors returned by Objective-C runtime operations.
var (
	ErrLibraryNotLoaded = errors.New("darwin: failed to load library")
	ErrSymbolNotFound   = errors.New("darwin: symbol not found")
	ErrClassNotFound    = errors.New("darwin: class not found")
)

// ID represents an Objective-C object pointer.
type ID uintptr

// Class represents an Objective-C class pointer.
type Class uintptr

// SEL represents an Objective-C selector (method name).
type SEL uintptr

// NSSize represents the dimensions of a rectangle.
type NSSize struct {
	Width  float64
	Height float64
}

// objcRuntime holds the loaded Objective-C runtime library and function pointers.
type objcRuntime struct {
	once sync.Once
	err  error

	// Library handles
	libobjc    unsafe.Pointer
	foundation unsafe.Pointer
	appKit     unsafe.Pointer

	// Function pointers
	objcGetClass          unsafe.Pointer
	objcMsgSend           unsafe.Pointer
	selRegisterName       unsafe.Pointer
	objcAllocateClassPair unsafe.Pointer
	classAddMethod        unsafe.Pointer
	objcRegisterClassPair unsafe.Pointer

	// Reusable call interfaces
	cifVoidPtr  *types.CallInterface // Returns pointer, 2 args (self, _cmd)
	cifSelector *types.CallInterface // Returns pointer, 1 arg (const char*)
}

// rt is the global Objective-C runtime state.
var rt objcRuntime

var nsSizeType = &types.TypeDescriptor{
	Size:      16,
	Alignment: 8,
	Kind:      types.StructType,
	Members: []*types.TypeDescriptor{
		types.DoubleTypeDescriptor,
		types.DoubleTypeDescriptor,
	},
}

// initRuntime initializes the Objective-C runtime, loading libraries and
// resolving symbols. It is called once on first use via sync.Once.
func initRuntime() error {
	rt.once.Do(func() {
		rt.err = loadRuntime()
	})
	return rt.err
}

// loadRuntime performs the actual library loading and symbol resolution.
func loadRuntime() error {
	var err error

	// Load libobjc.A.dylib
	rt.libobjc, err = ffi.LoadLibrary("/usr/lib/libobjc.A.dylib")
	if err != nil {
		return errors.Join(ErrLibraryNotLoaded, err)
	}

	// Load Foundation framework
	rt.foundation, err = ffi.LoadLibrary(
		"/System/Library/Frameworks/Foundation.framework/Foundation")
	if err != nil {
		return errors.Join(ErrLibraryNotLoaded, err)
	}

	// Load AppKit framework
	rt.appKit, err = ffi.LoadLibrary(
		"/System/Library/Frameworks/AppKit.framework/AppKit")
	if err != nil {
		return errors.Join(ErrLibraryNotLoaded, err)
	}

	// Resolve symbols
	rt.objcGetClass, err = ffi.GetSymbol(rt.libobjc, "objc_getClass")
	if err != nil {
		return errors.Join(ErrSymbolNotFound, err)
	}

	rt.objcMsgSend, err = ffi.GetSymbol(rt.libobjc, "objc_msgSend")
	if err != nil {
		return errors.Join(ErrSymbolNotFound, err)
	}

	rt.selRegisterName, err = ffi.GetSymbol(rt.libobjc, "sel_registerName")
	if err != nil {
		return errors.Join(ErrSymbolNotFound, err)
	}

	rt.objcAllocateClassPair, err = ffi.GetSymbol(rt.libobjc, "objc_allocateClassPair")
	if err != nil {
		return errors.Join(ErrSymbolNotFound, err)
	}

	rt.classAddMethod, err = ffi.GetSymbol(rt.libobjc, "class_addMethod")
	if err != nil {
		return errors.Join(ErrSymbolNotFound, err)
	}

	rt.objcRegisterClassPair, err = ffi.GetSymbol(rt.libobjc, "objc_registerClassPair")
	if err != nil {
		return errors.Join(ErrSymbolNotFound, err)
	}

	// Prepare reusable call interfaces.

	// CIF for generic pointer-returning calls: (self, _cmd) -> pointer
	rt.cifVoidPtr = &types.CallInterface{}
	err = ffi.PrepareCallInterface(
		rt.cifVoidPtr,
		types.DefaultCall,
		types.PointerTypeDescriptor,
		[]*types.TypeDescriptor{
			types.PointerTypeDescriptor, // self (ID)
			types.PointerTypeDescriptor, // _cmd (SEL)
		},
	)
	if err != nil {
		return err
	}

	// CIF for sel_registerName: (const char*) -> SEL
	rt.cifSelector = &types.CallInterface{}
	err = ffi.PrepareCallInterface(
		rt.cifSelector,
		types.DefaultCall,
		types.PointerTypeDescriptor,
		[]*types.TypeDescriptor{
			types.PointerTypeDescriptor, // name
		},
	)
	if err != nil {
		return err
	}

	return nil
}

// GetClass returns the Objective-C class with the given name.
// Returns 0 if the class is not found.
func GetClass(name string) Class {
	if err := initRuntime(); err != nil {
		return 0
	}

	cname := append([]byte(name), 0)

	var result uintptr
	namePtr := unsafe.Pointer(&cname[0])
	argBox := &struct {
		name unsafe.Pointer
	}{
		name: namePtr,
	}

	err := ffi.CallFunction(
		rt.cifSelector,
		rt.objcGetClass,
		unsafe.Pointer(&result),
		[]unsafe.Pointer{unsafe.Pointer(&argBox.name)},
	)
	if err != nil {
		return 0
	}

	return Class(result)
}

// RegisterSelector registers a selector name and returns its SEL.
// Selectors are cached by the runtime, so calling this multiple times
// with the same name returns the same SEL.
func RegisterSelector(name string) SEL {
	if err := initRuntime(); err != nil {
		return 0
	}

	cname := append([]byte(name), 0)

	var result uintptr
	namePtr := unsafe.Pointer(&cname[0])
	argBox := &struct {
		name unsafe.Pointer
	}{
		name: namePtr,
	}

	err := ffi.CallFunction(
		rt.cifSelector,
		rt.selRegisterName,
		unsafe.Pointer(&result),
		[]unsafe.Pointer{unsafe.Pointer(&argBox.name)},
	)
	if err != nil {
		return 0
	}

	return SEL(result)
}

// Send sends a zero-argument message to an Objective-C object and returns the result.
func (id ID) Send(sel SEL) ID {
	if id == 0 || sel == 0 {
		return 0
	}

	if err := initRuntime(); err != nil {
		return 0
	}

	var result uintptr
	argBox := &struct {
		self uintptr
		cmd  uintptr
	}{
		self: uintptr(id),
		cmd:  uintptr(sel),
	}

	err := ffi.CallFunction(
		rt.cifVoidPtr,
		rt.objcMsgSend,
		unsafe.Pointer(&result),
		[]unsafe.Pointer{
			unsafe.Pointer(&argBox.self),
			unsafe.Pointer(&argBox.cmd),
		},
	)
	if err != nil {
		return 0
	}

	return ID(result)
}

// SendClass sends a message to a Class and returns the result.
// Used for class methods like [NSStatusBar systemStatusBar].
func (c Class) Send(sel SEL) ID {
	return ID(c).Send(sel)
}

// IsNil returns true if the ID is nil (zero).
func (id ID) IsNil() bool {
	return id == 0
}

// Ptr returns the ID as a uintptr for use with FFI.
func (id ID) Ptr() uintptr {
	return uintptr(id)
}

// msgSend is a low-level helper that calls objc_msgSend with arbitrary arguments.
// Creates a new CIF for each call. For hot paths, use dedicated CIF-based methods.
func msgSend(self ID, sel SEL, args ...uintptr) ID {
	if self == 0 || sel == 0 {
		return 0
	}

	if err := initRuntime(); err != nil {
		return 0
	}

	// Build argument type list: self, _cmd, then user args
	argTypes := make([]*types.TypeDescriptor, 2+len(args))
	argTypes[0] = types.PointerTypeDescriptor // self
	argTypes[1] = types.PointerTypeDescriptor // _cmd
	for i := range args {
		argTypes[2+i] = types.PointerTypeDescriptor
	}

	cif := &types.CallInterface{}
	err := ffi.PrepareCallInterface(
		cif,
		types.DefaultCall,
		types.PointerTypeDescriptor,
		argTypes,
	)
	if err != nil {
		return 0
	}

	// Build argument values: self, sel, then user args (heap-backed).
	argVals := make([]uintptr, 2+len(args))
	argVals[0] = uintptr(self)
	argVals[1] = uintptr(sel)
	copy(argVals[2:], args)

	argPtrs := make([]unsafe.Pointer, 2+len(args))
	for i := range argVals {
		argPtrs[i] = unsafe.Pointer(&argVals[i])
	}

	var result uintptr
	err = ffi.CallFunction(
		cif,
		rt.objcMsgSend,
		unsafe.Pointer(&result),
		argPtrs,
	)
	if err != nil {
		return 0
	}

	return ID(result)
}

// SendPtr sends a message with one pointer argument.
func (id ID) SendPtr(sel SEL, arg uintptr) ID {
	return msgSend(id, sel, arg)
}

// SendBool sends a message with one boolean argument.
func (id ID) SendBool(sel SEL, arg bool) ID {
	var val uintptr
	if arg {
		val = 1
	}
	return msgSend(id, sel, val)
}

// SendInt sends a message with one integer argument.
func (id ID) SendInt(sel SEL, arg int64) ID {
	return msgSend(id, sel, uintptr(arg))
}

// SendSize sends a message with an NSSize argument.
func (id ID) SendSize(sel SEL, size NSSize) ID {
	if id == 0 || sel == 0 {
		return 0
	}

	if err := initRuntime(); err != nil {
		return 0
	}

	argTypes := []*types.TypeDescriptor{
		types.PointerTypeDescriptor, // self
		types.PointerTypeDescriptor, // _cmd
		nsSizeType,                  // size
	}

	cif := &types.CallInterface{}
	err := ffi.PrepareCallInterface(
		cif,
		types.DefaultCall,
		types.PointerTypeDescriptor,
		argTypes,
	)
	if err != nil {
		return 0
	}

	argBox := &struct {
		self uintptr
		sel  uintptr
		size NSSize
	}{
		self: uintptr(id),
		sel:  uintptr(sel),
		size: size,
	}

	argPtrs := []unsafe.Pointer{
		unsafe.Pointer(&argBox.self),
		unsafe.Pointer(&argBox.sel),
		unsafe.Pointer(&argBox.size),
	}

	var result uintptr
	err = ffi.CallFunction(
		cif,
		rt.objcMsgSend,
		unsafe.Pointer(&result),
		argPtrs,
	)
	if err != nil {
		return 0
	}

	return ID(result)
}

// SendDouble sends a message with one double (CGFloat) argument.
func (id ID) SendDouble(sel SEL, arg float64) ID {
	if id == 0 || sel == 0 {
		return 0
	}

	if err := initRuntime(); err != nil {
		return 0
	}

	argTypes := []*types.TypeDescriptor{
		types.PointerTypeDescriptor, // self
		types.PointerTypeDescriptor, // _cmd
		types.DoubleTypeDescriptor,  // arg
	}

	cif := &types.CallInterface{}
	err := ffi.PrepareCallInterface(
		cif,
		types.DefaultCall,
		types.PointerTypeDescriptor,
		argTypes,
	)
	if err != nil {
		return 0
	}

	argBox := &struct {
		self uintptr
		sel  uintptr
		arg  float64
	}{
		self: uintptr(id),
		sel:  uintptr(sel),
		arg:  arg,
	}

	argPtrs := []unsafe.Pointer{
		unsafe.Pointer(&argBox.self),
		unsafe.Pointer(&argBox.sel),
		unsafe.Pointer(&argBox.arg),
	}

	var result uintptr
	err = ffi.CallFunction(
		cif,
		rt.objcMsgSend,
		unsafe.Pointer(&result),
		argPtrs,
	)
	if err != nil {
		return 0
	}

	return ID(result)
}

// MsgSend3Ptr calls objc_msgSend with self, sel, and 3 pointer arguments.
// Used for initWithTitle:action:keyEquivalent: on NSMenuItem.
func MsgSend3Ptr(id ID, sel SEL, arg0, arg1, arg2 uintptr) ID {
	if err := initRuntime(); err != nil {
		return 0
	}

	cif := &types.CallInterface{}
	if err := ffi.PrepareCallInterface(cif, types.DefaultCall,
		types.PointerTypeDescriptor,
		[]*types.TypeDescriptor{
			types.PointerTypeDescriptor, // self
			types.PointerTypeDescriptor, // _cmd
			types.PointerTypeDescriptor, // arg0
			types.PointerTypeDescriptor, // arg1
			types.PointerTypeDescriptor, // arg2
		},
	); err != nil {
		return 0
	}

	self := uintptr(id)
	cmd := uintptr(sel)

	var result uintptr
	if err := ffi.CallFunction(cif,
		rt.objcMsgSend,
		unsafe.Pointer(&result),
		[]unsafe.Pointer{
			unsafe.Pointer(&self),
			unsafe.Pointer(&cmd),
			unsafe.Pointer(&arg0),
			unsafe.Pointer(&arg1),
			unsafe.Pointer(&arg2),
		},
	); err != nil {
		return 0
	}
	return ID(result)
}

// MsgSendPtrPtr calls objc_msgSend with self, sel, and 2 pointer arguments.
// Used for methods like dataWithBytes:length:.
func MsgSendPtrPtr(id ID, sel SEL, arg0, arg1 uintptr) ID {
	return msgSend(id, sel, arg0, arg1)
}

// AllocateClassPair creates a new ObjC class as a subclass of superclass.
// Returns the new Class, or 0 if allocation fails.
// Call RegisterClassPair after adding methods.
func AllocateClassPair(superclass Class, name string) Class {
	if err := initRuntime(); err != nil {
		return 0
	}

	nameBytes := append([]byte(name), 0)

	cif := &types.CallInterface{}
	if err := ffi.PrepareCallInterface(cif, types.DefaultCall,
		types.PointerTypeDescriptor,
		[]*types.TypeDescriptor{
			types.PointerTypeDescriptor,
			types.PointerTypeDescriptor,
			types.PointerTypeDescriptor,
		},
	); err != nil {
		return 0
	}

	super := uintptr(superclass)
	namePtr := uintptr(unsafe.Pointer(&nameBytes[0]))
	var extraBytes uintptr

	var result uintptr
	if err := ffi.CallFunction(cif,
		rt.objcAllocateClassPair,
		unsafe.Pointer(&result),
		[]unsafe.Pointer{
			unsafe.Pointer(&super),
			unsafe.Pointer(&namePtr),
			unsafe.Pointer(&extraBytes),
		},
	); err != nil {
		return 0
	}
	return Class(result)
}

// ClassAddMethod adds a method to a class. imp is a C function pointer
// (use ffi.NewCallback to create from Go function). typeEncoding is the ObjC
// type encoding string (e.g., "v@:@" for void(id,SEL,id)).
func ClassAddMethod(cls Class, sel SEL, imp uintptr, typeEncoding string) bool {
	if err := initRuntime(); err != nil {
		return false
	}

	typeBytes := append([]byte(typeEncoding), 0)

	cif := &types.CallInterface{}
	if err := ffi.PrepareCallInterface(cif, types.DefaultCall,
		types.PointerTypeDescriptor,
		[]*types.TypeDescriptor{
			types.PointerTypeDescriptor,
			types.PointerTypeDescriptor,
			types.PointerTypeDescriptor,
			types.PointerTypeDescriptor,
		},
	); err != nil {
		return false
	}

	clsPtr := uintptr(cls)
	selPtr := uintptr(sel)
	typePtr := uintptr(unsafe.Pointer(&typeBytes[0]))

	var result uintptr
	if err := ffi.CallFunction(cif,
		rt.classAddMethod,
		unsafe.Pointer(&result),
		[]unsafe.Pointer{
			unsafe.Pointer(&clsPtr),
			unsafe.Pointer(&selPtr),
			unsafe.Pointer(&imp),
			unsafe.Pointer(&typePtr),
		},
	); err != nil {
		return false
	}
	return result != 0
}

// RegisterClassPair registers a class that was allocated with AllocateClassPair.
func RegisterClassPair(cls Class) {
	if err := initRuntime(); err != nil {
		return
	}

	cif := &types.CallInterface{}
	if err := ffi.PrepareCallInterface(cif, types.DefaultCall,
		types.VoidTypeDescriptor,
		[]*types.TypeDescriptor{
			types.PointerTypeDescriptor,
		},
	); err != nil {
		return
	}

	clsPtr := uintptr(cls)
	_ = ffi.CallFunction(cif,
		rt.objcRegisterClassPair,
		nil,
		[]unsafe.Pointer{
			unsafe.Pointer(&clsPtr),
		},
	)
}

// NewNSString creates an NSString from a Go string.
// The caller is responsible for releasing the returned object.
func NewNSString(s string) ID {
	if err := initRuntime(); err != nil {
		return 0
	}

	nsStringClass := GetClass("NSString")
	if nsStringClass == 0 {
		return 0
	}

	selAlloc := RegisterSelector("alloc")
	selInitWithUTF8String := RegisterSelector("initWithUTF8String:")

	nsstr := nsStringClass.Send(selAlloc)
	if nsstr.IsNil() {
		return 0
	}

	cstr := append([]byte(s), 0)
	nsstr = nsstr.SendPtr(selInitWithUTF8String, uintptr(unsafe.Pointer(&cstr[0])))

	return nsstr
}

// NewNSData creates an NSData object from a byte slice.
// Uses [NSData dataWithBytes:length:].
func NewNSData(data []byte) ID {
	if err := initRuntime(); err != nil {
		return 0
	}

	if len(data) == 0 {
		return 0
	}

	nsDataClass := GetClass("NSData")
	if nsDataClass == 0 {
		return 0
	}

	selDataWithBytes := RegisterSelector("dataWithBytes:length:")

	return MsgSendPtrPtr(ID(nsDataClass), selDataWithBytes,
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
	)
}
