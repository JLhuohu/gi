// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"strconv"
	"strings"

	"goki.dev/gi/v2/gist"
	"goki.dev/gi/v2/icons"
	"goki.dev/gi/v2/oswin"
	"goki.dev/gi/v2/oswin/key"
	"goki.dev/gi/v2/oswin/mouse"
	"goki.dev/gi/v2/units"
	"goki.dev/ki/v2/ints"
	"goki.dev/ki/v2/ki"
	"goki.dev/ki/v2/kit"
	"goki.dev/mat32/v2"
)

////////////////////////////////////////////////////////////////////////////////////////
//    SplitView

// SplitView allocates a fixed proportion of space to each child, along given
// dimension, always using only the available space given to it by its parent
// (i.e., it will force its children, which should be layouts (typically
// Frame's), to have their own scroll bars as necessary).  It should
// generally be used as a main outer-level structure within a window,
// providing a framework for inner elements -- it allows individual child
// elements to update independently and thus is important for speeding update
// performance.  It uses the Widget Parts to hold the splitter widgets
// separately from the children that contain the rest of the scenegraph to be
// displayed within each region.
type SplitView struct {
	PartsWidgetBase

	// size of the handle region in the middle of each split region, where the splitter can be dragged -- other-dimension size is 2x of this
	HandleSize units.Value `xml:"handle-size" desc:"size of the handle region in the middle of each split region, where the splitter can be dragged -- other-dimension size is 2x of this"`

	// proportion (0-1 normalized, enforced) of space allocated to each element -- can enter 0 to collapse a given element
	Splits []float32 `desc:"proportion (0-1 normalized, enforced) of space allocated to each element -- can enter 0 to collapse a given element"`

	// A saved version of the splits which can be restored -- for dynamic collapse / expand operations
	SavedSplits []float32 `desc:"A saved version of the splits which can be restored -- for dynamic collapse / expand operations"`

	// dimension along which to split the space
	Dim mat32.Dims `desc:"dimension along which to split the space"`
}

var TypeSplitView = kit.Types.AddType(&SplitView{}, SplitViewProps)

// AddNewSplitView adds a new splitview to given parent node, with given name.
func AddNewSplitView(parent ki.Ki, name string) *SplitView {
	return parent.AddNewChild(TypeSplitView, name).(*SplitView)
}

func (sv *SplitView) OnInit() {
	sv.AddStyler(func(w *WidgetBase, s *gist.Style) {
		sv.HandleSize.SetPx(10)

		s.MaxWidth.SetPx(-1)
		s.MaxHeight.SetPx(-1)
		s.Margin.Set()
		s.Padding.Set()
	})
}

func (sv *SplitView) OnChildAdded(child ki.Ki) {
	if sp, ok := child.(*Splitter); ok {
		sp.ThumbSize = sv.HandleSize
	}
}

func (sv *SplitView) CopyFieldsFrom(frm any) {
	fr := frm.(*SplitView)
	sv.PartsWidgetBase.CopyFieldsFrom(&fr.PartsWidgetBase)
	sv.HandleSize = fr.HandleSize
	mat32.CopyFloat32s(&sv.Splits, fr.Splits)
	mat32.CopyFloat32s(&sv.SavedSplits, fr.SavedSplits)
	sv.Dim = fr.Dim
}

var SplitViewProps = ki.Props{
	ki.EnumTypeFlag: TypeNodeFlags,
}

// UpdateSplits updates the splits to be same length as number of children,
// and normalized
func (sv *SplitView) UpdateSplits() {
	sz := len(sv.Kids)
	if sz == 0 {
		return
	}
	if sv.Splits == nil || len(sv.Splits) != sz {
		sv.Splits = make([]float32, sz)
	}
	sum := float32(0.0)
	for _, sp := range sv.Splits {
		sum += sp
	}
	if sum == 0 { // set default even splits
		sv.EvenSplits()
		sum = 1.0
	} else {
		norm := 1.0 / sum
		for i := range sv.Splits {
			sv.Splits[i] *= norm
		}
	}
}

