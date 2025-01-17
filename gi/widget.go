// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"sync"

	"goki.dev/gi/v2/girl"
	"goki.dev/gi/v2/gist"
	"goki.dev/gi/v2/icons"
	"goki.dev/gi/v2/oswin"
	"goki.dev/gi/v2/oswin/cursor"
	"goki.dev/gi/v2/oswin/mouse"
	"goki.dev/gi/v2/units"
	"goki.dev/ki/v2/ints"
	"goki.dev/ki/v2/ki"
	"goki.dev/ki/v2/kit"
	"goki.dev/mat32/v2"
	"goki.dev/prof/v2"
)

// WidgetBase is the base type for all Widget Node2D elements, which are
// managed by a containing Layout, and use all 5 rendering passes.  All
// elemental widgets must support the Node Inactive and Selected states in a
// reasonable way (Selected only essential when also Inactive), so they can
// function appropriately in a chooser (e.g., SliceView or TableView) -- this
// includes toggling selection on left mouse press.
type WidgetBase struct {
	Node2DBase

	// text for tooltip for this widget -- can use HTML formatting
	Tooltip string `desc:"text for tooltip for this widget -- can use HTML formatting"`

	// a slice of stylers that are called in sequential descending order (so the first added styler is called last and thus overrides all other functions) to style the element; these should be set using AddStyler, which can be called by end-user and internal code
	Stylers []Styler `json:"-" xml:"-" copy:"-" desc:"a slice of stylers that are called in sequential descending order (so the first added styler is called last and thus overrides all other functions) to style the element; these should be set using AddStyler, which can be called by end-user and internal code"`

	// override the computed styles and allow directly editing Style
	OverrideStyle bool `json:"-" xml:"-" desc:"override the computed styles and allow directly editing Style"`

	// styling settings for this widget -- set in SetStyle2D during an initialization step, and when the structure changes; they are determined by, in increasing priority order, the default values, the ki node properties, and the StyleFunc (the recommended way to set styles is through the StyleFunc -- setting this field directly outside of that will have no effect unless OverrideStyle is on)
	Style gist.Style `json:"-" xml:"-" desc:"styling settings for this widget -- set in SetStyle2D during an initialization step, and when the structure changes; they are determined by, in increasing priority order, the default values, the ki node properties, and the StyleFunc (the recommended way to set styles is through the StyleFunc -- setting this field directly outside of that will have no effect unless OverrideStyle is on)"`

	// all the layout state information for this item
	LayState LayoutState `copy:"-" json:"-" xml:"-" desc:"all the layout state information for this item"`

	// [view: -] general widget signals supported by all widgets, including select, focus, and context menu (right mouse button) events, which can be used by views and other compound widgets
	WidgetSig ki.Signal `copy:"-" json:"-" xml:"-" view:"-" desc:"general widget signals supported by all widgets, including select, focus, and context menu (right mouse button) events, which can be used by views and other compound widgets"`

	// [view: -] optional context menu function called by MakeContextMenu AFTER any native items are added -- this function can decide where to insert new elements -- typically add a separator to disambiguate
	CtxtMenuFunc CtxtMenuFunc `copy:"-" view:"-" json:"-" xml:"-" desc:"optional context menu function called by MakeContextMenu AFTER any native items are added -- this function can decide where to insert new elements -- typically add a separator to disambiguate"`

	// [view: -] mutex protecting updates to the style
	StyMu sync.RWMutex `copy:"-" view:"-" json:"-" xml:"-" desc:"mutex protecting updates to the style"`
}

var TypeWidgetBase = kit.Types.AddType(&WidgetBase{}, WidgetBaseProps)

var WidgetBaseProps = ki.Props{
	"base-type":     true,
	ki.EnumTypeFlag: TypeNodeFlags,
}

// KiAsWidget returns the given Ki object
// as a widget base. It returns nil if it is not
// derived from a widget base.
func KiAsWidget(k ki.Ki) *WidgetBase {
	if n, ok := k.(Node2D); ok {
		return n.AsWidget()
	}
	return nil
}

func (wb *WidgetBase) CopyFieldsFrom(frm any) {
	fr, ok := frm.(*WidgetBase)
	if !ok {
		log.Printf("GoGi node of type: %v needs a CopyFieldsFrom method defined -- currently falling back on earlier WidgetBase one\n", ki.Type(wb).Name())
		ki.GenCopyFieldsFrom(wb.This(), frm)
		return
	}
	wb.Node2DBase.CopyFieldsFrom(&fr.Node2DBase)
	wb.Tooltip = fr.Tooltip
	wb.Style.CopyFrom(&fr.Style)
}

func (wb *WidgetBase) Disconnect() {
	wb.Node2DBase.Disconnect()
	wb.WidgetSig.DisconnectAll()
}

func (wb *WidgetBase) AsWidget() *WidgetBase {
	return wb
}

// ActiveStyle satisfies the ActiveStyler interface
// and returns the active style of the widget
func (wb *WidgetBase) ActiveStyle() *gist.Style {
	return &wb.Style
}

// StyleRLock does a read-lock for reading the style
func (wb *WidgetBase) StyleRLock() {
	wb.StyMu.RLock()
}

// StyleRUnlock unlocks the read-lock
func (wb *WidgetBase) StyleRUnlock() {
	wb.StyMu.RUnlock()
}

// BoxSpace returns the style BoxSpace value under read lock
func (wb *WidgetBase) BoxSpace() gist.SideFloats {
	wb.StyMu.RLock()
	bs := wb.Style.BoxSpace()
	wb.StyMu.RUnlock()
	return bs
}

// Init2DWidget handles basic node initialization -- Init2D can then do special things
func (wb *WidgetBase) Init2DWidget() {
	// fmt.Println("Init2DWidget", wb.Path())
	wb.BBoxMu.Lock()
	wb.StyMu.Lock()
	wb.Viewport = wb.ParentViewport()
	wb.Style.Defaults()
	wb.StyMu.Unlock()
	wb.LayState.Defaults() // doesn't overwrite
	wb.BBoxMu.Unlock()
	wb.ConnectToViewport()
}

