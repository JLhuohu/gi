// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !windows && !3d
// +build !windows,!3d

package windriver

import (
	"fmt"
	"runtime"

	"goki.dev/gi/v2/oswin"
	"goki.dev/gi/v2/oswin/driver/internal/errapp"
)

// Main is called by the program's main function to run the graphical
// application.
//
// It calls f on the App, possibly in a separate goroutine, as some OS-
// specific libraries require being on 'the main thread'. It returns when f
// returns.
func Main(f func(oswin.App)) {
	f(errapp.Stub(fmt.Errorf(
		"windriver: unsupported GOOS/GOARCH %s/%s", runtime.GOOS, runtime.GOARCH)))
}