// EvenSplits splits space evenly across all panels
func (sv *SplitView) EvenSplits() {
	sz := len(sv.Kids)
	if sz == 0 {
		return
	}
	even := 1.0 / float32(sz)
	for i := range sv.Splits {
		sv.Splits[i] = even
	}
}

// SetSplits sets the split proportions -- can use 0 to hide / collapse a
// child entirely -- just does the basic local update start / end
// use SetSplitsAction to trigger full rebuild which is typically required
func (sv *SplitView) SetSplits(splits ...float32) {
	updt := sv.UpdateStart()
	sv.UpdateSplits()
	sz := len(sv.Kids)
	mx := ints.MinInt(sz, len(splits))
	for i := 0; i < mx; i++ {
		sv.Splits[i] = splits[i]
	}
	sv.UpdateSplits()
	sv.UpdateEnd(updt)
}

// SetSplitsList sets the split proportions using a list (slice) argument,
// instead of variable args -- e.g., for Python or other external users.
// can use 0 to hide / collapse a child entirely -- just does the basic local
// update start / end -- use SetSplitsAction to trigger full rebuild
// which is typically required
func (sv *SplitView) SetSplitsList(splits []float32) {
	sv.SetSplits(splits...)
}

// SetSplitsAction sets the split proportions -- can use 0 to hide / collapse a
// child entirely -- does full rebuild at level of viewport
func (sv *SplitView) SetSplitsAction(splits ...float32) {
	sv.SetSplits(splits...)
	sv.ViewportSafe().SetNeedsFullRender()
}

// SaveSplits saves the current set of splits in SavedSplits, for a later RestoreSplits
func (sv *SplitView) SaveSplits() {
	sz := len(sv.Splits)
	if sz == 0 {
		return
	}
	if sv.SavedSplits == nil || len(sv.SavedSplits) != sz {
		sv.SavedSplits = make([]float32, sz)
	}
	copy(sv.SavedSplits, sv.Splits)
}

// RestoreSplits restores a previously-saved set of splits (if it exists), does an update
func (sv *SplitView) RestoreSplits() {
	if sv.SavedSplits == nil {
		return
	}
	sv.SetSplitsAction(sv.SavedSplits...)
}

// CollapseChild collapses given child(ren) (sets split proportion to 0),
// optionally saving the prior splits for later Restore function -- does an
// Update -- triggered by double-click of splitter
func (sv *SplitView) CollapseChild(save bool, idxs ...int) {
	updt := sv.UpdateStart()
	if save {
		sv.SaveSplits()
	}
	sz := len(sv.Kids)
	for _, idx := range idxs {
		if idx >= 0 && idx < sz {
			sv.Splits[idx] = 0
		}
	}
	sv.UpdateSplits()
	sv.ViewportSafe().SetNeedsFullRender() // splits typically require full rebuild
	sv.UpdateEnd(updt)
}

// RestoreChild restores given child(ren) -- does an Update
func (sv *SplitView) RestoreChild(idxs ...int) {
	updt := sv.UpdateStart()
	sz := len(sv.Kids)
	for _, idx := range idxs {
		if idx >= 0 && idx < sz {
			sv.Splits[idx] = 1.0 / float32(sz)
		}
	}
	sv.UpdateSplits()
	sv.ViewportSafe().SetNeedsFullRender() // splits typically require full rebuild
	sv.UpdateEnd(updt)
}

// IsCollapsed returns true if given split number is collapsed
func (sv *SplitView) IsCollapsed(idx int) bool {
	sz := len(sv.Kids)
	if idx >= 0 && idx < sz {
		return sv.Splits[idx] < 0.01
	}
	return false
}