func (wb *WidgetBase) Init2D() {
	wb.Init2DWidget()
}

// AddStyler adds the given styler to the
// widget's stylers, initializing them if necessary.
// This function can be called by both internal
// and end-user code.
func (wb *WidgetBase) AddStyler(s Styler) {
	if wb.Stylers == nil {
		wb.Stylers = []Styler{}
	}
	wb.Stylers = append(wb.Stylers, s)
}

// STYTODO: figure out what to do with this
// // AddChildStyler is a helper function that adds the
// // given styler to the child of the given name
// // if it exists, starting searching at the given start index.
// func (wb *WidgetBase) AddChildStyler(childName string, startIdx int, s Styler) {
// 	child := wb.ChildByName(childName, startIdx)
// 	if child != nil {
// 		wb, ok := child.Embed(TypeWidgetBase).(*WidgetBase)
// 		if ok {
// 			wb.AddStyler(func(w *WidgetBase, s *gist.Style) {
// 				f(wb)
// 			})
// 		}
// 	}
// }

// ParentWidget returns the nearest widget parent
// of the widget. It returns nil if no such parent
// is found; see [ParentWidgetTry] for a version with an error.
func (wb *WidgetBase) ParentWidget() *WidgetBase {
	par, _ := wb.ParentWidgetTry()
	return par
}

// ParentWidgetTry returns the nearest widget parent
// of the widget. It returns an error if no such parent
// is found; see [ParentWidget] for a version without an error.
func (wb *WidgetBase) ParentWidgetTry() (*WidgetBase, error) {
	par := wb.ParentByType(TypeWidgetBase, ki.Embeds)
	if par == nil {
		return nil, fmt.Errorf("(*gi.WidgetBase).ParentWidgetTry: widget %v has no parent widget base", wb)
	}
	pwb := par.Embed(TypeWidgetBase).(*WidgetBase)
	return pwb, nil
}

// ParentWidgetIf returns the nearest widget parent
// of the widget for which the given function returns true.
// It returns nil if no such parent is found;
// see [ParentWidgetIfTry] for a version with an error.
func (wb *WidgetBase) ParentWidgetIf(fun func(wb *WidgetBase) bool) *WidgetBase {
	par, _ := wb.ParentWidgetIfTry(fun)
	return par
}

// ParentWidgetIfTry returns the nearest widget parent
// of the widget for which the given function returns true.
// It returns an error if no such parent is found; see
// [ParentWidgetIf] for a version without an error.
func (wb *WidgetBase) ParentWidgetIfTry(fun func(wb *WidgetBase) bool) (*WidgetBase, error) {
	cur := wb
	for {
		par := cur.ParentByType(TypeWidgetBase, ki.Embeds)
		if par == nil {
			return nil, fmt.Errorf("(*gi.WidgetBase).ParentWidgetIfTry: widget %v has no parent widget base", wb)
		}
		pwb := par.Embed(TypeWidgetBase).(*WidgetBase)
		if fun(pwb) {
			return pwb, nil
		}
		cur = pwb
	}
}

// ParentBackgroundColor returns the background color
// of the nearest widget parent of the widget that
// has a defined background color. If no such parent is found,
// it returns a new [gist.ColorSpec] with a solid
// color of [ColorScheme.Background].
func (wb *WidgetBase) ParentBackgroundColor() gist.ColorSpec {
	par := wb.ParentWidgetIf(func(p *WidgetBase) bool {
		return !p.Style.BackgroundColor.IsNil()
	})
	if par == nil {
		cs := gist.ColorSpec{}
		cs.SetColor(ColorScheme.Background)
		return cs
	}
	return par.Style.BackgroundColor
}

// ParentCursor returns the cursor of the nearest
// widget parent of the widget that has a a non-default
// cursor. If no such parent is found, it returns the given
// cursor. This function can be used for elements like labels
// that have a default cursor ([cursor.IBeam]) but should
// not override the cursor of a parent.
func (wb *WidgetBase) ParentCursor(cur cursor.Shapes) cursor.Shapes {
	par := wb.ParentWidgetIf(func(p *WidgetBase) bool {
		return p.Style.Cursor != cursor.Arrow
	})
	if par == nil {
		return cur
	}
	return par.Style.Cursor
}

// ConnectEvents2D is the default event connection function
// for widgets. It calls [WidgetEvents], so any widget
// implementing a custom ConnectEvents2D function should
// first call [WidgetEvents].
func (wb *WidgetBase) ConnectEvents2D() {
	wb.WidgetEvents()
}

// WidgetEvents connects the default events for widgets.
// Any widget implementing a custom ConnectEvents2D function
// should first call this function.
func (wb *WidgetBase) WidgetEvents() {
	wb.Node2DEvents()
	wb.HoverTooltipEvent()
}

// Style helper methods

// SetMinPrefWidth sets minimum and preferred width;
// will get at least this amount; max unspecified.
// This adds a styler that calls [gist.Style.SetMinPrefWidth].
func (wb *WidgetBase) SetMinPrefWidth(val units.Value) {
	wb.AddStyler(func(w *WidgetBase, s *gist.Style) {
		s.SetMinPrefWidth(val)
	})
}

// SetMinPrefHeight sets minimum and preferred height;
// will get at least this amount; max unspecified.
// This adds a styler that calls [gist.Style.SetMinPrefHeight].
func (wb *WidgetBase) SetMinPrefHeight(val units.Value) {
	wb.AddStyler(func(w *WidgetBase, s *gist.Style) {
		s.SetMinPrefHeight(val)
	})
}

