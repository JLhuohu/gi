// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"image/color"

	"goki.dev/colors"
	"goki.dev/gi/v2/girl"
	"goki.dev/gi/v2/gist"
	"goki.dev/gi/v2/oswin"
	"goki.dev/gi/v2/oswin/cursor"
	"goki.dev/gi/v2/oswin/mouse"
	"goki.dev/ki/v2/ki"
	"goki.dev/ki/v2/kit"
	"goki.dev/mat32/v2"
)

////////////////////////////////////////////////////////////////////////////////////////
// Label

// Label is a widget for rendering text labels -- supports full widget model
// including box rendering, and full HTML styling, including links -- LinkSig
// emits link with data of URL -- opens default browser if nobody receiving
// signal.  The default white-space option is 'pre' -- set to 'normal' or
// other options to get word-wrapping etc.
type Label struct {
	WidgetBase

	// label to display
	Text string `xml:"text" desc:"label to display"`

	// is this label selectable? if so, it will change background color in response to selection events and update selection state on mouse clicks
	Selectable bool `desc:"is this label selectable? if so, it will change background color in response to selection events and update selection state on mouse clicks"`

	// is this label going to be redrawn frequently without an overall full re-render?  if so, you need to set this flag to avoid weird overlapping rendering results from antialiasing.  Also, if the label will change dynamically, this must be set to true, otherwise labels will illegibly overlay on top of each other.
	Redrawable bool `desc:"is this label going to be redrawn frequently without an overall full re-render?  if so, you need to set this flag to avoid weird overlapping rendering results from antialiasing.  Also, if the label will change dynamically, this must be set to true, otherwise labels will illegibly overlay on top of each other."`

	// the type of label
	Type LabelTypes `desc:"the type of label"`

	// [view: -] signal for clicking on a link -- data is a string of the URL -- if nobody receiving this signal, calls TextLinkHandler then URLHandler
	LinkSig ki.Signal `copy:"-" json:"-" xml:"-" view:"-" desc:"signal for clicking on a link -- data is a string of the URL -- if nobody receiving this signal, calls TextLinkHandler then URLHandler"`

	// render data for text label
	Render girl.Text `copy:"-" xml:"-" json:"-" desc:"render data for text label"`

	// position offset of start of text rendering, from last render -- AllocPos plus alignment factors for center, right etc.
	RenderPos mat32.Vec2 `copy:"-" xml:"-" json:"-" desc:"position offset of start of text rendering, from last render -- AllocPos plus alignment factors for center, right etc."`

	// current background color -- grabbed when rendering for first time, and used when toggling off of selected mode, or for redrawable, to wipe out bg
	CurBackgroundColor color.RGBA `copy:"-" xml:"-" json:"-" desc:"current background color -- grabbed when rendering for first time, and used when toggling off of selected mode, or for redrawable, to wipe out bg"`
}

var TypeLabel = kit.Types.AddType(&Label{}, LabelProps)

// LabelTypes is an enum containing the different
// possible types of labels
type LabelTypes int

const (
	// LabelDisplayLarge is a large, short, and important
	// display label with a default font size of 57px.
	LabelDisplayLarge LabelTypes = iota
	// LabelDisplayMedium is a medium-sized, short, and important
	// display label with a default font size of 45px.
	LabelDisplayMedium
	// LabelDisplaySmall is a small, short, and important
	// display label with a default font size of 36px.
	LabelDisplaySmall

	// LabelHeadlineLarge is a large, high-emphasis
	// headline label with a default font size of 32px.
	LabelHeadlineLarge
	// LabelHeadlineMedium is a medium-sized, high-emphasis
	// headline label with a default font size of 28px.
	LabelHeadlineMedium
	// LabelHeadlineSmall is a small, high-emphasis
	// headline label with a default font size of 24px.
	LabelHeadlineSmall

	// LabelTitleLarge is a large, medium-emphasis
	// title label with a default font size of 22px.
	LabelTitleLarge
	// LabelTitleMedium is a medium-sized, medium-emphasis
	// title label with a default font size of 16px.
	LabelTitleMedium
	// LabelTitleSmall is a small, medium-emphasis
	// title label with a default font size of 14px.
	LabelTitleSmall

	// LabelBodyLarge is a large body label used for longer
	// passages of text with a default font size of 16px.
	LabelBodyLarge
	// LabelBodyMedium is a medium-sized body label used for longer
	// passages of text with a default font size of 14px.
	LabelBodyMedium
	// LabelBodySmall is a small body label used for longer
	// passages of text with a default font size of 12px.
	LabelBodySmall

	// LabelLabelLarge is a large label used for label text (like a caption
	// or the text inside a button) with a default font size of 14px.
	LabelLabelLarge
	// LabelLabelMedium is a medium-sized label used for label text (like a caption
	// or the text inside a button) with a default font size of 12px.
	LabelLabelMedium
	// LabelLabelSmall is a small label used for label text (like a caption
	// or the text inside a button) with a default font size of 11px.
	LabelLabelSmall

	LabelTypesN
)

