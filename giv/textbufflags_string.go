// Code generated by "stringer -type=TextBufFlags"; DO NOT EDIT.

package giv

import (
	"errors"
	"strconv"
)

var _ = errors.New("dummy error")

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[TextBufAutoSaving-25]
	_ = x[TextBufMarkingUp-26]
	_ = x[TextBufChanged-27]
	_ = x[TextBufFileModOk-28]
	_ = x[TextBufFlagsN-29]
}

const _TextBufFlags_name = "TextBufAutoSavingTextBufMarkingUpTextBufChangedTextBufFileModOkTextBufFlagsN"

var _TextBufFlags_index = [...]uint8{0, 17, 33, 47, 63, 76}

func (i TextBufFlags) String() string {
	i -= 25
	if i < 0 || i >= TextBufFlags(len(_TextBufFlags_index)-1) {
		return "TextBufFlags(" + strconv.FormatInt(int64(i+25), 10) + ")"
	}
	return _TextBufFlags_name[_TextBufFlags_index[i]:_TextBufFlags_index[i+1]]
}

func StringToTextBufFlags(s string) (TextBufFlags, error) {
	for i := 0; i < len(_TextBufFlags_index)-1; i++ {
		if s == _TextBufFlags_name[_TextBufFlags_index[i]:_TextBufFlags_index[i+1]] {
			return TextBufFlags(i + 25), nil
		}
	}
	return 0, errors.New("String: " + s + " is not a valid option for type: TextBufFlags")
}