// SetStretchMaxWidth sets stretchy max width (-1);
// can grow to take up avail room.
// This adds a styler that calls [gist.Style.SetStretchMaxWidth].
func (wb *WidgetBase) SetStretchMaxWidth() {
	wb.AddStyler(func(w *WidgetBase, s *gist.Style) {
		s.SetStretchMaxWidth()
	})
}

// SetStretchMaxHeight sets stretchy max height (-1);
// can grow to take up avail room.
// This adds a styler that calls [gist.Style.SetStretchMaxHeight].
func (wb *WidgetBase) SetStretchMaxHeight() {
	wb.AddStyler(func(w *WidgetBase, s *gist.Style) {
		s.SetStretchMaxHeight()
	})
}

// SetStretchMax sets stretchy max width and height (-1);
// can grow to take up avail room.
// This adds a styler that calls [gist.Style.SetStretchMax].
func (wb *WidgetBase) SetStretchMax() {
	wb.AddStyler(func(w *WidgetBase, s *gist.Style) {
		s.SetStretchMaxWidth()
		s.SetStretchMaxHeight()
	})
}

// SetFixedWidth sets all width style options
// (Width, MinWidth, and MaxWidth) to
// the given fixed width unit value.
// This adds a styler that calls [gist.Style.SetFixedWidth].
func (wb *WidgetBase) SetFixedWidth(val units.Value) {
	wb.AddStyler(func(w *WidgetBase, s *gist.Style) {
		s.SetFixedWidth(val)
	})
}

// SetFixedHeight sets all height style options
// (Height, MinHeight, and MaxHeight) to
// the given fixed height unit value.
// This adds a styler that calls [gist.Style.SetFixedHeight].
func (wb *WidgetBase) SetFixedHeight(val units.Value) {
	wb.AddStyler(func(w *WidgetBase, s *gist.Style) {
		s.SetFixedHeight(val)
	})
}

// Style2DWidget styles the Style values from node properties and optional
// base-level defaults -- for Widget-style nodes.
// must be called under a StyMu Lock
func (wb *WidgetBase) Style2DWidget() {
	pr := prof.Start("Style2DWidget")
	defer pr.End()

	if wb.OverrideStyle {
		return
	}

	pcsn := prof.Start("Style2DWidget-SetCurStyleNode")

	// STYTODO: there should be a better way to do this
	gii, _ := wb.This().(Node2D)
	wb.Viewport.SetCurStyleNode(gii)
	defer wb.Viewport.SetCurStyleNode(nil)

	pcsn.End()

	wb.Style = gist.Style{}
	wb.Style.Defaults()

	pin := prof.Start("Style2DWidget-Inherit")

	if parSty := wb.ParentActiveStyle(); parSty != nil {
		wb.Style.InheritFields(parSty)
		wb.ParentStyleRUnlock()
	}
	pin.End()

	prun := prof.Start("Style2DWidget-RunStyleFuncs")

	wb.RunStyleFuncs()

	prun.End()

	puc := prof.Start("Style2DWidget-SetUnitContext")

	SetUnitContext(&wb.Style, wb.Viewport, mat32.Vec2{}, mat32.Vec2{})
	puc.End()

	psc := prof.Start("Style2DWidget-SetCurrentColor")

	if wb.Style.Inactive { // inactive can only set, not clear
		wb.SetDisabled()
	}

	wb.Viewport.SetCurrentColor(wb.Style.Color)

	psc.End()
}

// RunStyleFuncs runs the style functions specified in
// the StyleFuncs field in sequential ascending order.
func (wb *WidgetBase) RunStyleFuncs() {
	for _, s := range wb.Stylers {
		s(wb, &wb.Style)
	}
}

func (wb *WidgetBase) Style2D() {
	wb.StyMu.Lock()
	defer wb.StyMu.Unlock()

	wb.Style2DWidget()
	wb.LayState.SetFromStyle(&wb.Style) // also does reset
}

// SetUnitContext sets the unit context based on size of viewport, element, and parent
// element (from bbox) and then caches everything out in terms of raw pixel
// dots for rendering -- call at start of render. Zero values for element and parent size are ignored.
func SetUnitContext(st *gist.Style, vp *Viewport2D, el, par mat32.Vec2) {
	if vp != nil {
		if vp.Win != nil {
			st.UnContext.DPI = vp.Win.LogicalDPI()
		}
		if vp.Render.Image != nil {
			sz := vp.Geom.Size // Render.Image.Bounds().Size()
			st.UnContext.SetSizes(float32(sz.X), float32(sz.Y), el.X, el.Y, par.X, par.Y)
		}
	}
	pr := prof.Start("SetUnitContext-OpenFont")
	st.Font = girl.OpenFont(st.FontRender(), &st.UnContext) // calls SetUnContext after updating metrics
	pr.End()
	ptd := prof.Start("SetUnitContext-ToDots")
	st.ToDots()
	ptd.End()
}

func (wb *WidgetBase) InitLayout2D() bool {
	wb.StyMu.Lock()
	defer wb.StyMu.Unlock()
	wb.LayState.SetFromStyle(&wb.Style)
	return false
}

func (wb *WidgetBase) Size2DBase(iter int) {
	wb.InitLayout2D()
}

func (wb *WidgetBase) Size2D(iter int) {
	wb.Size2DBase(iter)
}

// AddParentPos adds the position of our parent to our layout position --
// layout computations are all relative to parent position, so they are
// finally cached out at this stage also returns the size of the parent for
// setting units context relative to parent objects
func (wb *WidgetBase) AddParentPos() mat32.Vec2 {
	if pni, _ := KiToNode2D(wb.Par); pni != nil {
		if pw := pni.AsWidget(); pw != nil {
			if !wb.IsField() {
				wb.LayState.Alloc.Pos = pw.LayState.Alloc.PosOrig.Add(wb.LayState.Alloc.PosRel)
			}
			return pw.LayState.Alloc.Size
		}
	}
	return mat32.Vec2Zero
}

