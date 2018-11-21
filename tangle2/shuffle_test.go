package tangle

import (
	"math"
	"math/rand"
	"testing"

	test "github.com/advanderveer/go-test"
)

func TestUint64n(t *testing.T) {
	rnd := rand.New(rand.NewSource(42))

	for i := 0; i < 1000; i++ {
		test.Equals(t, uint64(0), uint64n(rnd, 1))
	}

	dist := map[uint64]int{}
	for i := 0; i < 1000; i++ {
		dist[uint64n(rnd, 2)]++
	}

	test.Equals(t, 496, dist[0])
	test.Equals(t, 504, dist[1])

	dist = map[uint64]int{}
	for i := 0; i < 1000; i++ {
		dist[uint64n(rnd, 3)]++
	}

	test.Equals(t, 353, dist[0])
	test.Equals(t, 335, dist[1])
	test.Equals(t, 312, dist[2])
}

func TestPickWeightedID(t *testing.T) {
	rnd := rand.New(rand.NewSource(42))

	t.Run("equal weights", func(t *testing.T) {
		dist := map[int]int{}
		for i := 0; i < 100; i++ {
			id := pickWeightedID(rnd, []uint64{1, 2}, []uint64{1, 1})
			dist[id]++
		}

		test.Equals(t, 47, dist[0])
		test.Equals(t, 53, dist[1])
	})

	t.Run("zero weights", func(t *testing.T) {
		// dist := map[int]int{}
		for i := 0; i < 100; i++ {
			id := pickWeightedID(rnd, []uint64{1, 2}, []uint64{0, 0})
			test.Equals(t, -1, id)
			// dist[id]++
		}

		// test.Equals(t, 0, dist[0])
		// test.Equals(t, 0, dist[1])
	})

	t.Run("max weights", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("The code did not panic")
			}
		}()

		pickWeightedID(rnd, []uint64{1, 2}, []uint64{math.MaxUint64})    //should no panic
		pickWeightedID(rnd, []uint64{1, 2}, []uint64{math.MaxUint64, 1}) //should fail
	})

	t.Run("inequal weights", func(t *testing.T) {
		dist := map[int]int{}
		for i := 0; i < 100; i++ {
			id := pickWeightedID(rnd, []uint64{1, 2}, []uint64{3, 1})
			dist[id]++
		}

		test.Equals(t, 68, dist[0])
		test.Equals(t, 32, dist[1])
	})

	t.Run("inequal weights, 1 is zero", func(t *testing.T) {
		dist := map[int]int{}
		for i := 0; i < 100; i++ {
			id := pickWeightedID(rnd, []uint64{1, 2}, []uint64{101, 0})
			dist[id]++
		}

		test.Equals(t, 100, dist[0])
		test.Equals(t, 0, dist[1])
	})
}

func TestWeightedShuffle(t *testing.T) {
	rnd := rand.New(rand.NewSource(44))

	t.Run("equal weights", func(t *testing.T) {
		dist1th := map[uint64]int{}
		dist2th := map[uint64]int{}
		for i := 0; i < 100; i++ {
			shuffle := weightedShuffle(rnd, []uint64{1, 2}, []uint64{1, 1})
			dist1th[shuffle[0]]++
			dist2th[shuffle[1]]++
		}

		test.Equals(t, map[uint64]int{1: 52, 2: 48}, dist1th)
		test.Equals(t, map[uint64]int{2: 52, 1: 48}, dist2th)
	})

	t.Run("nonequal weights", func(t *testing.T) {
		dist1th := map[uint64]int{}
		dist2th := map[uint64]int{}
		for i := 0; i < 100; i++ {
			shuffle := weightedShuffle(rnd, []uint64{1, 2}, []uint64{100, 1})
			dist1th[shuffle[0]]++
			dist2th[shuffle[1]]++
		}

		test.Equals(t, map[uint64]int{1: 99, 2: 1}, dist1th)
		test.Equals(t, map[uint64]int{2: 99, 1: 1}, dist2th)
	})

	t.Run("for parts", func(t *testing.T) {
		for i := 0; i < 10000; i++ {
			shuffle := weightedShuffle(rnd, []uint64{1, 2}, []uint64{1, 1})
			if (shuffle[0] == 2 && shuffle[1] == 2) || (shuffle[0] == 1 && shuffle[1] == 1) {
				t.Fatal("shuffle caused same number twice")
			}
		}

	})
}

