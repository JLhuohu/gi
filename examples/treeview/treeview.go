// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"image"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/units"
	"goki.dev/ki/v2/ki"
	"goki.dev/ki/v2/kit"
	"goki.dev/mat32/v2"
)

// todo: enum field, etc

// A node for testing
type TestNodeA struct {
	ki.Node

	// a string
	StrField string `desc:"a string"`

	// [min: 5] [max: 25] [step: 5] an int
	IntField int `min:"5" max:"25" step:"5" desc:"an int"`

	// [min: -1] [max: 1] [step: .25] a float
	FloatField float64 `min:"-1" max:"1" step:".25" desc:"a float"`

	// a bool
	BoolField bool `desc:"a bool"`

	// a vec
	Vec mat32.Vec2 `desc:"a vec"`

	// rect
	Rect image.Rectangle `desc:"rect"`
}

var TypeTestNodeA = kit.Types.AddType(&TestNodeA{}, nil)

// B node for testing
type TestNodeB struct {
	ki.Node

	// a string
	StrField string `desc:"a string"`

	// [min: 5] [max: 25] [step: 5] an int
	IntField int `min:"5" max:"25" step:"5" desc:"an int"`

	// [min: -1] [max: 1] [step: .25] a float
	FloatField float64 `min:"-1" max:"1" step:".25" desc:"a float"`

	// a bool
	BoolField bool `desc:"a bool"`

	// a vec
	Vec mat32.Vec2 `desc:"a vec"`

	// rect
	Rect image.Rectangle `desc:"rect"`

	// a sub-object
	SubObj TestNodeA `desc:"a sub-object"`
}

var TypeTestNodeB = kit.Types.AddType(&TestNodeB{}, nil)

func main() {
	gimain.Main(mainrun)
}

func mainrun() {
	// a source tree to view
	srctree := TestNodeB{}
	srctree.InitName(&srctree, "par1")
	// child1 :=
	srctree.AddNewChild(TypeTestNodeB, "child1")
	child2 := srctree.AddNewChild(TypeTestNodeB, "child2")
	// child3 :=
	srctree.AddNewChild(TypeTestNodeB, "child3")
	// schild2 :=
	child2.AddNewChild(TypeTestNodeB, "subchild1")

	srctree.SetProp("test1", "string val")
	srctree.SetProp("test2", 3.14)
	srctree.SetProp("test3", false)

	// turn this on to see a trace of the rendering
	// gi.Render2DTrace = true
	// gi.Layout2DTrace = true

	gi.SetAppName("treeview")
	gi.SetAppAbout(`This is a demo of the treeview in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>
<p>Full Drag-and-Drop, Copy / Cut / Paste, and Keyboard Navigation is supported.</p>`)

	width := 1024
	height := 768
	win := gi.NewMainWindow("gogi-treeview-test", "TreeView Test", width, height)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	trow := gi.AddNewLayout(mfr, "trow", gi.LayoutHoriz)
	trow.SetProp("horizontal-align", "center")
	trow.SetProp("margin", 2.0) // raw numbers = px = 96 dpi pixels
	trow.SetStretchMaxWidth()

	spc := gi.AddNewSpace(mfr, "spc1")
	spc.SetFixedHeight(units.Em(2))

	gi.AddNewStretch(trow, "str1")
	lab1 := gi.AddNewLabel(trow, "lab1", "This is a test of the TreeView and StructView reflect-ive GUI")
	lab1.SetStretchMaxWidth()
	lab1.SetProp("text-align", "center")
	gi.AddNewStretch(trow, "str2")

	split := gi.AddNewSplitView(mfr, "split")
	split.Dim = mat32.X

	tvfr := gi.AddNewFrame(split, "tvfr", gi.LayoutHoriz)
	svfr := gi.AddNewFrame(split, "svfr", gi.LayoutHoriz)
	split.SetSplits(.3, .7)

	tv := giv.AddNewTreeView(tvfr, "tv")
	tv.SetRootNode(&srctree)

	sv := giv.AddNewStructView(svfr, "sv")
	sv.SetStretchMaxWidth()
	sv.SetStretchMaxHeight()
	sv.SetStruct(&srctree)

	tv.TreeViewSig.Connect(sv.This(), func(recv, send ki.Ki, sig int64, data any) {
		if data == nil {
			return
		}
		// tvr, _ := send.Embed(giv.TypeTreeView).(*gi.TreeView) // root is sender
		tvn, _ := data.(ki.Ki).Embed(giv.TypeTreeView).(*giv.TreeView)
		svr, _ := recv.Embed(giv.TypeStructView).(*giv.StructView)
		if sig == int64(giv.TreeViewSelected) {
			svr.SetStruct(tvn.SrcNode)
		}
	})

	// main menu
	appnm := gi.AppName()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "Edit", "Window"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Action)
	amen.Menu = make(gi.Menu, 0, 10)
	amen.Menu.AddAppMenu(win)

	emen := win.MainMenu.ChildByName("Edit", 1).(*gi.Action)
	emen.Menu = make(gi.Menu, 0, 10)
	emen.Menu.AddCopyCutPaste(win)

	win.SetCloseCleanFunc(func(w *gi.Window) {
		go gi.Quit() // once main window is closed, quit
	})

	win.MainMenuUpdated()

	vp.UpdateEndNoSig(updt)

	win.StartEventLoop()
}