// SetSplitAction sets the new splitter value, for given splitter -- new
// value is 0..1 value of position of that splitter -- it is a sum of all the
// positions up to that point.  Splitters are updated to ensure that selected
// position is achieved, while dividing remainder appropriately.
func (sv *SplitView) SetSplitAction(idx int, nwval float32) {
	sz := len(sv.Splits)
	oldsum := float32(0)
	for i := 0; i <= idx; i++ {
		oldsum += sv.Splits[i]
	}
	delta := nwval - oldsum
	oldval := sv.Splits[idx]
	uval := oldval + delta
	if uval < 0 {
		uval = 0
		delta = -oldval
		nwval = oldsum + delta
	}
	rmdr := 1 - nwval
	if idx < sz-1 {
		oldrmdr := 1 - oldsum
		if oldrmdr <= 0 {
			if rmdr > 0 {
				dper := rmdr / float32((sz-1)-idx)
				for i := idx + 1; i < sz; i++ {
					sv.Splits[i] = dper
				}
			}
		} else {
			for i := idx + 1; i < sz; i++ {
				curval := sv.Splits[i]
				sv.Splits[i] = rmdr * (curval / oldrmdr) // proportional
			}
		}
	}
	sv.Splits[idx] = uval
	// fmt.Printf("splits: %v value: %v  splts: %v\n", idx, nwval, sv.Splits)
	sv.UpdateSplits()
	// fmt.Printf("splits: %v\n", sv.Splits)
	sv.ViewportSafe().SetNeedsFullRender() // splits typically require full rebuild
}

func (sv *SplitView) Init2D() {
	sv.Parts.Lay = LayoutNil
	sv.Init2DWidget()
	sv.UpdateSplits()
	sv.ConfigSplitters()
}

func (sv *SplitView) ConfigSplitters() {
	sz := len(sv.Kids)
	mods, updt := sv.Parts.SetNChildren(sz-1, TypeSplitter, "Splitter")
	odim := mat32.OtherDim(sv.Dim)
	spc := sv.BoxSpace()
	size := sv.LayState.Alloc.Size.Dim(sv.Dim) - spc.Size().Dim(sv.Dim)
	handsz := sv.HandleSize.Dots
	mid := 0.5 * (sv.LayState.Alloc.Size.Dim(odim) - spc.Size().Dim(odim))
	spicon := icons.DragHandle
	if sv.Dim == mat32.X {
		spicon = icons.DragIndicator
	}
	for i, spk := range *sv.Parts.Children() {
		sp := spk.(*Splitter)
		sp.SplitterNo = i
		sp.Icon = spicon
		sp.Dim = sv.Dim
		sp.LayState.Alloc.Size.SetDim(sv.Dim, size)
		sp.LayState.Alloc.Size.SetDim(odim, handsz*2)
		sp.LayState.Alloc.SizeOrig = sp.LayState.Alloc.Size
		sp.LayState.Alloc.PosRel.SetDim(sv.Dim, 0)
		sp.LayState.Alloc.PosRel.SetDim(odim, mid-handsz+float32(i)*handsz*4)
		sp.LayState.Alloc.PosOrig = sp.LayState.Alloc.PosRel
		sp.Min = 0.0
		sp.Max = 1.0
		sp.Snap = false
		sp.ThumbSize = sv.HandleSize
		if mods {
			sp.SliderSig.ConnectOnly(sv.This(), func(recv, send ki.Ki, sig int64, data any) {
				if sig == int64(SliderReleased) {
					spr, _ := recv.Embed(TypeSplitView).(*SplitView)
					spl := send.(*Splitter)
					spr.SetSplitAction(spl.SplitterNo, spl.Value)
				}
			})
		}
	}
	if mods {
		sv.Parts.UpdateEnd(updt)
	}
}

func (sv *SplitView) KeyInput(kt *key.ChordEvent) {
	kc := string(kt.Chord())
	mod := "Control+"
	if oswin.TheApp.Platform() == oswin.MacOS {
		mod = "Meta+"
	}
	if !strings.HasPrefix(kc, mod) {
		return
	}
	kns := kc[len(mod):]

	knc, err := strconv.Atoi(kns)
	if err != nil {
		return
	}
	kn := int(knc)
	// fmt.Printf("kc: %v kns: %v kn: %v\n", kc, kns, kn)
	if kn == 0 {
		sv.EvenSplits()
		sv.SetFullReRender()
		sv.UpdateSig()
		kt.SetProcessed()
	} else if kn <= len(sv.Kids) {
		sv.SetFullReRender()
		if sv.Splits[kn-1] <= 0.01 {
			sv.RestoreChild(kn - 1)
		} else {
			sv.CollapseChild(true, kn-1)
		}
		kt.SetProcessed()
	}
}

