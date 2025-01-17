// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"log"
	"reflect"
	"sync"

	"goki.dev/colors"
	"goki.dev/gi/v2/gist"
	"goki.dev/gi/v2/icons"
	"goki.dev/gi/v2/oswin/cursor"
	"goki.dev/gi/v2/units"
	"goki.dev/ki/v2/ki"
	"goki.dev/ki/v2/kit"
)

// TabView switches among child widgets via tabs.  The selected widget gets
// the full allocated space avail after the tabs are accounted for.  The
// TabView is just a Vertical layout that manages two child widgets: a
// HorizFlow Layout for the tabs (which can flow across multiple rows as
// needed) and a Stacked Frame that actually contains all the children, and
// provides scrollbars as needed to any content within.  Typically should have
// max stretch and a set preferred size, so it expands.
type TabView struct {
	Layout

	// maximum number of characters to include in tab label -- elides labels that are longer than that
	MaxChars int `desc:"maximum number of characters to include in tab label -- elides labels that are longer than that"`

	// signal for tab widget -- see TabViewSignals for the types
	TabViewSig ki.Signal `copy:"-" json:"-" xml:"-" desc:"signal for tab widget -- see TabViewSignals for the types"`

	// show a new tab button at right of list of tabs
	NewTabButton bool `desc:"show a new tab button at right of list of tabs"`

	// if true, tabs are not user-deleteable
	NoDeleteTabs bool `desc:"if true, tabs are not user-deleteable"`

	// type of widget to create in a new tab via new tab button -- Frame by default
	NewTabType reflect.Type `desc:"type of widget to create in a new tab via new tab button -- Frame by default"`

	// [view: -] mutex protecting updates to tabs -- tabs can be driven programmatically and via user input so need extra protection
	Mu sync.Mutex `copy:"-" json:"-" xml:"-" view:"-" desc:"mutex protecting updates to tabs -- tabs can be driven programmatically and via user input so need extra protection"`
}

var TypeTabView = kit.Types.AddType(&TabView{}, TabViewProps)

// AddNewTabView adds a new tabview to given parent node, with given name.
func AddNewTabView(parent ki.Ki, name string) *TabView {
	return parent.AddNewChild(TypeTabView, name).(*TabView)
}

func (tv *TabView) OnInit() {
	tv.AddStyler(func(w *WidgetBase, s *gist.Style) {
		// need border for separators (see RenderTabSeps)
		// TODO: maybe better solution for tab sep styles?
		s.Border.Style.Set(gist.BorderSolid)
		s.Border.Width.Set(units.Px(1))
		s.Border.Color.Set(ColorScheme.OutlineVariant)
		s.BackgroundColor.SetSolid(ColorScheme.Background)
		s.Color = ColorScheme.OnBackground
		s.MaxWidth.SetPx(-1)
		s.MaxHeight.SetPx(-1)
	})
}

func (tv *TabView) OnChildAdded(child ki.Ki) {
	if w := KiAsWidget(child); w != nil {
		switch w.Name() {
		case "tabs":
			w.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.SetStretchMaxWidth()
				s.Height.SetEm(1.8)
				s.Overflow = gist.OverflowHidden // no scrollbars!
				s.Margin.Set()
				s.Padding.Set()
				// tabs.Spacing.SetPx(4 * Prefs.DensityMul())
				s.BackgroundColor.SetSolid(ColorScheme.SurfaceContainerLow)

				// s.Border.Style.Set(gist.BorderNone)
				// s.Border.Style.Bottom = gist.BorderSolid
				// s.Border.Width.Bottom.SetPx(1)
				// s.Border.Color.Bottom = ColorScheme.OutlineVariant
			})
		case "frame":
			frame := child.(*Frame)
			frame.StackTopOnly = true // key for allowing each tab to have its own size
			w.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.SetMinPrefWidth(units.Em(10))
				s.SetMinPrefHeight(units.Em(6))
				s.SetStretchMax()
			})
		}
	}
}

func (tv *TabView) CopyFieldsFrom(frm any) {
	fr := frm.(*TabView)
	tv.Layout.CopyFieldsFrom(&fr.Layout)
	tv.MaxChars = fr.MaxChars
	tv.NewTabButton = fr.NewTabButton
	tv.NewTabType = fr.NewTabType
}

