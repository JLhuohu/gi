// Code generated by "stringer -type=KeyFuns"; DO NOT EDIT.

package gi

import (
	"fmt"
	"strconv"
)

const _KeyFuns_name = "KeyFunNilKeyFunMoveUpKeyFunMoveDownKeyFunMoveRightKeyFunMoveLeftKeyFunPageUpKeyFunPageDownKeyFunHomeKeyFunEndKeyFunDocHomeKeyFunDocEndKeyFunWordRightKeyFunWordLeftKeyFunFocusNextKeyFunFocusPrevKeyFunEnterKeyFunAcceptKeyFunCancelSelectKeyFunSelectModeKeyFunSelectAllKeyFunAbortKeyFunCopyKeyFunCutKeyFunPasteKeyFunPasteHistKeyFunBackspaceKeyFunBackspaceWordKeyFunDeleteKeyFunDeleteWordKeyFunKillKeyFunDuplicateKeyFunUndoKeyFunRedoKeyFunInsertKeyFunInsertAfterKeyFunGoGiEditorKeyFunZoomOutKeyFunZoomInKeyFunPrefsKeyFunRefreshKeyFunRecenterKeyFunCompleteKeyFunSearchKeyFunFindKeyFunReplaceKeyFunJumpKeyFunHistPrevKeyFunHistNextKeyFunWinFocusNextKeyFunMenuNewKeyFunMenuNewAlt1KeyFunMenuNewAlt2KeyFunMenuOpenKeyFunMenuOpenAlt1KeyFunMenuOpenAlt2KeyFunMenuSaveKeyFunMenuSaveAsKeyFunMenuSaveAltKeyFunMenuCloseKeyFunMenuCloseAlt1KeyFunMenuCloseAlt2KeyFunsN"

var _KeyFuns_index = [...]uint16{0, 9, 21, 35, 50, 64, 76, 90, 100, 109, 122, 134, 149, 163, 178, 193, 204, 216, 234, 250, 265, 276, 286, 295, 306, 321, 336, 355, 367, 383, 393, 408, 418, 428, 440, 457, 473, 486, 498, 509, 522, 536, 550, 562, 572, 585, 595, 609, 623, 641, 654, 671, 688, 702, 720, 738, 752, 768, 785, 800, 819, 838, 846}

func (i KeyFuns) String() string {
	if i < 0 || i >= KeyFuns(len(_KeyFuns_index)-1) {
		return "KeyFuns(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _KeyFuns_name[_KeyFuns_index[i]:_KeyFuns_index[i+1]]
}

func (i *KeyFuns) FromString(s string) error {
	for j := 0; j < len(_KeyFuns_index)-1; j++ {
		if s == _KeyFuns_name[_KeyFuns_index[j]:_KeyFuns_index[j+1]] {
			*i = KeyFuns(j)
			return nil
		}
	}
	return fmt.Errorf("String %v is not a valid option for type KeyFuns", s)
}
