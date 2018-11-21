package tangle_test

import (
	"errors"
	"math"
	"sync"
	"testing"

	tangle "tangle/tangle2"
	"tangle/tangle2/store"

	test "github.com/advanderveer/go-test"
)

func checkCommit(t *testing.T, tx tangle.StoreTx) {
	err := tx.Commit()
	test.Ok(t, err)
}

func TestRandomWalk(t *testing.T) {
	s := store.NewSimple()
	g := tangle.NewGraph(42)
	tx := s.NewTransaction(true)
	defer checkCommit(t, tx)

	g.Append(tx, 0, []byte{})
	/**/ g.Append(tx, 1, []byte{0x0A}, 0)
	/**/ g.Append(tx, 2, []byte{0x0B}, 0)
	/**/ g.Append(tx, 3, []byte{0x0C}, 0)
	/**/ g.Append(tx, 4, []byte{0x0D}, 0)
	/*  */ g.Append(tx, 5, []byte{0x1A}, 4)
	/*  */ g.Append(tx, 6, []byte{0x1B}, 4)
	/*  */ g.Append(tx, 7, []byte{0x1C}, 4)
	/*    */ g.Append(tx, 8, []byte{0x2A}, 7)
	/*    */ g.Append(tx, 9, []byte{0x2A}, 7)

	t.Run("front to back depth-first", func(t *testing.T) {
		distFirstSplit := map[uint64]int{}
		for i := 0; i < 10; i++ {
			rw := []uint64{}
			test.Ok(t, g.Walk(tx, []uint64{0}, g.RevChildrenWRS, true, func(bid uint64, d []byte, m tangle.Meta, la []uint64) (err error) {
				rw = append(rw, bid)
				return
			}))

			distFirstSplit[rw[1]]++
		}

		//mostly picked weighted direction
		test.Equals(t, map[uint64]int{4: 9, 2: 1}, distFirstSplit)
	})
}

func TestStoreChildrenWRS(t *testing.T) {
	s := store.NewSimple()
	g := tangle.NewGraph(42)
	tx := s.NewTransaction(true)
	defer checkCommit(t, tx)

	//start with equal distribution
	g.Append(tx, 0, []byte{})
	g.Append(tx, 1, []byte{0x0A}, 0)
	g.Append(tx, 2, []byte{0x0B}, 0)
	g.Append(tx, 3, []byte{0x0C}, 0)
	g.Append(tx, 4, []byte{0x0D}, 0)

	t.Run("equal distribution selection", func(t *testing.T) {
		dist1th := map[uint64]int{}
		for i := 0; i < 100; i++ {
			ch := g.ChildrenWRS(tx, 0)
			dist1th[ch[0]]++
		}

		var tot int
		for _, n := range dist1th {
			tot += n
		}

		test.Equals(t, 100, tot)
		test.Equals(t, map[uint64]int{1: 27, 2: 23, 3: 20, 4: 30}, dist1th)
	})

	t.Run("biased children selection", func(t *testing.T) {
		for i := uint64(0); i < 100; i++ {
			g.Append(tx, 5+i, []byte{0xAA}, 2) //bias random walk
		}

		dist1th := map[uint64]int{}
		dist2th := map[uint64]int{}
		dist3th := map[uint64]int{}
		dist4th := map[uint64]int{}
		for i := 0; i < 4; i++ {
			ch := g.ChildrenWRS(tx, 0)
			dist1th[ch[0]]++
			dist2th[ch[1]]++
			dist3th[ch[2]]++
			dist4th[ch[2]]++
		}

		test.Equals(t, 1, len(dist1th))
		test.Equals(t, 3, len(dist2th))
		test.Equals(t, 3, len(dist3th))
		test.Equals(t, 3, len(dist4th))
	})
}

func TestStoreLinearBlocks(t *testing.T) {
	s := store.NewSimple()
	g := tangle.NewGraph(42)

	n := uint64(100) //insert this many blocks

	t.Run("should add 100 blocks in lineaur shape", func(t *testing.T) {
		tx := s.NewTransaction(true)
		defer checkCommit(t, tx)

		for i := uint64(0); i < n; i++ {
			if i > 0 {
				g.Append(tx, i, []byte{0x01}, i-1)
			} else {
				g.Append(tx, i, []byte{0x01})
			}
		}

		var visited []uint64
		var height uint64
		test.Ok(t, g.Walk(tx, []uint64{0}, g.Children, false, func(bid uint64, d []byte, m tangle.Meta, la []uint64) (err error) {
			if bid == 0 {
				test.Equals(t, uint64(99), m.Weight)
			}

			visited = append(visited, bid)
			height = m.Height
			return
		}))

		test.Equals(t, 100, len(visited))
		test.Equals(t, uint64(99), height)
	})
}