// BBoxFromAlloc gets our bbox from Layout allocation.
func (wb *WidgetBase) BBoxFromAlloc() image.Rectangle {
	return mat32.RectFromPosSizeMax(wb.LayState.Alloc.Pos, wb.LayState.Alloc.Size)
}

func (wb *WidgetBase) BBox2D() image.Rectangle {
	return wb.BBoxFromAlloc()
}

func (wb *WidgetBase) ComputeBBox2D(parBBox image.Rectangle, delta image.Point) {
	wb.ComputeBBox2DBase(parBBox, delta)
}

// Layout2DBase provides basic Layout2D functions -- good for most cases
func (wb *WidgetBase) Layout2DBase(parBBox image.Rectangle, initStyle bool, iter int) {
	nii, _ := wb.This().(Node2D)
	mvp := wb.ViewportSafe()
	if mvp == nil { // robust
		if nii.AsViewport2D() == nil {
			// todo: not so clear that this will do anything useful at this point
			// but at least it gets the viewport
			nii.Init2D()
			nii.Style2D()
			nii.Size2D(0)
			// fmt.Printf("node not init in Layout2DBase: %v\n", wb.Path())
		}
	}
	psize := wb.AddParentPos()
	wb.LayState.Alloc.PosOrig = wb.LayState.Alloc.Pos
	if initStyle {
		mvp := wb.ViewportSafe()
		SetUnitContext(&wb.Style, mvp, wb.NodeSize(), psize) // update units with final layout
	}
	wb.BBox = nii.BBox2D() // only compute once, at this point
	// note: if other styles are maintained, they also need to be updated!
	nii.ComputeBBox2D(parBBox, image.Point{}) // other bboxes from BBox
	if Layout2DTrace {
		fmt.Printf("Layout: %v alloc pos: %v size: %v vpbb: %v winbb: %v\n", wb.Path(), wb.LayState.Alloc.Pos, wb.LayState.Alloc.Size, wb.VpBBox, wb.WinBBox)
	}
	// typically Layout2DChildren must be called after this!
}

func (wb *WidgetBase) Layout2D(parBBox image.Rectangle, iter int) bool {
	wb.Layout2DBase(parBBox, true, iter)
	return wb.Layout2DChildren(iter)
}

// ChildrenBBox2DWidget provides a basic widget box-model subtraction of
// margin and padding to children -- call in ChildrenBBox2D for most widgets
func (wb *WidgetBase) ChildrenBBox2DWidget() image.Rectangle {
	nb := wb.VpBBox
	spc := wb.BoxSpace()
	nb.Min.X += int(spc.Left)
	nb.Min.Y += int(spc.Top)
	nb.Max.X -= int(spc.Right)
	nb.Max.Y -= int(spc.Bottom)
	return nb
}

func (wb *WidgetBase) ChildrenBBox2D() image.Rectangle {
	return wb.ChildrenBBox2DWidget()
}

// FullReRenderIfNeeded tests if the FullReRender flag has been set, and if
// so, calls ReRender2DTree and returns true -- call this at start of each
// Render2D
func (wb *WidgetBase) FullReRenderIfNeeded() bool {
	mvp := wb.ViewportSafe()
	if wb.This().(Node2D).IsVisible() && wb.NeedsFullReRender() && !mvp.IsDoingFullRender() {
		if Render2DTrace {
			fmt.Printf("Render: NeedsFullReRender for %v at %v\n", wb.Path(), wb.VpBBox)
		}
		// if ki.TypeEmbeds(wb.This(), TypeFrame) || strings.Contains(ki.Type(wb.This()).String(), "TextView") {
		// 	fmt.Printf("Render: NeedsFullReRender for %v at %v\n", wb.Path(), wb.VpBBox)
		// }
		wb.ClearFullReRender()
		wb.ReRender2DTree()
		return true
	}
	return false
}

// PushBounds pushes our bounding-box bounds onto the bounds stack if non-empty
// -- this limits our drawing to our own bounding box, automatically -- must
// be called as first step in Render2D returns whether the new bounds are
// empty or not -- if empty then don't render!
func (wb *WidgetBase) PushBounds() bool {
	if wb == nil || wb.This() == nil {
		return false
	}
	if !wb.This().(Node2D).IsVisible() {
		return false
	}
	if wb.VpBBox.Empty() {
		wb.ClearFullReRender()
		return false
	}
	mvp := wb.ViewportSafe()
	rs := &mvp.Render
	rs.PushBounds(wb.VpBBox)
	wb.ConnectToViewport()
	if Render2DTrace {
		fmt.Printf("Render: %v at %v\n", wb.Path(), wb.VpBBox)
	}
	return true
}

// PopBounds pops our bounding-box bounds -- last step in Render2D after
// rendering children
func (wb *WidgetBase) PopBounds() {
	wb.ClearFullReRender()
	if wb.IsDeleted() || wb.IsDestroyed() || wb.This() == nil {
		return
	}
	mvp := wb.ViewportSafe()
	if mvp == nil {
		return
	}
	rs := &mvp.Render
	rs.PopBounds()
}

func (wb *WidgetBase) Render2D() {
	if wb.FullReRenderIfNeeded() {
		return
	}
	if wb.PushBounds() {
		wb.This().(Node2D).ConnectEvents2D()
		if wb.NeedsStyle() {
			wb.This().(Node2D).Style2D()
			wb.ClearNeedsStyle()
		}
		wb.Render2DChildren()
		wb.PopBounds()
	} else {
		wb.DisconnectAllEvents(RegPri)
	}
}

