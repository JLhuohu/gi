// Code generated by "stringer -output stringer.go -type=Actions"; DO NOT EDIT.

package osevent

import (
	"errors"
	"strconv"
)

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[OpenFiles-0]
	_ = x[ActionsN-1]
}

const _Actions_name = "OpenFilesActionsN"

var _Actions_index = [...]uint8{0, 9, 17}

func (i Actions) String() string {
	if i < 0 || i >= Actions(len(_Actions_index)-1) {
		return "Actions(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Actions_name[_Actions_index[i]:_Actions_index[i+1]]
}

func (i *Actions) FromString(s string) error {
	for j := 0; j < len(_Actions_index)-1; j++ {
		if s == _Actions_name[_Actions_index[j]:_Actions_index[j+1]] {
			*i = Actions(j)
			return nil
		}
	}
	return errors.New("String: " + s + " is not a valid option for type: Actions")
}

var _Actions_descMap = map[Actions]string{
	0: `OpenFiles means the user indicated that the app should open file(s) stored in Files`,
	1: ``,
}

func (i Actions) Desc() string {
	if str, ok := _Actions_descMap[i]; ok {
		return str
	}
	return "Actions(" + strconv.FormatInt(int64(i), 10) + ")"
}