func (sv *SplitView) KeyChordEvent() {
	sv.ConnectEvent(oswin.KeyChordEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		svv := recv.Embed(TypeSplitView).(*SplitView)
		svv.KeyInput(d.(*key.ChordEvent))
	})
}

func (sv *SplitView) SplitViewEvents() {
	sv.KeyChordEvent()
}

func (sv *SplitView) StyleSplitView() {
	sv.Style2DWidget()
	sv.LayState.SetFromStyle(&sv.Style) // also does reset
	sv.HandleSize.SetFmInheritProp("handle-size", sv.This(), ki.NoInherit, ki.TypeProps)
	sv.HandleSize.ToDots(&sv.Style.UnContext)
}

func (sv *SplitView) Style2D() {
	sv.StyMu.Lock()

	sv.StyleSplitView()
	sv.LayState.SetFromStyle(&sv.Style) // also does reset
	sv.UpdateSplits()
	sv.StyMu.Unlock()

	sv.ConfigSplitters()
}

func (sv *SplitView) Layout2D(parBBox image.Rectangle, iter int) bool {
	sv.ConfigSplitters()
	sv.Layout2DBase(parBBox, true, iter) // init style
	sv.Layout2DParts(parBBox, iter)
	sv.UpdateSplits()

	handsz := sv.HandleSize.Dots
	// fmt.Printf("handsz: %v\n", handsz)
	sz := len(sv.Kids)
	odim := mat32.OtherDim(sv.Dim)
	spc := sv.BoxSpace()
	size := sv.LayState.Alloc.Size.Dim(sv.Dim) - spc.Size().Dim(sv.Dim)
	avail := size - handsz*float32(sz-1)
	// fmt.Printf("avail: %v\n", avail)
	osz := sv.LayState.Alloc.Size.Dim(odim) - spc.Size().Dim(odim)
	pos := float32(0.0)

	spsum := float32(0)
	for i, sp := range sv.Splits {
		gis := sv.Kids[i].(Node2D).AsWidget()
		if gis == nil {
			continue
		}
		if ki.TypeEmbeds(gis, TypeFrame) {
			gis.SetReRenderAnchor()
		}
		isz := sp * avail
		gis.LayState.Alloc.Size.SetDim(sv.Dim, isz)
		gis.LayState.Alloc.Size.SetDim(odim, osz)
		gis.LayState.Alloc.SizeOrig = gis.LayState.Alloc.Size
		gis.LayState.Alloc.PosRel.SetDim(sv.Dim, pos)
		gis.LayState.Alloc.PosRel.SetDim(odim, spc.Pos().Dim(odim))
		// fmt.Printf("spl: %v sp: %v size: %v alloc: %v  pos: %v\n", i, sp, isz, gis.LayState.Alloc.SizeOrig, gis.LayState.Alloc.PosRel)

		pos += isz + handsz

		spsum += sp
		if i < sz-1 {
			spl := sv.Parts.Child(i).(*Splitter)
			spl.Value = spsum
			spl.UpdatePosFromValue()
		}
	}

	return sv.Layout2DChildren(iter)
}

func (sv *SplitView) Render2D() {
	if sv.FullReRenderIfNeeded() {
		return
	}
	if sv.PushBounds() {
		sv.This().(Node2D).ConnectEvents2D()
		for i, kid := range sv.Kids {
			nii, ni := KiToNode2D(kid)
			if nii != nil {
				sp := sv.Splits[i]
				if sp <= 0.01 {
					ni.SetInvisible()
				} else {
					ni.ClearInvisible()
				}
				nii.Render2D() // needs to disconnect using invisible
			}
		}
		sv.Parts.Render2DTree()
		sv.PopBounds()
	}
}

func (sv *SplitView) ConnectEvents2D() {
	sv.SplitViewEvents()
}

func (sv *SplitView) HasFocus2D() bool {
	return sv.ContainsFocus() // anyone within us gives us focus..
}