// ReRender2DTree does a re-render of the tree -- after it has already been
// initialized and styled -- redoes the full stack
func (wb *WidgetBase) ReRender2DTree() {
	parBBox := image.Rectangle{}
	pni, _ := KiToNode2D(wb.Par)
	if pni != nil {
		parBBox = pni.ChildrenBBox2D()
	}
	delta := wb.LayState.Alloc.Pos.Sub(wb.LayState.Alloc.PosOrig)
	wb.LayState.Alloc.Pos = wb.LayState.Alloc.PosOrig
	ld := wb.LayState // save our current layout data
	updt := wb.UpdateStart()
	wb.Init2DTree()
	wb.Style2DTree()
	wb.Size2DTree(0)
	wb.LayState = ld // restore
	wb.Layout2DTree()
	if !delta.IsNil() {
		wb.Move2D(delta.ToPointFloor(), parBBox)
	}
	wb.Render2DTree()
	wb.UpdateEndNoSig(updt)
}

// Move2DBase does the basic move on this node
func (wb *WidgetBase) Move2DBase(delta image.Point, parBBox image.Rectangle) {
	wb.LayState.Alloc.Pos = wb.LayState.Alloc.PosOrig.Add(mat32.NewVec2FmPoint(delta))
	wb.This().(Node2D).ComputeBBox2D(parBBox, delta)
}

func (wb *WidgetBase) Move2D(delta image.Point, parBBox image.Rectangle) {
	wb.Move2DBase(delta, parBBox)
	wb.Move2DChildren(delta)
}

// Move2DTree does move2d pass -- each node iterates over children for maximum
// control -- this starts with parent ChildrenBBox and current delta -- can be
// called de novo
func (wb *WidgetBase) Move2DTree() {
	if wb.HasNoLayout() {
		return
	}
	parBBox := image.Rectangle{}
	pnii, pn := KiToNode2D(wb.Par)
	if pn != nil {
		parBBox = pnii.ChildrenBBox2D()
	}
	delta := wb.LayState.Alloc.Pos.Sub(wb.LayState.Alloc.PosOrig).ToPoint()
	wb.This().(Node2D).Move2D(delta, parBBox) // important to use interface version to get interface!
}

// ParentLayout returns the parent layout
func (wb *WidgetBase) ParentLayout() *Layout {
	var parLy *Layout
	wb.FuncUpParent(0, wb.This(), func(k ki.Ki, level int, d any) bool {
		nii, ok := k.(Node2D)
		if !ok {
			return ki.Break // don't keep going up
		}
		ly := nii.AsLayout2D()
		if ly != nil {
			parLy = ly
			return ki.Break // done
		}
		return ki.Continue
	})
	return parLy
}

// CtxtMenuFunc is a function for creating a context menu for given node
type CtxtMenuFunc func(g Node2D, m *Menu)

func (wb *WidgetBase) MakeContextMenu(m *Menu) {
	// derived types put native menu code here
	if wb.CtxtMenuFunc != nil {
		wb.CtxtMenuFunc(wb.This().(Node2D), m)
	}
	mvp := wb.ViewportSafe()
	TheViewIFace.CtxtMenuView(wb.This(), wb.IsDisabled(), mvp, m)
}

// TooltipConfigStyles configures the default styles
// for the given tooltip frame with the given parent.
// It should be called on tooltips when they are created.
func TooltipConfigStyles(par *WidgetBase, tooltip *Frame) {
	tooltip.AddStyler(func(w *WidgetBase, s *gist.Style) {
		s.Border.Style.Set(gist.BorderNone)
		s.Border.Radius = gist.BorderRadiusExtraSmall
		s.Padding.Set(units.Px(8 * Prefs.DensityMul()))
		s.BackgroundColor.SetSolid(ColorScheme.InverseSurface)
		s.Color = ColorScheme.InverseOnSurface
		s.BoxShadow = BoxShadow1 // STYTODO: not sure whether we should have this
	})
}

// PopupTooltip pops up a viewport displaying the tooltip text
func PopupTooltip(tooltip string, x, y int, parVp *Viewport2D, name string) *Viewport2D {
	win := parVp.Win
	mainVp := win.Viewport
	pvp := Viewport2D{}
	pvp.InitName(&pvp, name+"Tooltip")
	pvp.Win = win
	updt := pvp.UpdateStart()
	pvp.Fill = true
	pvp.SetFlag(int(VpFlagPopup))
	pvp.SetFlag(int(VpFlagTooltip))
	pvp.AddStyler(func(w *WidgetBase, s *gist.Style) {
		// TOOD: get border radius actually working
		// without having parent background color workaround

		s.Border.Radius = gist.BorderRadiusExtraSmall
		s.BackgroundColor = pvp.ParentBackgroundColor()
	})

	pvp.Geom.Pos = image.Point{x, y}
	pvp.SetFlag(int(VpFlagPopupDestroyAll)) // nuke it all
	frame := AddNewFrame(&pvp, "Frame", LayoutVert)
	lbl := AddNewLabel(frame, "ttlbl", tooltip)
	lbl.Type = LabelBodyMedium

	TooltipConfigStyles(&pvp.WidgetBase, frame)

	lbl.AddStyler(func(w *WidgetBase, s *gist.Style) {
		mwdots := parVp.Style.UnContext.ToDots(40, units.UnitEm)
		mwdots = mat32.Min(mwdots, float32(mainVp.Geom.Size.X-20))

		s.MaxWidth.SetDot(mwdots)
	})

	frame.Init2DTree()
	frame.Style2DTree()                                    // sufficient to get sizes
	frame.LayState.Alloc.Size = mainVp.LayState.Alloc.Size // give it the whole vp initially
	frame.Size2DTree(0)                                    // collect sizes
	pvp.Win = nil
	vpsz := frame.LayState.Size.Pref.Min(mainVp.LayState.Alloc.Size).ToPoint()

	x = ints.MinInt(x, mainVp.Geom.Size.X-vpsz.X) // fit
	y = ints.MinInt(y, mainVp.Geom.Size.Y-vpsz.Y) // fit
	pvp.Resize(vpsz)
	pvp.Geom.Pos = image.Point{x, y}
	pvp.UpdateEndNoSig(updt)

	win.PushPopup(pvp.This())
	return &pvp
}

