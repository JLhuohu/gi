// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package oswin

import (
	"goki.dev/gi/v2/oswin/clip"
	"goki.dev/gi/v2/oswin/cursor"
	"goki.dev/ki/v2/kit"
)

// TheApp is the current oswin App -- only ever one in effect
var TheApp App

// App represents the overall OS GUI hardware, and creates Images, Textures
// and Windows, appropriate for that hardware / OS, and maintains data about
// the physical screen(s)
type App interface {
	// Platform returns the platform type -- can use this for conditionalizing
	// behavior in minor, simple ways
	Platform() Platforms

	// Name is the overall name of the application -- used for specifying an
	// application-specific preferences directory, etc
	Name() string

	// SetName sets the application name -- defaults to GoGi if not otherwise set
	SetName(name string)

	// GetScreens gets the current list of screens
	GetScreens()

	// NScreens returns the number of different logical and/or physical
	// screens managed under this overall screen hardware
	NScreens() int

	// Screen returns screen for given screen number, or nil if not a
	// valid screen number.
	Screen(scrN int) *Screen

	// ScreenByName returns screen for given screen name, or nil if not a
	// valid screen name.
	ScreenByName(name string) *Screen

	// NoScreens returns true if there are no active screens currently
	// (e.g., for a closed laptop with no external monitor attached)
	// The previous list of Screens is retained so this is the check.
	NoScreens() bool

	// NWindows returns the number of windows open for this app.
	NWindows() int

	// Window returns given window in list of windows opened under this screen
	// -- list is not in any guaranteed order, but typically in order of
	// creation (see also WindowByName) -- returns nil for invalid index.
	Window(win int) Window

	// WindowByName returns given window in list of windows opened under this
	// screen, by name -- nil if not found.
	WindowByName(name string) Window

	// WindowInFocus returns the window currently in focus (receiving keyboard
	// input) -- could be nil if none are.
	WindowInFocus() Window

	// ContextWindow returns the window passed as context for clipboard, cursor, etc calls.
	ContextWindow() Window

	// NewWindow returns a new Window for this screen. A nil opts is valid and
	// means to use the default option values.
	NewWindow(opts *NewWindowOptions) (Window, error)

	// ClipBoard returns the clip.Board handler for the system, in context of given window.
	ClipBoard(win Window) clip.Board

	// Cursor returns the cursor.Cursor handler for the system, in context of given window.
	Cursor(win Window) cursor.Cursor

	// PrefsDir returns the OS-specific preferences directory: Mac: ~/Library,
	// Linux: ~/.config, Windows: ?
	PrefsDir() string

	// GoGiPrefsDir returns the GoGi preferences directory: PrefsDir + GoGi --
	// ensures that the directory exists first.
	GoGiPrefsDir() string

	// AppPrefsDir returns the application-specific preferences directory:
	// PrefsDir + App.Name --ensures that the directory exists first.
	AppPrefsDir() string

	// FontPaths returns the default system font paths.
	FontPaths() []string

	// About is an informative message about the app.  Can use HTML
	// formatting, including links.
	About() string

	// SetAbout sets the about info.
	SetAbout(about string)

	// OpenURL opens the given URL in the user's default browser.  On Linux
	// this requires that xdg-utils package has been installed -- uses
	// xdg-open command.
	OpenURL(url string)

	// OpenFiles returns file names that have been set to be open at startup.
	OpenFiles() []string

	// SetQuitReqFunc sets the function that is called whenever there is a
	// request to quit the app (via a OS or a call to QuitReq() method).  That
	// function can then adjudicate whether and when to actually call Quit.
	SetQuitReqFunc(fun func())

	// SetQuitCleanFunc sets the function that is called whenever app is
	// actually about to quit (irrevocably) -- can do any necessary
	// last-minute cleanup here.
	SetQuitCleanFunc(fun func())

	// QuitReq is a quit request, triggered either by OS or user call (e.g.,
	// via Quit menu action) -- calls function previously-registered by
	// SetQuitReqFunc, which is then solely responsible for actually calling
	// Quit.
	QuitReq()

	// IsQuitting returns true when the app is actually quitting -- it is set
	// to true just before the QuitClean function is called, and before all
	// the windows are closed.
	IsQuitting() bool

	// QuitClean calls the function setup in SetQuitCleanFunc and does other
	// app cleanup -- called on way to quitting.
	QuitClean()

	// Quit closes all windows and exits the program.
	Quit()

	// RunOnMain runs given function on main thread (where main event loop is running)
	// Some functions (GUI-specific etc) must run on this initial main thread for the
	// overall app.
	RunOnMain(f func())

	// GoRunOnMain runs given function on main thread and returns immediately
	// Some functions (GUI-specific etc) must run on this initial main thread for the
	// overall app.
	GoRunOnMain(f func())

	// SendEmptyEvent sends an empty, blank event to global event processing
	// system, which has the effect of pushing the system along during cases when
	// the event loop needs to be "pinged" to get things moving along.
	// See also similar method on Window.
	SendEmptyEvent()

	// PollEvents tells the main event loop to check for any gui events right now.
	// Call this periodically from longer-running functions to ensure
	// GUI responsiveness.
	PollEvents()

	// ShowVirtualKeyboard shows a virtual keyboard of the given type.
	// ShowVirtualKeyboard only has an effect on mobile platforms (iOS and Android).
	ShowVirtualKeyboard(typ VirtualKeyboardTypes)

	// HideVirtualKeyboard hides the virtual keyboard.
	// HideVirtualKeyboard only has an effect on mobile platforms (iOS and Android).
	HideVirtualKeyboard()
}

// Platforms are all the supported platforms for OSWin
type Platforms int32

const (
	// MacOS is a mac desktop machine (aka Darwin)
	MacOS Platforms = iota

	// LinuxX11 is a Linux OS machine running X11 window server
	LinuxX11

	// Windows is a Microsoft Windows machine
	Windows

	// IOS is an Apple iOS or iPadOS mobile phone or iPad
	IOS

	// Android is an Android mobile phone or tablet
	Android

	PlatformsN
)

var TypePlatforms = kit.Enums.AddEnum(PlatformsN, kit.NotBitFlag, nil)

// IsMobile returns whether the platform is a mobile platform (iOS or Android)
func (p Platforms) IsMobile() bool {
	return p == IOS || p == Android
}

// VirtualKeyboardTypes are all of the supported virtual keyboard types for mobile platforms
type VirtualKeyboardTypes int32

const (
	// DefaultKeyboard is the keyboard with default input style and "return" return key
	DefaultKeyboard VirtualKeyboardTypes = iota
	// SingleLineKeyboard is the keyboard with default input style and "Done" return key
	SingleLineKeyboard
	// NumberKeyboard is the keyboard with number input style and "Done" return key
	NumberKeyboard

	VirtualKeyboardTypesN
)
