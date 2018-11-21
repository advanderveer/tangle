package tangle_test

import (
	"testing"

	tangle "tangle/tangle2"

	test "github.com/advanderveer/go-test"
)

func TestTipSelection(t *testing.T) {
	tngl := tangle.NewTangle()
	g := tngl.Genesis()
	test.Equals(t, 2, len(g))
	test.Equals(t, uint64(1), g[0])
	test.Equals(t, uint64(2), g[1])

	tips := tngl.SelectTips(2, 100)
	test.Equals(t, uint64(1), tips[0])
	test.Equals(t, uint64(2), tips[1])
}