var TypeLabelTypes = kit.Enums.AddEnumAltLower(LabelTypesN, kit.NotBitFlag, gist.StylePropProps, "Label")

// AddNewLabel adds a new label to given parent node, with given name and text.
func AddNewLabel(parent ki.Ki, name string, text string) *Label {
	lb := parent.AddNewChild(TypeLabel, name).(*Label)
	lb.Text = text
	return lb
}

func (lb *Label) OnInit() {
	lb.Type = LabelLabelLarge
	lb.AddStyler(func(w *WidgetBase, s *gist.Style) {
		s.Cursor = lb.ParentCursor(cursor.IBeam)
		s.Text.WhiteSpace = gist.WhiteSpaceNormal
		s.AlignV = gist.AlignMiddle
		s.BackgroundColor.SetSolid(colors.Transparent)
		// Label styles based on https://m3.material.io/styles/typography/type-scale-tokens
		// TODO: maybe support brand and plain global fonts with larger labels defaulting to brand and smaller to plain
		switch lb.Type {
		case LabelLabelLarge:
			s.Text.LineHeight.SetPx(20)
			s.Font.Size.SetPx(14)
			s.Text.LetterSpacing.SetPx(0.1)
			s.Font.Weight = gist.WeightMedium
		case LabelLabelMedium:
			s.Text.LineHeight.SetPx(16)
			s.Font.Size.SetPx(12)
			s.Text.LetterSpacing.SetPx(0.5)
			s.Font.Weight = gist.WeightMedium
		case LabelLabelSmall:
			s.Text.LineHeight.SetPx(16)
			s.Font.Size.SetPx(11)
			s.Text.LetterSpacing.SetPx(0.5)
			s.Font.Weight = gist.WeightMedium
		case LabelBodyLarge:
			s.Text.LineHeight.SetPx(24)
			s.Font.Size.SetPx(16)
			s.Text.LetterSpacing.SetPx(0.5)
			s.Font.Weight = gist.WeightNormal
		case LabelBodyMedium:
			s.Text.LineHeight.SetPx(20)
			s.Font.Size.SetPx(14)
			s.Text.LetterSpacing.SetPx(0.25)
			s.Font.Weight = gist.WeightNormal
		case LabelBodySmall:
			s.Text.LineHeight.SetPx(16)
			s.Font.Size.SetPx(12)
			s.Text.LetterSpacing.SetPx(0.4)
			s.Font.Weight = gist.WeightNormal
		case LabelTitleLarge:
			s.Text.LineHeight.SetPx(28)
			s.Font.Size.SetPx(22)
			s.Text.LetterSpacing.SetPx(0)
			s.Font.Weight = gist.WeightNormal
		case LabelTitleMedium:
			s.Text.LineHeight.SetPx(24)
			s.Font.Size.SetPx(16)
			s.Text.LetterSpacing.SetPx(0.15)
			s.Font.Weight = gist.WeightMedium
		case LabelTitleSmall:
			s.Text.LineHeight.SetPx(20)
			s.Font.Size.SetPx(14)
			s.Text.LetterSpacing.SetPx(0.1)
			s.Font.Weight = gist.WeightMedium
		case LabelHeadlineLarge:
			s.Text.LineHeight.SetPx(40)
			s.Font.Size.SetPx(32)
			s.Text.LetterSpacing.SetPx(0)
			s.Font.Weight = gist.WeightNormal
		case LabelHeadlineMedium:
			s.Text.LineHeight.SetPx(36)
			s.Font.Size.SetPx(28)
			s.Text.LetterSpacing.SetPx(0)
			s.Font.Weight = gist.WeightNormal
		case LabelHeadlineSmall:
			s.Text.LineHeight.SetPx(32)
			s.Font.Size.SetPx(24)
			s.Text.LetterSpacing.SetPx(0)
			s.Font.Weight = gist.WeightNormal
		case LabelDisplayLarge:
			s.Text.LineHeight.SetPx(64)
			s.Font.Size.SetPx(57)
			s.Text.LetterSpacing.SetPx(-0.25)
			s.Font.Weight = gist.WeightNormal
		case LabelDisplayMedium:
			s.Text.LineHeight.SetPx(52)
			s.Font.Size.SetPx(45)
			s.Text.LetterSpacing.SetPx(0)
			s.Font.Weight = gist.WeightNormal
		case LabelDisplaySmall:
			s.Text.LineHeight.SetPx(44)
			s.Font.Size.SetPx(36)
			s.Text.LetterSpacing.SetPx(0)
			s.Font.Weight = gist.WeightNormal
		}
		if w.IsDisabled() {
			s.Font.Opacity = 0.7
		}
		if w.IsSelected() {
			s.BackgroundColor.SetSolid(ColorScheme.TertiaryContainer)
			s.Color = ColorScheme.OnTertiaryContainer
		}
	})
}