////////////////////////////////////////////////////////////////////////////////////////
//    Splitter

// Splitter provides the splitter handle and line separating two elements in a
// SplitView, with draggable resizing of the splitter -- parent is Parts
// layout of the SplitView -- based on SliderBase
type Splitter struct {
	SliderBase

	// splitter number this one is
	SplitterNo int `desc:"splitter number this one is"`

	// copy of the win bbox, used for translating mouse events when the bbox is restricted to the slider itself
	OrigWinBBox image.Rectangle `copy:"-" json:"-" xml:"-" desc:"copy of the win bbox, used for translating mouse events when the bbox is restricted to the slider itself"`
}

var TypeSplitter = kit.Types.AddType(&Splitter{}, SplitterProps)

var SplitterProps = ki.Props{
	ki.EnumTypeFlag: TypeNodeFlags,
}

func (sr *Splitter) OnInit() {
	// STYTODO: fix splitter styles
	sr.ValThumb = false
	sr.ThumbSize = units.Px(10) // will be replaced by parent HandleSize
	sr.Step = 0.01
	sr.PageStep = 0.1
	sr.Max = 1.0
	sr.Snap = false
	sr.Prec = 4
	sr.SetFlag(int(InstaDrag))

	sr.AddStyler(func(w *WidgetBase, s *gist.Style) {
		s.Margin.Set()
		s.Padding.Set(units.Px(6 * Prefs.DensityMul()))
		s.BackgroundColor.SetSolid(ColorScheme.TertiaryContainer)
		s.Color = ColorScheme.OnBackground
		if sr.Dim == mat32.X {
			s.MinWidth.SetPx(2)
			s.MinHeight.SetPx(100)
			s.Height.SetPx(100)
			s.MaxHeight.SetPx(100)
		} else {
			s.MinHeight.SetPx(2)
			s.MinWidth.SetPx(100)
		}
	})

}

func (sr *Splitter) OnChildAdded(child ki.Ki) {
	if w := KiAsWidget(child); w != nil {
		switch w.Name() {
		case "icon":
			// w.AddStyler(func(w *WidgetBase, s *gist.Style) {
			// 	s.MaxWidth.SetEm(1)
			// 	s.MaxHeight.SetEm(5)
			// 	s.MinWidth.SetEm(1)
			// 	s.MinHeight.SetEm(5)
			// 	s.Margin.Set()
			// 	s.Padding.Set()
			// 	s.AlignV = gist.AlignMiddle
			// })
		}
	}
}

func (sr *Splitter) Init2D() {
	sr.Init2DSlider()
	sr.ConfigParts()
}

func (sr *Splitter) ConfigPartsIfNeeded(render bool) {
	if sr.PartsNeedUpdateIconLabel(sr.Icon, "") {
		sr.ConfigParts()
	}
	if !TheIconMgr.IsValid(sr.Icon) || !sr.Parts.HasChildren() {
		return
	}
	ick := sr.Parts.ChildByType(TypeIcon, ki.Embeds, 0)
	if ick == nil {
		return
	}
	ic := ick.(*Icon)
	handsz := sr.ThumbSize.Dots
	spc := sr.BoxSpace()
	odim := mat32.OtherDim(sr.Dim)
	sr.LayState.Alloc.Size.SetDim(odim, 2*(handsz+spc.Size().Dim(odim)))
	sr.LayState.Alloc.SizeOrig = sr.LayState.Alloc.Size

	ic.LayState.Alloc.Size.SetDim(odim, 2*handsz)
	ic.LayState.Alloc.Size.SetDim(sr.Dim, handsz)
	ic.LayState.Alloc.PosRel.SetDim(sr.Dim, sr.Pos-(0.5*(handsz+spc.Pos().Dim(sr.Dim))))
	ic.LayState.Alloc.PosRel.SetDim(odim, 0)
	if render {
		ic.Layout2DTree()
	}
}

func (sr *Splitter) Style2D() {
	sr.ClearFlag(int(CanFocus))
	sr.StyleSlider()
	sr.StyMu.Lock()
	sr.LayState.SetFromStyle(&sr.Style) // also does reset
	sr.StyMu.Unlock()
	sr.ConfigParts()
}