func (tv *TabView) Disconnect() {
	tv.Layout.Disconnect()
	tv.TabViewSig.DisconnectAll()
}

var TabViewProps = ki.Props{
	ki.EnumTypeFlag: TypeNodeFlags,
}

// NTabs returns number of tabs
func (tv *TabView) NTabs() int {
	fr := tv.Frame()
	if fr == nil {
		return 0
	}
	return len(fr.Kids)
}

// CurTab returns currently-selected tab, and its index -- returns false none
func (tv *TabView) CurTab() (Node2D, int, bool) {
	if tv.NTabs() == 0 {
		return nil, -1, false
	}
	tv.Mu.Lock()
	defer tv.Mu.Unlock()
	fr := tv.Frame()
	if fr.StackTop < 0 {
		return nil, -1, false
	}
	widg := fr.Child(fr.StackTop).(Node2D)
	return widg, fr.StackTop, true
}

// AddTab adds a widget as a new tab, with given tab label, and returns the
// index of that tab
func (tv *TabView) AddTab(widg Node2D, label string) int {
	fr := tv.Frame()
	idx := len(*fr.Children())
	tv.InsertTab(widg, label, idx)
	return idx
}

// InsertTabOnlyAt inserts just the tab at given index -- after panel has
// already been added to frame -- assumed to be wrapped in update.  Generally
// for internal use.
func (tv *TabView) InsertTabOnlyAt(widg Node2D, label string, idx int) {
	tb := tv.Tabs()
	tb.SetChildAdded()
	tab := tb.InsertNewChild(TypeTabButton, idx, label).(*TabButton)
	tab.Data = idx
	tab.Tooltip = label
	tab.NoDelete = tv.NoDeleteTabs
	tab.SetText(label)
	tab.ActionSig.ConnectOnly(tv.This(), func(recv, send ki.Ki, sig int64, data any) {
		tvv := recv.Embed(TypeTabView).(*TabView)
		act := send.Embed(TypeTabButton).(*TabButton)
		tabIdx := act.Data.(int)
		tvv.SelectTabIndexAction(tabIdx)
	})
	fr := tv.Frame()
	if len(fr.Kids) == 1 {
		fr.StackTop = 0
		tab.SetSelectedState(true)
	} else {
		widg.AsNode2D().SetInvisible() // new tab is invisible until selected
	}
}

// InsertTab inserts a widget into given index position within list of tabs
func (tv *TabView) InsertTab(widg Node2D, label string, idx int) {
	tv.Mu.Lock()
	fr := tv.Frame()
	updt := tv.UpdateStart()
	tv.SetFullReRender()
	fr.SetChildAdded()
	fr.InsertChild(widg, idx)
	tv.InsertTabOnlyAt(widg, label, idx)
	tv.Mu.Unlock()
	tv.UpdateEnd(updt)
}

// AddNewTab adds a new widget as a new tab of given widget type, with given
// tab label, and returns the new widget
func (tv *TabView) AddNewTab(typ reflect.Type, label string) Node2D {
	fr := tv.Frame()
	idx := len(*fr.Children())
	widg := tv.InsertNewTab(typ, label, idx)
	return widg
}

// AddNewTabLayout adds a new widget as a new tab of given widget type,
// with given tab label, and returns the new widget.
// A Layout is added first and the widget is added to that layout.
// The Layout has "-lay" suffix added to name.
func (tv *TabView) AddNewTabLayout(typ reflect.Type, label string) (Node2D, *Layout) {
	ly := tv.AddNewTab(TypeLayout, label).(*Layout)
	ly.SetName(label + "-lay")
	widg := ly.AddNewChild(typ, label).(Node2D)
	return widg, ly
}

// AddNewTabFrame adds a new widget as a new tab of given widget type,
// with given tab label, and returns the new widget.
// A Frame is added first and the widget is added to that Frame.
// The Frame has "-frame" suffix added to name.
func (tv *TabView) AddNewTabFrame(typ reflect.Type, label string) (Node2D, *Frame) {
	fr := tv.AddNewTab(TypeFrame, label).(*Frame)
	fr.SetName(label + "-frame")
	widg := fr.AddNewChild(typ, label).(Node2D)
	return widg, fr
}