// WidgetSignals are general signals that all widgets can send, via WidgetSig
// signal
type WidgetSignals int64

const (
	// WidgetSelected is triggered when a widget is selected, typically via
	// left mouse button click (see EmitSelectedSignal) -- is NOT contingent
	// on actual IsSelected status -- just reports the click event.
	// The data is the index of the selected item for multi-item widgets
	// (-1 = none / unselected)
	WidgetSelected WidgetSignals = iota

	// WidgetFocused is triggered when a widget receives keyboard focus (see
	// EmitFocusedSignal -- call in FocusChanged2D for gotFocus
	WidgetFocused

	// WidgetContextMenu is triggered when a widget receives a
	// right-mouse-button press, BEFORE generating and displaying the context
	// menu, so that relevant state can be updated etc (see
	// EmitContextMenuSignal)
	WidgetContextMenu

	WidgetSignalsN
)

// EmitSelectedSignal emits the WidgetSelected signal for this widget
func (wb *WidgetBase) EmitSelectedSignal() {
	wb.WidgetSig.Emit(wb.This(), int64(WidgetSelected), nil)
}

// EmitFocusedSignal emits the WidgetFocused signal for this widget
func (wb *WidgetBase) EmitFocusedSignal() {
	wb.WidgetSig.Emit(wb.This(), int64(WidgetFocused), nil)
}

// EmitContextMenuSignal emits the WidgetContextMenu signal for this widget
func (wb *WidgetBase) EmitContextMenuSignal() {
	wb.WidgetSig.Emit(wb.This(), int64(WidgetContextMenu), nil)
}

// HoverTooltipEvent connects to HoverEvent and pops up a tooltip -- most
// widgets should call this as part of their event connection method
func (wb *WidgetBase) HoverTooltipEvent() {
	wb.ConnectEvent(oswin.MouseHoverEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.HoverEvent)
		wbb := recv.Embed(TypeWidgetBase).(*WidgetBase)
		if wbb.Tooltip != "" {
			me.SetProcessed()
			pos := wbb.WinBBox.Max
			pos.X -= 20
			mvp := wbb.ViewportSafe()
			PopupTooltip(wbb.Tooltip, pos.X, pos.Y, mvp, wbb.Nm)
		}
	})
}

// WidgetMouseEvents connects to either or both mouse events -- IMPORTANT: if
// you need to also connect to other mouse events, you must copy this code --
// all processing of a mouse event must happen within one function b/c there
// can only be one registered per receiver and event type.  sel = Left button
// mouse.Press event, toggles the selected state, and emits a SelectedEvent.
// ctxtMenu = connects to Right button mouse.Press event, and sends a
// WidgetSig WidgetContextMenu signal, followed by calling ContextMenu method
// -- signal can be used to change state prior to generating context menu,
// including setting a CtxtMenuFunc that removes all items and thus negates
// the presentation of any menu
func (wb *WidgetBase) WidgetMouseEvents(sel, ctxtMenu bool) {
	if !sel && !ctxtMenu {
		return
	}
	wb.ConnectEvent(oswin.MouseEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.Event)
		if sel {
			if me.Action == mouse.Press && me.Button == mouse.Left {
				me.SetProcessed()
				wbb := recv.Embed(TypeWidgetBase).(*WidgetBase)
				wbb.SetSelectedState(!wbb.IsSelected())
				wbb.EmitSelectedSignal()
				wbb.UpdateSig()
			}
		}
		if ctxtMenu {
			if me.Action == mouse.Release && me.Button == mouse.Right {
				me.SetProcessed()
				wbb := recv.Embed(TypeWidgetBase).(*WidgetBase)
				wbb.EmitContextMenuSignal()
				wbb.This().(Node2D).ContextMenu()
			}
		}
	})
}

////////////////////////////////////////////////////////////////////////////////
//  Standard rendering

// RenderLock returns the locked girl.State, Paint, and Style with StyMu locked.
// This should be called at start of widget-level rendering.
func (wb *WidgetBase) RenderLock() (*girl.State, *girl.Paint, *gist.Style) {
	wb.StyMu.RLock()
	rs := &wb.Viewport.Render
	rs.Lock()
	return rs, &rs.Paint, &wb.Style
}

// RenderUnlock unlocks girl.State and style
func (wb *WidgetBase) RenderUnlock(rs *girl.State) {
	rs.Unlock()
	wb.StyMu.RUnlock()
}

// RenderBoxImpl implements the standard box model rendering -- assumes all
// paint params have already been set
func (wb *WidgetBase) RenderBoxImpl(pos mat32.Vec2, sz mat32.Vec2, bs gist.Border) {
	rs := &wb.Viewport.Render
	pc := &rs.Paint
	pc.DrawBorder(rs, pos.X, pos.Y, sz.X, sz.Y, bs)
}