func (sr *Splitter) Size2D(iter int) {
	sr.InitLayout2D()
}

func (sr *Splitter) Layout2D(parBBox image.Rectangle, iter int) bool {
	sr.ConfigPartsIfNeeded(false)
	sr.Layout2DBase(parBBox, true, iter) // init style
	sr.Layout2DParts(parBBox, iter)
	// sr.SizeFromAlloc()
	sr.Size = sr.LayState.Alloc.Size.Dim(sr.Dim)
	sr.UpdatePosFromValue()
	sr.DragPos = sr.Pos
	sr.BBoxMu.RLock()
	sr.OrigWinBBox = sr.WinBBox
	sr.BBoxMu.RUnlock()
	return sr.Layout2DChildren(iter)
}

func (sr *Splitter) PointToRelPos(pt image.Point) image.Point {
	// this updates the SliderPositioner interface to use OrigWinBBox
	return pt.Sub(sr.OrigWinBBox.Min)
}

func (sr *Splitter) UpdateSplitterPos() {
	spc := sr.BoxSpace()
	handsz := sr.ThumbSize.Dots
	off := 0
	if sr.Dim == mat32.X {
		off = sr.OrigWinBBox.Min.X
	} else {
		off = sr.OrigWinBBox.Min.Y
	}
	sz := handsz
	if !sr.IsDragging() {
		sz += spc.Size().Dim(sr.Dim)
	}
	pos := off + int(sr.Pos-0.5*sz)
	mxpos := off + int(sr.Pos+0.5*sz)

	// SidesTODO: this is all sketchy

	if sr.IsDragging() {
		win := sr.ParentWindow()
		spnm := "gi.Splitter:" + sr.Name()
		spr, ok := win.SpriteByName(spnm)
		if ok {
			spr.Geom.Pos = image.Point{pos, sr.ObjBBox.Min.Y + int(spc.Top)}
		}
	} else {
		sr.BBoxMu.Lock()

		if sr.Dim == mat32.X {
			sr.VpBBox = image.Rect(pos, sr.ObjBBox.Min.Y+int(spc.Top), mxpos, sr.ObjBBox.Max.Y+int(spc.Bottom))
			sr.WinBBox = image.Rect(pos, sr.ObjBBox.Min.Y+int(spc.Top), mxpos, sr.ObjBBox.Max.Y+int(spc.Bottom))
		} else {
			sr.VpBBox = image.Rect(sr.ObjBBox.Min.X+int(spc.Left), pos, sr.ObjBBox.Max.X+int(spc.Right), mxpos)
			sr.WinBBox = image.Rect(sr.ObjBBox.Min.X+int(spc.Left), pos, sr.ObjBBox.Max.X+int(spc.Right), mxpos)
		}
		sr.BBoxMu.Unlock()
	}
}

// SplitView returns our parent split view
func (sr *Splitter) SplitView() *SplitView {
	if sr.Par == nil || sr.Par.Parent() == nil {
		return nil
	}
	svi := sr.Par.Parent().Embed(TypeSplitView)
	if svi == nil {
		return nil
	}
	return svi.(*SplitView)
}

func (sr *Splitter) MouseEvent() {
	sr.ConnectEvent(oswin.MouseEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.Event)
		srr := recv.Embed(TypeSplitter).(*Splitter)
		if srr.IsDisabled() {
			me.SetProcessed()
			srr.SetSelectedState(!srr.IsSelected())
			srr.EmitSelectedSignal()
			srr.UpdateSig()
		} else {
			if me.Button == mouse.Left {
				me.SetProcessed()
				if me.Action == mouse.Press {
					ed := srr.This().(SliderPositioner).PointToRelPos(me.Where)
					st := &srr.Style
					// SidesTODO: unsure about dim
					spc := st.EffMargin().Pos().Dim(srr.Dim) + 0.5*srr.ThSize
					if srr.Dim == mat32.X {
						srr.SliderPress(float32(ed.X) - spc)
					} else {
						srr.SliderPress(float32(ed.Y) - spc)
					}
				} else if me.Action == mouse.DoubleClick {
					sv := srr.SplitView()
					if sv != nil {
						if sv.IsCollapsed(srr.SplitterNo) {
							sv.RestoreSplits()
						} else {
							sv.CollapseChild(true, srr.SplitterNo)
						}
					}
				} else {
					srr.SliderRelease()
				}
			}
		}
	})
}