// AddNewTabAction adds a new widget as a new tab of given widget type, with given
// tab label, and returns the new widget -- emits TabAdded signal
func (tv *TabView) AddNewTabAction(typ reflect.Type, label string) Node2D {
	widg := tv.AddNewTab(typ, label)
	fr := tv.Frame()
	idx := len(*fr.Children()) - 1
	tv.TabViewSig.Emit(tv.This(), int64(TabAdded), idx)
	return widg
}

// InsertNewTab inserts a new widget of given type into given index position
// within list of tabs, and returns that new widget
func (tv *TabView) InsertNewTab(typ reflect.Type, label string, idx int) Node2D {
	fr := tv.Frame()
	updt := tv.UpdateStart()
	tv.SetFullReRender()
	fr.SetChildAdded()
	widg := fr.InsertNewChild(typ, idx, label).(Node2D)
	tv.InsertTabOnlyAt(widg, label, idx)
	tv.UpdateEnd(updt)
	return widg
}

// TabAtIndex returns content widget and tab button at given index, false if
// index out of range (emits log message)
func (tv *TabView) TabAtIndex(idx int) (Node2D, *TabButton, bool) {
	tv.Mu.Lock()
	defer tv.Mu.Unlock()

	fr := tv.Frame()
	tb := tv.Tabs()
	sz := len(*fr.Children())
	if idx < 0 || idx >= sz {
		log.Printf("giv.TabView: index %v out of range for number of tabs: %v\n", idx, sz)
		return nil, nil, false
	}
	tab := tb.Child(idx).Embed(TypeTabButton).(*TabButton)
	widg := fr.Child(idx).(Node2D)
	return widg, tab, true
}

// SelectTabIndex selects tab at given index, returning it -- returns false if
// index is invalid
func (tv *TabView) SelectTabIndex(idx int) (Node2D, bool) {
	widg, tab, ok := tv.TabAtIndex(idx)
	if !ok {
		return nil, false
	}
	fr := tv.Frame()
	if fr.StackTop == idx {
		return widg, true
	}
	tv.Mu.Lock()
	// tv.Viewport.BlockUpdates() // not needed for this apparently
	updt := tv.UpdateStart()
	tv.UnselectOtherTabs(idx)
	tab.SetSelectedState(true)
	fr.StackTop = idx
	fr.SetFullReRender()
	// tv.Viewport.UnblockUpdates()
	tv.Mu.Unlock()
	tv.UpdateEnd(updt)
	return widg, true
}

// SelectTabIndexAction selects tab at given index and emits selected signal,
// with the index of the selected tab -- this is what is called when a tab is
// clicked
func (tv *TabView) SelectTabIndexAction(idx int) {
	_, ok := tv.SelectTabIndex(idx)
	if ok {
		tv.TabViewSig.Emit(tv.This(), int64(TabSelected), idx)
	}
}

// TabByName returns tab with given name (nil if not found -- see TabByNameTry)
func (tv *TabView) TabByName(label string) Node2D {
	t, _ := tv.TabByNameTry(label)
	return t
}

// TabByNameTry returns tab with given name, and an error if not found.
func (tv *TabView) TabByNameTry(label string) (Node2D, error) {
	tv.Mu.Lock()
	defer tv.Mu.Unlock()

	tb := tv.Tabs()
	idx, ok := tb.Children().IndexByName(label, 0)
	if !ok {
		return nil, fmt.Errorf("gi.TabView: Tab named %v not found in %v", label, tv.Path())
	}
	fr := tv.Frame()
	widg := fr.Child(idx).(Node2D)
	return widg, nil
}

// TabIndexByName returns tab index for given tab name, and an error if not found.
func (tv *TabView) TabIndexByName(label string) (int, error) {
	tv.Mu.Lock()
	defer tv.Mu.Unlock()

	tb := tv.Tabs()
	idx, ok := tb.Children().IndexByName(label, 0)
	if !ok {
		return -1, fmt.Errorf("gi.TabView: Tab named %v not found in %v", label, tv.Path())
	}
	return idx, nil
}

// TabName returns tab name at given index
func (tv *TabView) TabName(idx int) string {
	tv.Mu.Lock()
	defer tv.Mu.Unlock()

	tb := tv.Tabs()
	tbut, err := tb.ChildTry(idx)
	if err != nil {
		return ""
	}
	return tbut.Name()
}