// RenderStdBox draws standard box using given style.
// girl.State and Style must already be locked at this point (RenderLock)
func (wb *WidgetBase) RenderStdBox(st *gist.Style) {
	// SidesTODO: this is a pretty critical function, so a good place to look if things aren't working
	wb.StyMu.RLock()
	defer wb.StyMu.RUnlock()

	rs := &wb.Viewport.Render
	pc := &rs.Paint

	// TODO: maybe implement some version of this to render background color
	// in margin if the parent element doesn't render for us
	// if pwb, ok := wb.Parent().(*WidgetBase); ok {
	// 	if pwb.Embed(TypeLayout) != nil && pwb.Embed(TypeFrame) == nil {
	// 		pc.FillBox(rs, wb.LayState.Alloc.Pos, wb.LayState.Alloc.Size, &st.BackgroundColor)
	// 	}
	// }

	pos := wb.LayState.Alloc.Pos.Add(st.EffMargin().Pos())
	sz := wb.LayState.Alloc.Size.Sub(st.EffMargin().Size())
	rad := st.Border.Radius.Dots()

	// the background color we actually use
	bg := st.BackgroundColor
	// the surrounding background color
	sbg := wb.ParentBackgroundColor()
	if bg.IsNil() {
		// we need to do this to prevent
		// elements from rendering over themselves
		// (see https://goki.dev/gi/v2/issues/565)
		bg = sbg
	}

	// We need to fill the whole box where the
	// box shadows / element can go to prevent growing
	// box shadows and borders. We couldn't just
	// do this when there are box shadows, as they
	// may be removed and then need to be covered up.
	// This also fixes https://goki.dev/gi/v2/issues/579.
	// This isn't an ideal solution because of performance,
	// so TODO: maybe come up with a better solution for this.
	// We need to use raw LayState data because we need to clear
	// any box shadow that may have gone in margin.
	mspos, mssz := st.BoxShadowPosSize(wb.LayState.Alloc.Pos, wb.LayState.Alloc.Size)
	pc.FillBox(rs, mspos, mssz, &sbg)

	// first do any shadow
	if st.HasBoxShadow() {
		for _, shadow := range st.BoxShadow {
			pc.StrokeStyle.SetColor(nil)
			pc.FillStyle.SetColor(shadow.Color)

			// TODO: better handling of opacity?
			prevOpacity := pc.FillStyle.Opacity
			pc.FillStyle.Opacity = float32(shadow.Color.A) / 255
			// we only want radius for border, no actual border
			wb.RenderBoxImpl(shadow.BasePos(pos), shadow.BaseSize(sz), gist.Border{Radius: st.Border.Radius})
			// pc.FillStyle.Opacity = 1.0
			if shadow.Blur.Dots != 0 {
				// must divide by 2 like CSS
				pc.BlurBox(rs, shadow.Pos(pos), shadow.Size(sz), shadow.Blur.Dots/2)
			}
			pc.FillStyle.Opacity = prevOpacity
		}
	}

	// then draw the box over top of that.
	// need to set clipping to box first.. (?)
	// we need to draw things twice here because we need to clear
	// the whole area with the background color first so the border
	// doesn't render weirdly
	if rad.IsZero() {
		pc.FillBox(rs, pos, sz, &bg)
	} else {
		pc.FillStyle.SetColorSpec(&bg)
		// no border -- fill only
		pc.DrawRoundedRectangle(rs, pos.X, pos.Y, sz.X, sz.Y, rad)
		pc.Fill(rs)
	}

	// pc.StrokeStyle.SetColor(&st.Border.Color)
	// pc.StrokeStyle.Width = st.Border.Width
	// pc.FillStyle.SetColorSpec(&st.BackgroundColor)
	pos.SetAdd(st.Border.Width.Dots().Pos().MulScalar(0.5))
	sz.SetSub(st.Border.Width.Dots().Size().MulScalar(0.5))
	pc.FillStyle.SetColor(nil)
	// now that we have drawn background color
	// above, we can draw the border
	wb.RenderBoxImpl(pos, sz, st.Border)
}

// set our LayState.Alloc.Size from constraints
func (wb *WidgetBase) Size2DFromWH(w, h float32) {
	st := &wb.Style
	if st.Width.Dots > 0 {
		w = mat32.Max(st.Width.Dots, w)
	}
	if st.Height.Dots > 0 {
		h = mat32.Max(st.Height.Dots, h)
	}
	spcsz := st.BoxSpace().Size()
	w += spcsz.X
	h += spcsz.Y
	wb.LayState.Alloc.Size = mat32.Vec2{w, h}
}

// Size2DAddSpace adds space to existing AllocSize
func (wb *WidgetBase) Size2DAddSpace() {
	spc := wb.BoxSpace()
	wb.LayState.Alloc.Size.SetAdd(spc.Size())
}

// Size2DSubSpace returns AllocSize minus 2 * BoxSpace -- the amount avail to the internal elements
func (wb *WidgetBase) Size2DSubSpace() mat32.Vec2 {
	spc := wb.BoxSpace()
	return wb.LayState.Alloc.Size.Sub(spc.Size())
}

///////////////////////////////////////////////////////////////////
// PartsWidgetBase

// PartsWidgetBase is the base type for all Widget Node2D elements that manage
// a set of constituent parts
type PartsWidgetBase struct {
	WidgetBase

	// a separate tree of sub-widgets that implement discrete parts of a widget -- positions are always relative to the parent widget -- fully managed by the widget and not saved
	Parts Layout `json:"-" xml:"-" view-closed:"true" desc:"a separate tree of sub-widgets that implement discrete parts of a widget -- positions are always relative to the parent widget -- fully managed by the widget and not saved"`
}

var TypePartsWidgetBase = kit.Types.AddType(&PartsWidgetBase{}, PartsWidgetBaseProps)

var PartsWidgetBaseProps = ki.Props{
	"base-type":     true,
	ki.EnumTypeFlag: TypeNodeFlags,
}

func (wb *PartsWidgetBase) CopyFieldsFrom(frm any) {
	fr, ok := frm.(*PartsWidgetBase)
	if !ok {
		log.Printf("GoGi node of type: %v needs a CopyFieldsFrom method defined -- currently falling back on earlier PartsWidgetBase one\n", wb.This().Name())
		ki.GenCopyFieldsFrom(wb.This(), frm)
		return
	}
	wb.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	// wb.Parts.CopyFrom(&fr.Parts) // managed by widget -- we don't copy..
}

// standard FunDownMeFirst etc operate automatically on Field structs such as
// Parts -- custom calls only needed for manually-recursive traversal in
// Layout and Render