func (lb *Label) OnAdd() {
	lb.Selectable = lb.ParentByType(TypeButtonBase, ki.Embeds) == nil
}

func (lb *Label) CopyFieldsFrom(frm any) {
	fr := frm.(*Label)
	lb.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	lb.Text = fr.Text
	lb.Selectable = fr.Selectable
	lb.Redrawable = fr.Redrawable
}

func (lb *Label) Disconnect() {
	lb.WidgetBase.Disconnect()
	lb.LinkSig.DisconnectAll()
}

var LabelProps = ki.Props{
	ki.EnumTypeFlag: TypeNodeFlags,
}

// SetText sets the text and updates the rendered version.
// Note: if there is already a label set, and no other
// larger updates are taking place, the new label may just
// illegibly overlay on top of the old one.
// Set Redrawable = true to fix this issue (it will redraw
// the background -- sampling from actual if none is set).
func (lb *Label) SetText(txt string) {
	updt := lb.UpdateStart()
	// if lb.Text != "" { // not good to automate this -- better to use docs -- bg can be bad
	// 	lb.Redrawable = true
	// }

	lb.StyMu.RLock()
	needSty := lb.Style.Font.Size.Val == 0
	lb.StyMu.RUnlock()
	if needSty {
		lb.StyleLabel()
	}
	lb.StyMu.RLock()
	lb.Text = txt
	lb.Style.BackgroundColor.Color = colors.Transparent // always use transparent bg for actual text
	// this makes it easier for it to update with dynamic bgs
	if lb.Text == "" {
		lb.Render.SetHTML(" ", lb.Style.FontRender(), &lb.Style.Text, &lb.Style.UnContext, lb.CSSAgg)
	} else {
		lb.Render.SetHTML(lb.Text, lb.Style.FontRender(), &lb.Style.Text, &lb.Style.UnContext, lb.CSSAgg)
	}
	spc := lb.BoxSpace()
	sz := lb.LayState.Alloc.Size
	if sz.IsNil() {
		sz = lb.LayState.SizePrefOrMax()
	}
	if !sz.IsNil() {
		sz.SetSub(spc.Size())
	}
	lb.Render.LayoutStdLR(&lb.Style.Text, lb.Style.FontRender(), &lb.Style.UnContext, sz)
	lb.StyMu.RUnlock()
	lb.UpdateEnd(updt)
}

