// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package oswin

import (
	"image"

	"github.com/goki/gi/oswin/clip"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/ki/kit"
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

	// NScreens returns the number of different logical and/or physical
	// screens managed under this overall screen hardware
	NScreens() int

	// Screen returns screen for given screen number, or nil if not a
	// valid screen number.
	Screen(scrN int) *Screen

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

	// NewWindow returns a new Window for this screen. A nil opts is valid and
	// means to use the default option values.
	NewWindow(opts *NewWindowOptions) (Window, error)

	// NewImage returns a new Image for this screen.  Images can be drawn upon
	// directly using image and other packages, and have an accessable []byte
	// slice holding the image data.
	NewImage(size image.Point) (Image, error)

	// NewTexture returns a new Texture for the given window.  Textures are opaque
	// and could be non-local, but very fast for rendering to windows --
	// typically create a texture of each window and render to that texture,
	// then Draw that texture to the window when it is time to update (call
	// Publish on window after drawing).
	NewTexture(win Window, size image.Point) (Texture, error)

	// ClipBoard returns the clip.Board handler for the system.
	ClipBoard() clip.Board

	// Cursor returns the cursor.Cursor handler for the system.
	Cursor() cursor.Cursor

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

	// SetQuitReqFunc sets the function that is called whenver there is a
	// request to quit the app (via a OS or a call to QuitReq() method).  That
	// function can then adjudicate whether and when to actually call Quit.
	SetQuitReqFunc(fun func())

	// SetQuitCleanFunc sets the function that is called whenver app is
	// actually about to quit (irrevocably) -- can do any necessary
	// last-minute cleanup here.
	SetQuitCleanFunc(fun func())

	// QuitReq is a quit request, triggered either by OS or user call (e.g.,
	// via Quit menu action) -- calls function previously-registered by
	// SetQuitReqFunc, which is then solely responsible for actually calling
	// Quit.
	QuitReq()

	// QuitClean calls the function setup in SetQuitCleanFunc and does other
	// app cleanup -- called on way to quitting.
	QuitClean()

	// Quit closes all windows and exits the program.
	Quit()
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

	PlatformsN
)

//go:generate stringer -type=Platforms

var KiT_Platforms = kit.Enums.AddEnum(PlatformsN, false, nil)
