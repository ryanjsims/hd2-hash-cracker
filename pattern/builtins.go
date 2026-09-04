package pattern

import (
	"fmt"
	"iter"
	"regexp"
	"strconv"
	"strings"

	"github.com/xypwn/hd2-hash-cracker/util"
)

var builtinVars = map[string]IrSegment{}

func builtinHelperTransformChoiceOfStrings(
	choices IrSegmentChoice,
	transform func(choices iter.Seq2[int, string]) (res []IrSegment, err error),
) (IrSegmentChoice, error) {
	prevStr := ""
	for i, seg := range choices {
		s, ok := seg.(IrSegmentStr)
		if !ok {
			var sfx string
			if i > 0 {
				sfx = fmt.Sprintf(" (after %q)", prevStr)
			}
			return nil, fmt.Errorf("expected choice to consist only of strings, but got non-string at index %d%s", i, sfx)
		}
		prevStr = string(s)
	}
	newChoices, err := transform(func(yield func(int, string) bool) {
		for i, seg := range choices {
			if !yield(i, string(seg.(IrSegmentStr))) {
				break
			}
		}
	})
	if err != nil {
		return nil, err
	}
	return IrSegmentChoice(newChoices), nil
}

var builtinFuncs = map[string]any{
	// limit limits the number of choices to at most n, where
	// n must be a non-negative number.
	"limit": func(s IrSegmentChoice, n string) (IrSegmentChoice, error) {
		nInt, err := strconv.Atoi(n)
		if err != nil {
			return nil, err
		}
		if len(s) > nInt {
			s = s[:nInt]
		}
		return s, nil
	},
	// filter only keeps the choices that fully match the given regex.
	//
	// All choices must be strings.
	"filter": func(choices IrSegmentChoice, re string) (IrSegmentChoice, error) {
		r, err := regexp.Compile("^" + re + "$") // HACK
		if err != nil {
			return nil, err
		}
		return builtinHelperTransformChoiceOfStrings(choices, func(choices iter.Seq2[int, string]) (res []IrSegment, err error) {
			for _, s := range choices {
				if r.MatchString(s) {
					res = append(res, IrSegmentStr(s))
				}
			}
			return
		})
	},
	// Like filter, but removes all choices that fully match the given regex.
	"remove": func(choices IrSegmentChoice, re string) (IrSegmentChoice, error) {
		r, err := regexp.Compile("^" + re + "$") // HACK
		if err != nil {
			return nil, err
		}
		return builtinHelperTransformChoiceOfStrings(choices, func(choices iter.Seq2[int, string]) (res []IrSegment, err error) {
			for _, s := range choices {
				if !r.MatchString(s) {
					res = append(res, IrSegmentStr(s))
				}
			}
			return
		})
	},
	// split splits each string in a choice of strings by any of the
	// given delimiters. Eliminates any duplicates.
	"split": func(choices IrSegmentChoice, delims string) (IrSegmentChoice, error) {
		return builtinHelperTransformChoiceOfStrings(choices, func(choices iter.Seq2[int, string]) (res []IrSegment, err error) {
			seen := make(map[string]struct{})
			for _, choice := range choices {
				for s := range util.SplitStringAnySeq(choice, delims) {
					if _, exists := seen[s]; exists {
						continue
					}
					seen[s] = struct{}{}
					res = append(res, IrSegmentStr(s))
				}
			}
			return
		})
	},
	// returns a list of non-empty deduplicated prefixes of the input up to any of delims.
	//
	// Example: <a/b/c|0/1|a/b_d> -> <a|a/b|a/b/c|0|0/1|a/b_d>
	"prefixes": func(choices IrSegmentChoice, delims string) (IrSegmentChoice, error) {
		return builtinHelperTransformChoiceOfStrings(choices, func(choices iter.Seq2[int, string]) (res []IrSegment, err error) {
			seen := make(map[string]struct{})
			seen[""] = struct{}{}
			for _, s := range choices {
				sp := util.SplitStringAfterAny(s, delims)
				for i := 1; i <= len(sp); i++ {
					pfx := strings.Join(sp[:i], "")
					if len(pfx) > 0 && strings.ContainsRune(delims, rune(pfx[len(pfx)-1])) {
						pfx = pfx[:len(pfx)-1]
					}
					if _, ok := seen[pfx]; ok {
						continue
					}
					res = append(res, IrSegmentStr(pfx))
					seen[pfx] = struct{}{}
				}
			}
			return
		})
	},
	// returns a list of non-empty deduplicated suffixes of the input up to any of delims.
	//
	// Example: <a/b/c|0/1|x_b/c> -> <c|b/c|a/b/c|1|0/1|x_b/c>
	"suffixes": func(choices IrSegmentChoice, delims string) (IrSegmentChoice, error) {
		return builtinHelperTransformChoiceOfStrings(choices, func(choices iter.Seq2[int, string]) (res []IrSegment, err error) {
			seen := make(map[string]struct{})
			seen[""] = struct{}{}
			for _, s := range choices {
				sp := util.SplitStringAfterAny(s, delims)
				for i := len(sp) - 1; i >= 0; i-- {
					sfx := strings.Join(sp[i:], "")
					if _, ok := seen[sfx]; ok {
						continue
					}
					res = append(res, IrSegmentStr(sfx))
					seen[sfx] = struct{}{}
				}
			}
			return
		})
	},

	// =====================
	// Special functions
	// =====================
	// These have to be implemented in the parser, as they
	// modify the current parser state.

	// load loads the given pattern from a file.
	"load": (func(filename string) (IrSegment, error))(nil),
	// import imports any variables from the given
	// file.
	"import": (func(filename string) (IrSegment, error))(nil),
	// wordlist loads a wordlist from a file as a choice expression.
	//
	// Each non-empty line is counted as a word.
	//
	// Lines starting with "//" or "#" are ignored.
	"wordlist": (func(filename string) (IrSegmentChoice, error))(nil),

	// =====================
	// End special functions
	// =====================
}