// SelectTabByName selects tab by name, returning it.
func (tv *TabView) SelectTabByName(label string) Node2D {
	idx, err := tv.TabIndexByName(label)
	if err == nil {
		tv.SelectTabIndex(idx)
		fr := tv.Frame()
		return fr.Child(idx).(Node2D)
	}
	return nil
}

// SelectTabByNameTry selects tab by name, returning it.  Returns error if not found.
func (tv *TabView) SelectTabByNameTry(label string) (Node2D, error) {
	idx, err := tv.TabIndexByName(label)
	if err == nil {
		tv.SelectTabIndex(idx)
		fr := tv.Frame()
		return fr.Child(idx).(Node2D), nil
	}
	return nil, err
}

// RecycleTab returns a tab with given name, first by looking for an existing one,
// and if not found, making a new one with widget of given type.
// If sel, then select it.  returns widget for tab.
func (tv *TabView) RecycleTab(label string, typ reflect.Type, sel bool) Node2D {
	widg, err := tv.TabByNameTry(label)
	if err == nil {
		if sel {
			tv.SelectTabByName(label)
		}
		return widg
	}
	widg = tv.AddNewTab(typ, label)
	if sel {
		tv.SelectTabByName(label)
	}
	return widg
}

// DeleteTabIndex deletes tab at given index, optionally calling destroy on
// tab contents -- returns widget if destroy == false, tab name, and bool success
func (tv *TabView) DeleteTabIndex(idx int, destroy bool) (Node2D, string, bool) {
	widg, _, ok := tv.TabAtIndex(idx)
	if !ok {
		return nil, "", false
	}

	tnm := tv.TabName(idx)
	tv.Mu.Lock()
	fr := tv.Frame()
	sz := len(*fr.Children())
	tb := tv.Tabs()
	updt := tv.UpdateStart()
	tv.SetFullReRender()
	nxtidx := -1
	if fr.StackTop == idx {
		if idx > 0 {
			nxtidx = idx - 1
		} else if idx < sz-1 {
			nxtidx = idx
		}
	}
	fr.DeleteChildAtIndex(idx, destroy)
	tb.DeleteChildAtIndex(idx, ki.DestroyKids) // always destroy -- we manage
	tv.RenumberTabs()
	tv.Mu.Unlock()
	if nxtidx >= 0 {
		tv.SelectTabIndex(nxtidx)
	}
	tv.UpdateEnd(updt)
	if destroy {
		return nil, tnm, true
	} else {
		return widg, tnm, true
	}
}

// DeleteTabIndexAction deletes tab at given index using destroy flag, and
// emits TabDeleted signal with name of deleted tab
// this is called by the delete button on the tab
func (tv *TabView) DeleteTabIndexAction(idx int) {
	_, tnm, ok := tv.DeleteTabIndex(idx, true)
	if ok {
		tv.TabViewSig.Emit(tv.This(), int64(TabDeleted), tnm)
	}
}

// ConfigNewTabButton configures the new tab + button at end of list of tabs
func (tv *TabView) ConfigNewTabButton() bool {
	sz := tv.NTabs()
	tb := tv.Tabs()
	ntb := len(tb.Kids)
	if tv.NewTabButton {
		if ntb == sz+1 {
			return false
		}
		if tv.NewTabType == nil {
			tv.NewTabType = TypeFrame
		}
		tab := tb.InsertNewChild(TypeAction, ntb, "new-tab").(*Action)
		tab.Data = -1
		tab.SetIcon(icons.Add)
		tab.ActionSig.ConnectOnly(tv.This(), func(recv, send ki.Ki, sig int64, data any) {
			tvv := recv.Embed(TypeTabView).(*TabView)
			tvv.SetFullReRender()
			tvv.AddNewTabAction(tvv.NewTabType, "New Tab")
			tvv.SelectTabIndex(len(*tvv.Frame().Children()) - 1)
		})
		return true
	} else {
		if ntb == sz {
			return false
		}
		tb.DeleteChildAtIndex(ntb-1, ki.DestroyKids) // always destroy -- we manage
		return true
	}
}

// TabViewSignals are signals that the TabView can send
type TabViewSignals int64