// func TestStoreChildrenWRS(t *testing.T) {
//
// 	t.Run("pick wheighted", func(t *testing.T) {
// 		rnd := rand.New(rand.NewSource(42))
//
// 		t.Run("equal wheights", func(t *testing.T) {
// 			dist := map[uint64]int{}
// 			ids := []uint64{1, 2}
// 			weights := []uint64{100, 100}
// 			for i := 0; i < 100; i++ {
// 				pick := tangle.PickWeightedID(rnd, ids, weights)
// 				dist[pick]++
// 			}
//
// 			test.Equals(t, 55, dist[1])
// 			test.Equals(t, 45, dist[2]) //about equal distribution
//
// 		})
//
// 		t.Run("inequal wheights", func(t *testing.T) {
// 			dist := map[uint64]int{}
// 			ids := []uint64{1, 2}
// 			weights := []uint64{100, 3}
// 			for i := 0; i < 100; i++ {
// 				pick := tangle.PickWeightedID(rnd, ids, weights)
// 				dist[pick]++
// 			}
//
// 			test.Equals(t, 95, dist[1])
// 			test.Equals(t, 5, dist[2]) //about equal distribution
// 		})
//
// 		//the algorithm described here https://medium.com/@peterkellyonline/weighted-random-selection-3ff222917eb6
// 		//had the weird edge case that a weight of 1 would caue the distribution to always flip to first dist having everything
// 		t.Run("1 weight edge case", func(t *testing.T) {
// 			dist := map[uint64]int{}
// 			ids := []uint64{1, 2}
// 			weights := []uint64{10, 1}
// 			for i := 0; i < 100; i++ {
// 				pick := tangle.PickWeightedID(rnd, ids, weights)
// 				dist[pick]++
// 			}
//
// 			test.Equals(t, 89, dist[1])
// 			test.Equals(t, 11, dist[2]) //about equal distribution
// 		})
//
// 		t.Run("max int weights", func(t *testing.T) {
// 			dist := map[uint64]int{}
// 			ids := []uint64{1, 2}
// 			weights := []uint64{math.MaxUint64 / 4, math.MaxUint64 / 4, 1} //just about full
// 			for i := 0; i < 100; i++ {
// 				pick := tangle.PickWeightedID(rnd, ids, weights)
// 				dist[pick]++
// 			}
//
// 			test.Equals(t, 54, dist[1])
// 			test.Equals(t, 46, dist[2]) //about equal distribution
// 		})
//
// 		t.Run("one 0 weights", func(t *testing.T) {
// 			dist := map[uint64]int{}
// 			ids := []uint64{1, 2}
// 			weights := []uint64{0, 1} //just about full
// 			for i := 0; i < 10000; i++ {
// 				pick := tangle.PickWeightedID(rnd, ids, weights)
// 				dist[pick]++
// 			}
//
// 			test.Equals(t, 0, dist[1])
// 			test.Equals(t, 10000, dist[2]) //all to 10
// 		})
//
// 		t.Run("all 0 weights", func(t *testing.T) {
// 			dist := map[uint64]int{}
// 			ids := []uint64{1, 2}
// 			weights := []uint64{0, 0} //just about full
// 			for i := 0; i < 100; i++ {
// 				pick := tangle.PickWeightedID(rnd, ids, weights)
// 				dist[pick]++
// 			}
//
// 			test.Equals(t, 51, dist[1])
// 			test.Equals(t, 49, dist[2]) //about equal distribution
// 		})
//
// 	})
//
// 	s, err := tangle.NewStore()
// 	test.Ok(t, err)
//
// 	tx := s.NewTransaction()
// 	defer checkCommit(t, tx)
//
// 	//start with equal distribution
// 	tx.Append(0, []byte{})
// 	tx.Append(1, []byte{0x0A}, 0)
// 	tx.Append(2, []byte{0x0B}, 0)
// 	tx.Append(3, []byte{0x0C}, 0)
// 	tx.Append(4, []byte{0x0D}, 0)
//
// 	dist := map[uint64]int{}
// 	for i := 0; i < 100; i++ {
// 		ch := tx.ChildrenWRS(0)
// 		dist[ch[0]]++
// 	}
//
// 	fmt.Println(dist)
//
// 	for i := uint64(0); i < 100; i++ {
// 		tx.Append(5+i, []byte{0xAA}, 2) //bias random shuffle towards always picking
// 	}
//
// 	dist = map[uint64]int{}
// 	for i := 0; i < 1; i++ {
// 		ch := tx.ChildrenWRS(0)
// 		dist[ch[0]]++
// 		fmt.Println(ch)
// 	}
//
// 	fmt.Println(dist)
//
// }
