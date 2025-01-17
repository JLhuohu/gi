// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gist"
	"goki.dev/gi/v2/icons"
	"goki.dev/ki/v2/ki"
	"goki.dev/ki/v2/kit"
	"goki.dev/vci/v2"
)

// VCSLogView is a view of the variables
type VCSLogView struct {
	gi.Layout

	// current log
	Log vci.Log `desc:"current log"`

	// file that this is a log of -- if blank then it is entire repository
	File string `desc:"file that this is a log of -- if blank then it is entire repository"`

	// date expression for how long ago to include log entries from
	Since string `desc:"date expression for how long ago to include log entries from"`

	// version control system repository
	Repo vci.Repo `json:"-" xml:"-" copy:"-" desc:"version control system repository"`

	// revision A -- defaults to HEAD
	RevA string `desc:"revision A -- defaults to HEAD"`

	// revision B -- blank means current working copy
	RevB string `desc:"revision B -- blank means current working copy"`

	// double-click will set the A revision -- else B
	SetA bool `desc:"double-click will set the A revision -- else B"`
}

var TypeVCSLogView = kit.Types.AddType(&VCSLogView{}, VCSLogViewProps)

func (lv *VCSLogView) OnInit() {
	lv.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		s.SetStretchMax()
	})
}

func (lv *VCSLogView) OnChildAdded(child ki.Ki) {
	if w := gi.KiAsWidget(child); w != nil {
		switch w.Name() {
		case "a-tf", "b-tf":
			w.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
				s.Width.SetEm(12)
			})
		}
	}
}

// Config configures to given repo, log and file (file could be empty)
func (lv *VCSLogView) Config(repo vci.Repo, lg vci.Log, file, since string) {
	lv.Repo = repo
	lv.Log = lg
	lv.File = file
	lv.Since = since
	lv.Lay = gi.LayoutVert
	config := kit.TypeAndNameList{}
	config.Add(gi.TypeToolBar, "toolbar")
	config.Add(TypeTableView, "log")
	mods, updt := lv.ConfigChildren(config)
	tv := lv.TableView()
	if mods {
		lv.RevA = "HEAD"
		lv.RevB = ""
		lv.SetA = true
		lv.ConfigToolBar()
		tv.SliceViewSig.Connect(lv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(SliceViewDoubleClicked) {
				idx := data.(int)
				if idx >= 0 && idx < len(lv.Log) {
					cmt := lv.Log[idx]
					if lv.File != "" {
						if lv.SetA {
							lv.SetRevA(cmt.Rev)
						} else {
							lv.SetRevB(cmt.Rev)
						}
						lv.ToggleRev()
					}
					cinfo, err := lv.Repo.CommitDesc(cmt.Rev, false)
					if err == nil {
						TextViewDialog(lv.ViewportSafe(), cinfo, DlgOpts{Title: "Commit Info: " + cmt.Rev, Ok: true})
					}
				}
			}
		})
	} else {
		updt = lv.UpdateStart()
	}
	tv.SetDisabled()
	tv.SetSlice(&lv.Log)
	lv.UpdateEnd(updt)
}

// SetRevA sets the RevA to use
func (lv *VCSLogView) SetRevA(rev string) {
	lv.RevA = rev
	tb := lv.ToolBar()
	tfi := tb.ChildByName("a-tf", 2)
	if tfi == nil {
		return
	}
	tfi.(*gi.TextField).SetText(rev)
}

// SetRevB sets the RevB to use
func (lv *VCSLogView) SetRevB(rev string) {
	lv.RevB = rev
	tb := lv.ToolBar()
	tfi := tb.ChildByName("b-tf", 2)
	if tfi == nil {
		return
	}
	tfi.(*gi.TextField).SetText(rev)
}

