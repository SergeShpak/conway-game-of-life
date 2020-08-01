package engine

import (
	"fmt"
	"testing"
)

func Test_getCoordWithWrap(t *testing.T) {
	cases := []struct {
		coord int32
		len   int32
		out   int32
	}{
		{
			coord: 5,
			len:   7,
			out:   5,
		},
		{
			coord: 10,
			len:   0,
			out:   0,
		},
		{
			coord: 10,
			len:   1,
			out:   0,
		},
		{
			coord: 15,
			len:   3,
			out:   0,
		},
		{
			coord: 16,
			len:   3,
			out:   1,
		},
		{
			coord: -11,
			len:   3,
			out:   1,
		},
	}
	for i, tc := range cases {
		i, tc := i, tc
		t.Run(fmt.Sprintf("running test case #%d", i), func(t *testing.T) {
			actualOut := normalizeDim(tc.coord, tc.len)
			if actualOut != tc.out {
				t.Fatalf("expected out %d, actual out %d", tc.out, actualOut)
			}
		})
	}
}
