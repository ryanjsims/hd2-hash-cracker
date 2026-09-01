package pattern

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/xypwn/hd2-hash-cracker/util"
)

var builtinVars = map[string]IrSegment{}

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
	"filter": func(s IrSegmentChoice, re string) (IrSegmentChoice, error) {
		r, err := regexp.Compile("^" + re + "$") // HACK
		if err != nil {
			return nil, err
		}
		var res IrSegmentChoice
		for _, c := range s {
			str, ok := c.(IrSegmentStr)
			if !ok {
				return nil, fmt.Errorf("expected choice to consist only of strings")
			}
			if r.MatchString(string(str)) {
				res = append(res, c)
			}
		}
		return res, nil
	},
	// split splits each string in a choice of strings by any of the
	// given delimiters. Eliminates any duplicates.
	"split": func(cs IrSegmentChoice, delims string) (IrSegmentChoice, error) {
		seen := make(map[string]struct{})
		var res IrSegmentChoice
		for _, c := range cs {
			str, ok := c.(IrSegmentStr)
			if !ok {
				return nil, fmt.Errorf("expected choice to consist only of strings")
			}
			for s := range util.SplitStringAnySeq(string(str), delims) {
				if _, exists := seen[s]; exists {
					continue
				}
				seen[s] = struct{}{}
				res = append(res, IrSegmentStr(s))
			}
		}
		return res, nil
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
	"wordlist": (func(filename string) (IrSegmentChoice, error))(nil),

	// =====================
	// End special functions
	// =====================
}