const (
	// TabSelected indicates tab was selected -- data is the tab index
	TabSelected TabViewSignals = iota

	// TabAdded indicates tab was added -- data is the tab index
	TabAdded

	// TabDeleted indicates tab was deleted -- data is the tab name
	TabDeleted

	TabViewSignalsN
)

// Config initializes the tab widget children if it hasn't been done yet
func (tv *TabView) Config() {
	if len(tv.Kids) != 0 {
		return
	}
	tv.StyMu.RLock()
	needSty := tv.Style.Font.Size.Val == 0
	tv.StyMu.RUnlock()
	if needSty {
		tv.StyleLayout()
	}

	updt := tv.UpdateStart()
	tv.Lay = LayoutVert
	tv.SetReRenderAnchor()

	AddNewFrame(tv, "tabs", LayoutHorizFlow)

	frame := AddNewFrame(tv, "frame", LayoutStacked)
	frame.SetReRenderAnchor()

	tv.ConfigNewTabButton()

	tv.UpdateEnd(updt)
}

// Tabs returns the layout containing the tabs -- the first element within us
func (tv *TabView) Tabs() *Frame {
	tv.Config()
	return tv.Child(0).(*Frame)
}

// Frame returns the stacked frame layout -- the second element
func (tv *TabView) Frame() *Frame {
	tv.Config()
	return tv.Child(1).(*Frame)
}

// UnselectOtherTabs turns off all the tabs except given one
func (tv *TabView) UnselectOtherTabs(idx int) {
	sz := tv.NTabs()
	tbs := tv.Tabs()
	for i := 0; i < sz; i++ {
		if i == idx {
			continue
		}
		tb := tbs.Child(i).Embed(TypeTabButton).(*TabButton)
		if tb.IsSelected() {
			tb.SetSelectedState(false)
		}
	}
}

// RenumberTabs assigns proper index numbers to each tab
func (tv *TabView) RenumberTabs() {
	sz := tv.NTabs()
	tbs := tv.Tabs()
	for i := 0; i < sz; i++ {
		tb := tbs.Child(i).Embed(TypeTabButton).(*TabButton)
		tb.Data = i
	}
}

func (tv *TabView) Style2D() {
	tv.Config()
	tv.Layout.Style2D()
}

// RenderTabSeps renders the separators between tabs
func (tv *TabView) RenderTabSeps() {
	rs, pc, st := tv.RenderLock()
	defer tv.RenderUnlock(rs)

	// just like with standard separator, use top width like CSS
	// (see https://www.w3schools.com/howto/howto_css_dividers.asp)
	pc.StrokeStyle.Width = st.Border.Width.Top
	pc.StrokeStyle.SetColor(&st.Border.Color.Top)
	bw := st.Border.Width.Dots()

	tbs := tv.Tabs()
	sz := len(tbs.Kids)
	for i := 1; i < sz; i++ {
		tb := tbs.Child(i).(Node2D)
		ni := tb.AsWidget()

		pos := ni.LayState.Alloc.Pos
		sz := ni.LayState.Alloc.Size.Sub(st.EffMargin().Size())
		pc.DrawLine(rs, pos.X-bw.Pos().X, pos.Y, pos.X-bw.Pos().X, pos.Y+sz.Y)
	}
	pc.FillStrokeClear(rs)
}