// SizeFromParts sets our size from those of our parts -- default..
func (wb *PartsWidgetBase) SizeFromParts(iter int) {
	wb.LayState.Alloc.Size = wb.Parts.LayState.Size.Pref // get from parts
	wb.Size2DAddSpace()
	if Layout2DTrace {
		fmt.Printf("Size:   %v size from parts: %v, parts pref: %v\n", wb.Path(), wb.LayState.Alloc.Size, wb.Parts.LayState.Size.Pref)
	}
}

func (wb *PartsWidgetBase) Size2DParts(iter int) {
	wb.InitLayout2D()
	wb.SizeFromParts(iter) // get our size from parts
}

func (wb *PartsWidgetBase) Size2D(iter int) {
	wb.Size2DParts(iter)
}

func (wb *PartsWidgetBase) ComputeBBox2DParts(parBBox image.Rectangle, delta image.Point) {
	wb.ComputeBBox2DBase(parBBox, delta)
	wb.Parts.This().(Node2D).ComputeBBox2D(parBBox, delta)
}

func (wb *PartsWidgetBase) ComputeBBox2D(parBBox image.Rectangle, delta image.Point) {
	wb.ComputeBBox2DParts(parBBox, delta)
}

func (wb *PartsWidgetBase) Layout2DParts(parBBox image.Rectangle, iter int) {
	spc := wb.BoxSpace()
	wb.Parts.LayState.Alloc.Pos = wb.LayState.Alloc.Pos.Add(spc.Pos())
	wb.Parts.LayState.Alloc.Size = wb.LayState.Alloc.Size.Sub(spc.Size())
	wb.Parts.Layout2D(parBBox, iter)
}

func (wb *PartsWidgetBase) Layout2D(parBBox image.Rectangle, iter int) bool {
	wb.Layout2DBase(parBBox, true, iter) // init style
	wb.Layout2DParts(parBBox, iter)
	return wb.Layout2DChildren(iter)
}

func (wb *PartsWidgetBase) Render2DParts() {
	wb.Parts.Render2DTree()
}

func (wb *PartsWidgetBase) Move2D(delta image.Point, parBBox image.Rectangle) {
	wb.Move2DBase(delta, parBBox)
	wb.Parts.This().(Node2D).Move2D(delta, parBBox)
	wb.Move2DChildren(delta)
}

///////////////////////////////////////////////////////////////////
// ConfigParts building-blocks

// ConfigPartsIconLabel adds to config to create parts, of icon
// and label left-to right in a row, based on whether items are nil or empty
func (wb *PartsWidgetBase) ConfigPartsIconLabel(config *kit.TypeAndNameList, icnm icons.Icon, txt string) (icIdx, lbIdx int) {
	if wb.Style.Template != "" {
		wb.Parts.Style.Template = wb.Style.Template + ".Parts"
	}
	icIdx = -1
	lbIdx = -1
	if TheIconMgr.IsValid(icnm) {
		icIdx = len(*config)
		config.Add(TypeIcon, "icon")
		if txt != "" {
			config.Add(TypeSpace, "space")
		}
	}
	if txt != "" {
		lbIdx = len(*config)
		config.Add(TypeLabel, "label")
	}
	return
}

// ConfigPartsSetIconLabel sets the icon and text values in parts, and get
// part style props, using given props if not set in object props
func (wb *PartsWidgetBase) ConfigPartsSetIconLabel(icnm icons.Icon, txt string, icIdx, lbIdx int) {
	if icIdx >= 0 {
		ic := wb.Parts.Child(icIdx).(*Icon)
		if wb.Style.Template != "" {
			ic.Style.Template = wb.Style.Template + ".icon"
		}
		ic.SetIcon(icnm)
	}
	if lbIdx >= 0 {
		lbl := wb.Parts.Child(lbIdx).(*Label)
		if wb.Style.Template != "" {
			lbl.Style.Template = wb.Style.Template + ".icon"
		}
		if lbl.Text != txt {
			// avoiding SetText here makes it so label default
			// styles don't end up first, which is needed for
			// parent styles to override. However, there might have
			// been a reason for calling SetText, so we will see if
			// any bugs show up. TODO: figure out a good long-term solution for this.
			lbl.Text = txt
			// lbl.SetText(txt)
		}
	}
}

// PartsNeedUpdateIconLabel check if parts need to be updated -- for ConfigPartsIfNeeded
func (wb *PartsWidgetBase) PartsNeedUpdateIconLabel(icnm icons.Icon, txt string) bool {
	if TheIconMgr.IsValid(icnm) {
		ick := wb.Parts.ChildByName("icon", 0)
		if ick == nil {
			return true
		}
		ic := ick.(*Icon)
		if !ic.HasChildren() || ic.IconNm != icnm || wb.NeedsFullReRender() {
			return true
		}
	} else {
		cn := wb.Parts.ChildByName("icon", 0)
		if cn != nil { // need to remove it
			return true
		}
	}
	if txt != "" {
		lblk := wb.Parts.ChildByName("label", 2)
		if lblk == nil {
			return true
		}
		lbl := lblk.(*Label)
		lbl.Style.Color = wb.Style.Color
		if lbl.Text != txt {
			return true
		}
	} else {
		cn := wb.Parts.ChildByName("label", 2)
		if cn != nil {
			return true
		}
	}
	return false
}

// SetFullReRenderIconLabel sets the icon and label to be re-rendered, needed
// when styles change
func (wb *PartsWidgetBase) SetFullReRenderIconLabel() {
	if ick := wb.Parts.ChildByName("icon", 0); ick != nil {
		ic := ick.(*Icon)
		ic.SetFullReRender()
	}
	if lblk := wb.Parts.ChildByName("label", 2); lblk != nil {
		lbl := lblk.(*Label)
		lbl.SetFullReRender()
	}
	wb.Parts.StyMu.Lock()
	wb.Parts.Style2DWidget() // restyle parent so parts inherit
	wb.Parts.StyMu.Unlock()
}
