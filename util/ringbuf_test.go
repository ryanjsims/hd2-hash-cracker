package util

import (
	"bytes"
	"math"
	"math/rand"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func FuzzRingBufVsByteBuf(f *testing.F) {
	f.Fuzz(func(t *testing.T, seed int64) {
		require := require.New(t)
		rng := rand.New(rand.NewSource(seed))
		var rb RingBuf[byte]
		var b bytes.Buffer
		for i := range 100 {
			{
				n := rng.Intn(64)
				nums := make([]byte, n)
				for i := range nums {
					nums[i] = byte(rng.Intn(math.MaxUint8))
				}
				rb.Write(nums...)
				b.Write(nums)
			}
			require.Equal(b.Len(), rb.Len(), "iteration %d, rb %+v", i, rb)
			var rbData []byte
			{
				a, b := rb.PeekAll()
				rbData = append(slices.Clone(a), b...)
			}
			require.Equal(rb.Len(), len(rbData), "iteration %d, rb %+v", i, rb)
			if !(len(b.Bytes()) == 0 && len(rbData) == 0) { // ignore if b.Bytes() and rbData are unequal because one is nil and the other is empty
				require.Equal(b.Bytes(), rbData, "iteration %d, rb %+v", i, rb)
			}
			{
				n := min(rng.Intn(64), b.Len())
				s1 := make([]byte, n)
				s2 := make([]byte, n)
				b.Read(s1)
				for i := range s2 {
					s2[i] = rb.Read()
				}
				require.Equal(s1, s2)
			}
		}
	})
}

func TestRingBufBasic(t *testing.T) {
	require := require.New(t)
	var rb RingBuf[int]
	rb.GrowCap(6)
	rb.Write(1, 2, 3, 4, 5)
	require.Equal(RingBuf[int]{rp: 0, wp: 5, size: 5, b: []int{1, 2, 3, 4, 5, 0}}, rb)
	rb.Read()
	rb.Read()
	rb.Write(6, 7)
	require.Equal(RingBuf[int]{rp: 2, wp: 1, size: 5, b: []int{7, 2, 3, 4, 5, 6}}, rb)
	rb.Write(8, 9)
	require.Equal(RingBuf[int]{rp: 8, wp: 3, size: 7, b: []int{7, 8, 9, 4, 5, 6, 0, 0, 3, 4, 5, 6}}, rb)
}
