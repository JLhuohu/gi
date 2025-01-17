// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/ki/v2/ki"
	"goki.dev/ki/v2/kit"
)

// todo: needs to be impl

// ClipPath is used for holding a path that renders as a clip path
type ClipPath struct {
	NodeBase
}

var TypeClipPath = kit.Types.AddType(&ClipPath{}, ki.Props{ki.EnumTypeFlag: gi.TypeNodeFlags})

// AddNewClipPath adds a new clippath to given parent node, with given name.
func AddNewClipPath(parent ki.Ki, name string) *ClipPath {
	return parent.AddNewChild(TypeClipPath, name).(*ClipPath)
}

func (g *ClipPath) SVGName() string { return "clippath" }

func (g *ClipPath) CopyFieldsFrom(frm any) {
	fr := frm.(*ClipPath)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
}
