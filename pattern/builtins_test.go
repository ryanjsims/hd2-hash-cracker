package pattern

import (
	"reflect"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuiltins(t *testing.T) {
	choice := func(s ...string) IrSegmentChoice {
		return ChoiceFromStrings(slices.Values(s))
	}
	testCases := []struct {
		testName    string
		builtinName string
		args        []any
		want        IrSegment
	}{
		{"limit", "limit",
			[]any{choice("a", "b", "c"), "2"},
			choice("a", "b")},
		{"filter", "filter",
			[]any{choice("a", "b", "1", "2"), `\d+`},
			choice("1", "2")},
		{"remove", "remove",
			[]any{choice("a", "b", "1", "2"), `\d+`},
			choice("a", "b")},
		{"replace", "replace",
			[]any{choice("hello world 123", "hello dave"), `hello (\w+)`, "$1"},
			choice("world 123", "dave")},
		{"split", "split",
			[]any{choice("a_b", "c:d_e:b"), "_:"},
			choice("a", "b", "c", "d", "e")},
		{"prefixes", "prefixes",
			[]any{choice("a/b/c", "0/1", "a/b_d", "_x"), "/_"},
			choice("a", "a/b", "a/b/c", "0", "0/1", "a/b_d", "_x")},
		{"suffixes", "suffixes",
			[]any{choice("a/b/c", "0/1", "a/b_d", "y/", "/z"), "/_"},
			choice("c", "b/c", "a/b/c", "1", "0/1", "d", "b_d", "a/b_d", "y/", "z", "/z")},
		{"merge", "merge",
			[]any{choice("a", "b", "c"), choice("c", "b", "x")},
			choice("a", "b", "c", "x")},
		{"merge_2", "merge",
			[]any{choice("a", "b", "c", "d"), choice("c", "b", "xx"), choice("xx", "yy", "zz")},
			choice("a", "b", "c", "d", "xx", "yy", "zz")},
	}
	for _, c := range testCases {
		rArgs := make([]reflect.Value, len(c.args))
		for i := range c.args {
			rArgs[i] = reflect.ValueOf(c.args[i])
		}

		fn := reflect.ValueOf(builtinFuncs[c.builtinName])

		t.Run(c.testName, func(t *testing.T) {
			require := require.New(t)
			res := fn.Call(rArgs)
			if len(res) > 0 {
				require.Equal(2, len(res))
				require.True(res[1].Type().Implements(reflect.TypeFor[error]()), "2nd return type must implement error")
				if !res[1].IsNil() {
					require.NoError(res[1].Interface().(error))
				}
			}
			require.Equal(res[0].Interface(), c.want)
		})
	}
}