func (sr *Splitter) MouseScrollEvent() {
	// todo: just disabling at this point to prevent bad side-effects
	// sr.ConnectEvent(oswin.MouseScrollEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
	// 	srr := recv.Embed(TypeSliderBase).(*SliderBase)
	// 	if srr.IsInactive() {
	// 		return
	// 	}
	// 	me := d.(*mouse.ScrollEvent)
	// 	me.SetProcessed()
	// 	cur := float32(srr.Pos)
	// 	if srr.Dim == mat32.X {
	// 		srr.SliderMove(cur, cur+float32(me.NonZeroDelta(true))) // preferX
	// 	} else {
	// 		srr.SliderMove(cur, cur-float32(me.NonZeroDelta(false))) // preferY
	// 	}
	// })
}

func (sr *Splitter) SplitterEvents() {
	sr.MouseDragEvent()
	sr.MouseEvent()
	sr.MouseFocusEvent()
	sr.MouseScrollEvent()
	sr.KeyChordEvent()
}

func (sr *Splitter) ConnectEvents2D() {
	sr.SplitterEvents()
}

func (sr *Splitter) Render2D() {
	win := sr.ParentWindow()
	sr.This().(Node2D).ConnectEvents2D()
	spnm := "gi.Splitter:" + sr.Name()
	if sr.IsDragging() {
		ick := sr.Parts.ChildByType(TypeIcon, ki.Embeds, 0)
		if ick == nil {
			return
		}
		ic := ick.(*Icon)
		icvp := ic.ChildByType(TypeViewport2D, ki.Embeds, 0)
		if icvp == nil {
			return
		}
		spr, ok := win.SpriteByName(spnm)
		if !ok {
			spr = NewSprite(spnm, image.Point{}, sr.VpBBox.Min)
			spr.GrabRenderFrom(icvp.(Node2D))
			win.AddSprite(spr)
			win.ActivateSprite(spnm)
		}
		sr.UpdateSplitterPos()
		win.UpdateSig()
	} else {
		if win.DeleteSprite(spnm) {
			win.UpdateSig()
		}
		sr.UpdateSplitterPos()
		if sr.FullReRenderIfNeeded() {
			return
		}
		if sr.PushBounds() {
			sr.RenderSplitter()
			sr.Render2DChildren()
			sr.PopBounds()
		}
	}
}

// RenderSplitter does the default splitter rendering
func (sr *Splitter) RenderSplitter() {
	sr.UpdateSplitterPos()
	sr.ConfigPartsIfNeeded(true)

	if TheIconMgr.IsValid(sr.Icon) && sr.Parts.HasChildren() {
		sr.Parts.Render2DTree()
	}
	// else {
	rs, pc, st := sr.RenderLock()

	pc.StrokeStyle.SetColor(nil)
	pc.FillStyle.SetColorSpec(&st.BackgroundColor)

	// pos := mat32.NewVec2FmPoint(sr.VpBBox.Min)
	// pos.SetSubDim(mat32.OtherDim(sr.Dim), 10.0)
	// sz := mat32.NewVec2FmPoint(sr.VpBBox.Size())
	// sr.RenderBoxImpl(pos, sz, st.Border)

	sr.RenderStdBox(st)

	sr.RenderUnlock(rs)
	// }
}

func (sr *Splitter) FocusChanged2D(change FocusChanges) {
	switch change {
	case FocusLost:
		sr.SetSliderState(SliderActive) // lose any hover state but whatever..
		sr.UpdateSig()
	case FocusGot:
		sr.SetSliderState(SliderFocus)
		sr.EmitFocusedSignal()
		sr.UpdateSig()
	case FocusInactive: // don't care..
	case FocusActive:
	}
}