func TestStoreFanOutConcurrentBlockPut(t *testing.T) {
	n := uint64(100) //insert this many blocks

	s := store.NewSimple()
	g := tangle.NewGraph(42)

	t.Run("add genesis", func(t *testing.T) {
		tx := s.NewTransaction(true)
		defer checkCommit(t, tx)

		g.Append(tx, math.MaxUint64, []byte{0x01})

		tips := g.Tips(tx)
		test.Equals(t, 1, len(tips))
		test.Equals(t, uint64(math.MaxUint64), tips[0])
		test.Equals(t, uint64(0), g.Weight(tx, math.MaxUint64))
	})

	t.Run("should add 100 blocks concurrently", func(t *testing.T) {
		var wg sync.WaitGroup //insert 100 block concurrently
		defer wg.Wait()

		for i := uint64(0); i < n; i++ {
			wg.Add(1)
			go func(i uint64) {
				defer wg.Done()

				b := []byte{0x01}
				func(i uint64) { //append blocks
					var err error

					tx := s.NewTransaction(true)
					g.Append(tx, i, b, math.MaxUint64)

					err = tx.Commit()
					test.Ok(t, err)
				}(i)

				func(i uint64) { //read blocks
					tx := s.NewTransaction(true)
					defer checkCommit(t, tx)

					b2 := g.Get(tx, i)
					test.Equals(t, b, b2)
				}(i)
			}(i)
		}
	})

	t.Run("should correctly report parents and children", func(t *testing.T) {
		tx := s.NewTransaction(true)
		defer checkCommit(t, tx)

		test.Equals(t, int(n), len(g.Tips(tx))) //test tip
		test.Equals(t, 1, len(g.Parents(tx, 0)))
		test.Equals(t, uint64(math.MaxUint64), g.Parents(tx, 0)[0])
		test.Equals(t, 100, len(g.Children(tx, math.MaxUint64)))
		test.Equals(t, uint64(100), g.Weight(tx, math.MaxUint64))
	})

	t.Run("should correctly walk graph", func(t *testing.T) {
		tx := s.NewTransaction(true)
		defer checkCommit(t, tx)

		t.Run("front to back", func(t *testing.T) {
			var f2b []uint64 //walk front 2 back
			test.Ok(t, g.Walk(tx, g.Tips(tx), g.Parents, false, func(bid uint64, d []byte, m tangle.Meta, la []uint64) (err error) {
				f2b = append(f2b, bid)
				return
			}))

			test.Equals(t, int(n+1), len(f2b))
			test.Equals(t, uint64(math.MaxUint64), f2b[n]) //should have visited genesis last
		})

		t.Run("front to back depth-first", func(t *testing.T) {
			var f2b []uint64 //walk front 2 back
			test.Ok(t, g.Walk(tx, g.Tips(tx), g.Parents, true, func(bid uint64, d []byte, m tangle.Meta, la []uint64) (err error) {
				f2b = append(f2b, bid)
				return
			}))

			test.Equals(t, int(n+1), len(f2b))
			test.Equals(t, uint64(math.MaxUint64), f2b[1]) //should have visited as second
		})

		t.Run("back to front", func(t *testing.T) {
			var b2f []uint64 //walk back to front
			height := uint64(math.MaxUint64)
			test.Ok(t, g.Walk(tx, []uint64{math.MaxUint64}, g.Children, false, func(bid uint64, d []byte, m tangle.Meta, la []uint64) (err error) {
				b2f = append(b2f, bid)
				height = m.Height
				return
			}))

			test.Equals(t, int(n+1), len(b2f))
			test.Equals(t, uint64(math.MaxUint64), b2f[0]) //should have visited genesis first
			test.Equals(t, uint64(1), height)
		})

		t.Run("err walk", func(t *testing.T) {
			testErr := errors.New("test error")
			var errv []uint64 //walk back to front
			height := uint64(math.MaxUint64)
			test.Equals(t, testErr, g.Walk(tx, []uint64{math.MaxUint64}, g.Children, false, func(bid uint64, d []byte, m tangle.Meta, la []uint64) (err error) {
				errv = append(errv, bid)
				height = m.Height
				return testErr
			})) //should pass back walk error from func

			test.Equals(t, 1, len(errv))
			test.Equals(t, uint64(0), height)
		})
	})

}