// OpenLink opens given link, either by sending LinkSig signal if there are
// receivers, or by calling the TextLinkHandler if non-nil, or URLHandler if
// non-nil (which by default opens user's default browser via
// oswin/App.OpenURL())
func (lb *Label) OpenLink(tl *girl.TextLink) {
	tl.Widget = lb.This()
	if len(lb.LinkSig.Cons) == 0 {
		if girl.TextLinkHandler != nil {
			if girl.TextLinkHandler(*tl) {
				return
			}
		}
		if girl.URLHandler != nil {
			girl.URLHandler(tl.URL)
		}
		return
	}
	lb.LinkSig.Emit(lb.This(), 0, tl.URL) // todo: could potentially signal different target=_blank kinds of options here with the sig
}

func (lb *Label) HoverEvent() {
	lb.ConnectEvent(oswin.MouseHoverEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.HoverEvent)
		llb := recv.Embed(TypeLabel).(*Label)
		hasLinks := len(lb.Render.Links) > 0
		if hasLinks {
			pos := llb.RenderPos
			for ti := range llb.Render.Links {
				tl := &llb.Render.Links[ti]
				tlb := tl.Bounds(&llb.Render, pos)
				if me.Where.In(tlb) {
					PopupTooltip(tl.URL, tlb.Max.X, tlb.Max.Y, llb.Viewport, llb.Nm)
					me.SetProcessed()
					return
				}
			}
		}
		if llb.Tooltip != "" {
			me.SetProcessed()
			llb.BBoxMu.RLock()
			pos := llb.WinBBox.Max
			llb.BBoxMu.RUnlock()
			pos.X -= 20
			PopupTooltip(llb.Tooltip, pos.X, pos.Y, llb.Viewport, llb.Nm)
		}
	})
}

func (lb *Label) MouseEvent() {
	lb.ConnectEvent(oswin.MouseEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.Event)
		llb := recv.Embed(TypeLabel).(*Label)
		hasLinks := len(llb.Render.Links) > 0
		pos := llb.RenderPos
		if me.Action == mouse.Press && me.Button == mouse.Left && hasLinks {
			for ti := range llb.Render.Links {
				tl := &llb.Render.Links[ti]
				tlb := tl.Bounds(&llb.Render, pos)
				if me.Where.In(tlb) {
					llb.OpenLink(tl)
					me.SetProcessed()
					return
				}
			}
		}
		if me.Action == mouse.DoubleClick && me.Button == mouse.Left && llb.Selectable {
			updt := llb.UpdateStart()
			llb.SetSelectedState(!llb.IsSelected())
			llb.EmitSelectedSignal()
			llb.UpdateEnd(updt)
		}
		if me.Action == mouse.Release && me.Button == mouse.Right {
			me.SetProcessed()
			llb.EmitContextMenuSignal()
			llb.This().(Node2D).ContextMenu()
		}
	})
}

func (lb *Label) MouseMoveEvent() {
	hasLinks := len(lb.Render.Links) > 0
	if !hasLinks {
		return
	}
	lb.ConnectEvent(oswin.MouseMoveEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.MoveEvent)
		me.SetProcessed()
		llb := recv.Embed(TypeLabel).(*Label)
		pos := llb.RenderPos
		inLink := false
		for _, tl := range llb.Render.Links {
			tlb := tl.Bounds(&llb.Render, pos)
			if me.Where.In(tlb) {
				inLink = true
				break
			}
		}
		// TODO: figure out how to get links to work with new cursor setup
		if inLink {
			oswin.TheApp.Cursor(lb.ParentWindow().OSWin).PushIfNot(cursor.HandPointing)
		} else {
			oswin.TheApp.Cursor(lb.ParentWindow().OSWin).PopIf(cursor.HandPointing)
		}
	})
}

func (lb *Label) LabelEvents() {
	lb.HoverEvent()
	lb.MouseEvent()
	lb.MouseMoveEvent()
}

