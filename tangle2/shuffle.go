package tangle

import (
	"math"
	"math/rand"
)

//@TODO we use this to shuffle random walks through the graph which has cyptographic
//significance without me knowing what i'm doing here. Code taken and adapted from
//stdlib int63n: https://golang.org/src/math/rand/rand.go#L319
func uint64n(rnd *rand.Rand, n uint64) uint64 {
	if n <= 0 {
		panic("invalid argument to uint64n")
	}
	if n&(n-1) == 0 { // n is power of two, can mask
		return rnd.Uint64() & (n - 1)
	}
	max := uint64((1 << 63) - 1 - (1<<63)%uint64(n))
	v := rnd.Uint64()
	for v > max {
		v = rnd.Uint64()
	}
	return v % n
}

//@TODO we use this to shuffle random walks through the graph which has cyptographic
//significance without me knowing what i'm doing here, code inspired from:
//https://medium.com/@peterkellyonline/weighted-random-selection-3ff222917eb6
func pickWeightedID(rnd *rand.Rand, ids, weights []uint64) (i int) {
	if len(ids) != len(weights) {
		panic("number of weights must equal number of ids")
	}

	var tot uint64
	for _, w := range weights {
		ntot := tot + w
		if ntot < tot {
			panic("weights too big, wraps around maxuint64")
		}

		tot = ntot
	}

	if tot == 0 {
		return -1
	}

	r := uint64n(rnd, tot) //pick from [0, tot)
	for i := range ids {
		r -= weights[i]
		if r > tot || r == math.MaxUint64 { //wrapped around
			return i
		}
	}

	panic("unable to pick weighted random id")
}

//@TODO we use this for selecting during a random walk which influences the security
//of the network while I don't know exactly what i'm doing here, code inspired by:
//http://nicky.vanforeest.com/probability/weightedRandomShuffling/weighted.html
func weightedShuffle(rnd *rand.Rand, ids, weights []uint64) (shuffled []uint64) {
	shuffled = make([]uint64, len(ids))
	for i := 0; i < len(ids); i++ {
		j := pickWeightedID(rnd, ids, weights)
		if j == -1 {
			panic("unable to pick weighted id, all zero")
		}

		shuffled[i] = ids[j]
		weights[j] = 0
	}

	return
}
