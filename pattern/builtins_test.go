package pattern

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuiltins(t *testing.T) {
	choice := func(s ...string) IrSegmentChoice {
		return ChoiceFromStrings(slices.Values(s))
	}
	t.Run("limit", func(t *testing.T) {
		require := require.New(t)
		fn := builtinFuncs["limit"].(func(s IrSegmentChoice, n string) (IrSegmentChoice, error))
		res, err := fn(choice("a", "b", "c"), "2")
		require.NoError(err)
		require.Equal(choice("a", "b"), res)
	})
	t.Run("filter", func(t *testing.T) {
		require := require.New(t)
		fn := builtinFuncs["filter"].(func(choices IrSegmentChoice, re string) (IrSegmentChoice, error))
		res, err := fn(choice("a", "b", "1", "2"), `\d+`)
		require.NoError(err)
		require.Equal(choice("1", "2"), res)
	})
	t.Run("remove", func(t *testing.T) {
		require := require.New(t)
		fn := builtinFuncs["remove"].(func(choices IrSegmentChoice, re string) (IrSegmentChoice, error))
		res, err := fn(choice("a", "b", "1", "2"), `\d+`)
		require.NoError(err)
		require.Equal(choice("a", "b"), res)
	})
	t.Run("split", func(t *testing.T) {
		require := require.New(t)
		fn := builtinFuncs["split"].(func(choices IrSegmentChoice, delims string) (IrSegmentChoice, error))
		res, err := fn(choice("a_b", "c:d_e:b"), `_:`)
		require.NoError(err)
		require.Equal(choice("a", "b", "c", "d", "e"), res)
	})
}