func (tv *TabView) Render2D() {
	if tv.FullReRenderIfNeeded() {
		return
	}
	if tv.PushBounds() {
		tv.This().(Node2D).ConnectEvents2D()
		tv.RenderScrolls()
		tv.Render2DChildren()
		tv.RenderTabSeps()
		tv.PopBounds()
	} else {
		tv.DisconnectAllEvents(AllPris) // uses both Low and Hi
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// TabButton

// TabButton is a larger select action and a small close action. Indicator
// icon is used for close icon.
type TabButton struct {
	Action

	// if true, this tab does not have the delete button avail
	NoDelete bool `desc:"if true, this tab does not have the delete button avail"`
}

var TypeTabButton = kit.Types.AddType(&TabButton{}, TabButtonProps)

var TabButtonProps = ki.Props{
	ki.EnumTypeFlag: TypeButtonFlags,
}

func (tb *TabButton) OnInit() {
	tb.AddStyler(func(w *WidgetBase, s *gist.Style) {
		s.Cursor = cursor.HandPointing
		s.MinWidth.SetCh(8)
		s.MaxWidth.SetPx(500)
		s.MinHeight.SetEm(1.6)

		// s.Border.Style.Right = gist.BorderSolid
		// s.Border.Width.Right.SetPx(1)

		s.BackgroundColor.SetSolid(ColorScheme.SurfaceContainerLow)
		s.Color = ColorScheme.OnSurface

		s.Border.Radius.Set()
		s.Text.Align = gist.AlignCenter
		s.Margin.Set()
		s.Padding.Set(units.Px(8 * Prefs.DensityMul()))

		// s.Border.Style.Set(gist.BorderNone)
		// if tb.IsSelected() {
		// 	s.Border.Style.Bottom = gist.BorderSolid
		// 	s.Border.Width.Bottom.SetPx(2)
		// 	s.Border.Color.Bottom = ColorScheme.Primary
		// }

		if tb.IsHovered() {
			s.BackgroundColor.SetSolid(ColorScheme.SurfaceContainerHighest)
		}
		if w.HasFocus() {
			s.Border.Style.Set(gist.BorderSolid)
			s.Border.Width.Set(units.Px(2))
			s.Border.Color.Set(ColorScheme.Outline)
		}
	})
}

func (tb *TabButton) OnChildAdded(child ki.Ki) {
	if w := KiAsWidget(child); w != nil {
		switch w.Name() {
		case "Parts":
			w.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.Overflow = gist.OverflowHidden // no scrollbars!
			})
		case "icon":
			w.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.Width.SetEm(1)
				s.Height.SetEm(1)
				s.Margin.Set()
				s.Padding.Set()
			})
		case "label":
			label := child.(*Label)
			label.Type = LabelTitleSmall
			w.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.Margin.Set()
				s.Padding.Set()
			})
		case "close-stretch":
			w.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.Width.SetCh(1)
			})
		case "close":
			w.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.Width.SetEx(0.5)
				s.Height.SetEx(0.5)
				s.Margin.Set()
				s.Padding.Set()
				s.AlignV = gist.AlignMiddle
				s.Border.Radius = gist.BorderRadiusFull
				s.BackgroundColor.SetSolid(colors.Transparent)
			})
		case "sc-stretch":
			w.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.MinWidth.SetCh(2)
			})
		case "shortcut":
			w.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.Margin.Set()
				s.Padding.Set()
			})
		}
	}
}

func (tb *TabButton) TabView() *TabView {
	tv := tb.ParentByType(TypeTabView, ki.Embeds)
	if tv == nil {
		return nil
	}
	return tv.Embed(TypeTabView).(*TabView)
}

func (tb *TabButton) ConfigParts() {
	if !tb.NoDelete {
		tb.ConfigPartsDeleteButton()
		return
	}
	tb.Action.ConfigParts() // regular
}

func (tb *TabButton) ConfigPartsDeleteButton() {
	config := kit.TypeAndNameList{}
	icIdx, lbIdx := tb.ConfigPartsIconLabel(&config, tb.Icon, tb.Text)
	config.Add(TypeStretch, "close-stretch")
	clsIdx := len(config)
	config.Add(TypeAction, "close")
	mods, updt := tb.Parts.ConfigChildren(config)
	tb.ConfigPartsSetIconLabel(tb.Icon, tb.Text, icIdx, lbIdx)
	if mods {
		cls := tb.Parts.Child(clsIdx).(*Action)
		if tb.Indicator.IsNil() {
			tb.Indicator = icons.Close
		}

		icnm := tb.Indicator
		cls.SetIcon(icnm)
		cls.SetProp("no-focus", true)
		cls.ActionSig.ConnectOnly(tb.This(), func(recv, send ki.Ki, sig int64, data any) {
			tbb := recv.Embed(TypeTabButton).(*TabButton)
			tabIdx := tbb.Data.(int)
			tvv := tb.TabView()
			if tvv != nil {
				if !Prefs.Params.OnlyCloseActiveTab || tbb.IsSelected() { // only process delete when already selected if OnlyCloseActiveTab is on
					tvv.DeleteTabIndexAction(tabIdx)
				} else {
					tvv.SelectTabIndexAction(tabIdx) // otherwise select
				}
			}
		})
		tb.UpdateEnd(updt)
	}
}

func (tb *TabButton) Init2D() {
	tb.Init2DWidget()
	tb.ConfigParts()
}