// ToggleRev switches the active revision to set
func (lv *VCSLogView) ToggleRev() {
	tb := lv.ToolBar()
	updt := tb.UpdateStart()
	cba := tb.ChildByName("a-rev", 2).(*gi.CheckBox)
	cbb := tb.ChildByName("b-rev", 2).(*gi.CheckBox)
	lv.SetA = !lv.SetA
	cba.SetChecked(lv.SetA)
	cbb.SetChecked(!lv.SetA)
	tb.UpdateEnd(updt)
}

// ToolBar returns the toolbar
func (lv *VCSLogView) ToolBar() *gi.ToolBar {
	return lv.ChildByName("toolbar", 0).(*gi.ToolBar)
}

// TableView returns the tableview
func (lv *VCSLogView) TableView() *TableView {
	return lv.ChildByName("log", 1).(*TableView)
}

// ConfigToolBar
func (lv *VCSLogView) ConfigToolBar() {
	tb := lv.ToolBar()
	if lv.File != "" {
		gi.AddNewLabel(tb, "fl", "File: "+DirAndFile(lv.File))
		tb.AddSeparator("flsep")
		cba := gi.AddNewCheckBox(tb, "a-rev")
		cba.SetText("A Rev: ")
		cba.Tooltip = "If selected, double-clicking in log will set this A Revision to use for Diff"
		cba.SetChecked(true)
		tfa := gi.AddNewTextField(tb, "a-tf")
		tfa.SetText(lv.RevA)
		tfa.TextFieldSig.Connect(lv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.TextFieldDone) || sig == int64(gi.TextFieldDeFocused) {
				lv.RevA = tfa.Text()
			}
		})
		tb.AddSeparator("absep")
		cbb := gi.AddNewCheckBox(tb, "b-rev")
		cbb.SetText("B Rev: ")
		cbb.Tooltip = "If selected, double-clicking in log will set this B Revision to use for Diff"
		tfb := gi.AddNewTextField(tb, "b-tf")
		tfb.SetText(lv.RevB)
		tfb.TextFieldSig.Connect(lv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.TextFieldDone) || sig == int64(gi.TextFieldDeFocused) {
				lv.RevB = tfb.Text()
			}
		})
		tb.AddSeparator("dsep")
		tb.AddAction(gi.ActOpts{Label: "Diff", Icon: icons.Difference, Tooltip: "Show the diffs between two revisions -- if blank, A is current HEAD, and B is current working copy"}, lv.This(),
			func(recv, send ki.Ki, sig int64, data any) {
				lvv := recv.Embed(TypeVCSLogView).(*VCSLogView)
				DiffViewDialogFromRevs(lvv.ViewportSafe(), lvv.Repo, lvv.File, nil, lvv.RevA, lvv.RevB)
			})

		cba.ButtonSig.Connect(lv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.ButtonToggled) {
				lv.SetA = cba.IsChecked()
				cbb.SetChecked(!lv.SetA)
				cbb.UpdateSig()
			}
		})
		cbb.ButtonSig.Connect(lv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.ButtonToggled) {
				lv.SetA = !cbb.IsChecked()
				cba.SetChecked(lv.SetA)
				cba.UpdateSig()
			}
		})
	}

}

// VCSLogViewProps are style properties for DebugView
var VCSLogViewProps = ki.Props{
	ki.EnumTypeFlag: gi.TypeNodeFlags,
}

// VCSLogViewDialog opens a VCS Log View for given repo, log and file (file could be empty)
func VCSLogViewDialog(repo vci.Repo, lg vci.Log, file, since string) *gi.Dialog {
	title := "VCS Log: "
	if file == "" {
		title += "All files"
	} else {
		title += DirAndFile(file)
	}
	if since != "" {
		title += " since: " + since
	}
	dlg := gi.NewStdDialog(gi.DlgOpts{Title: title}, gi.NoOk, gi.NoCancel)
	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	lv := frame.InsertNewChild(TypeVCSLogView, prIdx+1, "vcslog").(*VCSLogView)
	lv.Viewport = dlg.Embed(gi.TypeViewport2D).(*gi.Viewport2D)
	lv.Config(repo, lg, file, since)

	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, nil, nil)
	return dlg
}