func (lb *Label) GrabCurBackgroundColor() {
	if lb.Viewport == nil || lb.IsSelected() {
		return
	}
	if !gist.RebuildDefaultStyles && !colors.IsNil(lb.CurBackgroundColor) {
		return
	}
	pos := lb.ContextMenuPos()
	clr := lb.Viewport.Pixels.At(pos.X, pos.Y)
	lb.CurBackgroundColor = colors.AsRGBA(clr)
}

// StyleLabel does label styling -- it sets the StyMu Lock
func (lb *Label) StyleLabel() {
	lb.StyMu.Lock()
	defer lb.StyMu.Unlock()

	lb.Style2DWidget()
}

func (lb *Label) LayoutLabel() {
	lb.StyMu.RLock()
	defer lb.StyMu.RUnlock()

	lb.Style.BackgroundColor.Color = colors.Transparent // always use transparent bg for actual text
	lb.Render.SetHTML(lb.Text, lb.Style.FontRender(), &lb.Style.Text, &lb.Style.UnContext, lb.CSSAgg)
	spc := lb.BoxSpace()
	sz := lb.LayState.SizePrefOrMax()
	if !sz.IsNil() {
		sz.SetSub(spc.Size())
	}
	lb.Render.LayoutStdLR(&lb.Style.Text, lb.Style.FontRender(), &lb.Style.UnContext, sz)
}

func (lb *Label) Style2D() {
	lb.StyleLabel()
	lb.StyMu.Lock()
	lb.LayState.SetFromStyle(&lb.Style) // also does reset
	lb.StyMu.Unlock()
	lb.LayoutLabel()
}

func (lb *Label) Size2D(iter int) {
	if iter > 0 && lb.Style.Text.HasWordWrap() {
		return // already updated in previous iter, don't redo!
	} else {
		lb.InitLayout2D()
		sz := lb.LayState.Size.Pref // SizePrefOrMax()
		sz = sz.Max(lb.Render.Size)
		lb.Size2DFromWH(sz.X, sz.Y)
	}
}

func (lb *Label) Layout2D(parBBox image.Rectangle, iter int) bool {
	lb.Layout2DBase(parBBox, true, iter)
	lb.Layout2DChildren(iter) // todo: maybe shouldn't call this on known terminals?
	sz := lb.Size2DSubSpace()
	lb.Style.BackgroundColor.Color = colors.Transparent // always use transparent bg for actual text
	lb.Render.SetHTML(lb.Text, lb.Style.FontRender(), &lb.Style.Text, &lb.Style.UnContext, lb.CSSAgg)
	lb.Render.LayoutStdLR(&lb.Style.Text, lb.Style.FontRender(), &lb.Style.UnContext, sz)
	if lb.Style.Text.HasWordWrap() {
		if lb.Render.Size.Y < (sz.Y - 1) { // allow for numerical issues
			lb.LayState.SetFromStyle(&lb.Style)
			lb.Size2DFromWH(lb.Render.Size.X, lb.Render.Size.Y)
			return true // needs a redo!
		}
	}
	return false
}

func (lb *Label) TextPos() mat32.Vec2 {
	lb.StyMu.RLock()
	pos := lb.LayState.Alloc.Pos.Add(lb.Style.BoxSpace().Pos())
	lb.StyMu.RUnlock()
	return pos
}

func (lb *Label) RenderLabel() {
	lb.GrabCurBackgroundColor()
	rs, _, st := lb.RenderLock()
	defer lb.RenderUnlock(rs)
	lb.RenderPos = lb.TextPos()
	lb.RenderStdBox(st)
	lb.Render.Render(rs, lb.RenderPos)
}

func (lb *Label) Render2D() {
	if lb.FullReRenderIfNeeded() {
		return
	}
	if lb.PushBounds() {
		lb.This().(Node2D).ConnectEvents2D()
		lb.RenderLabel()
		lb.Render2DChildren()
		lb.PopBounds()
	} else {
		lb.DisconnectAllEvents(RegPri)
	}
}

func (lb *Label) ConnectEvents2D() {
	lb.WidgetEvents()
	lb.LabelEvents()
}
