package tangle_test

import (
	"testing"

	tangle "tangle/tangle2"

	test "github.com/advanderveer/go-test"
)

func TestIterWithoutCrr(t *testing.T) {
	iter := tangle.NewIter(1, 2, 34)
	for iter.Next() {
		//should not loop infinitely
	}
}

func TestIter(t *testing.T) {
	iter := tangle.NewIter(1, 2, 34)
	var saw []uint64
	for iter.Next() {
		curr := iter.Curr()
		saw = append(saw, curr)

		if curr == 2 { //append on the fly
			iter.Append(20, 21)
		}

		if curr == 20 { //prepend on the fly
			iter.Prepend(44)
		}
	}

	test.Equals(t, []uint64{1, 2, 34, 20, 44, 21}, saw)
}
