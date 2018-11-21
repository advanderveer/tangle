package tangle_test

import (
	"errors"
	"math"
	"sync"
	"testing"

	tangle "tangle/tangle2"

	test "github.com/advanderveer/go-test"
)

func checkCommit(t *testing.T, tx *tangle.GraphTx) {
	err := tx.Commit()
	test.Ok(t, err)
}

func TestRandomWalk(t *testing.T) {
	s, err := tangle.NewGraph(42)
	test.Ok(t, err)
	tx := s.NewTransaction()
	defer checkCommit(t, tx)

	tx.Append(0, []byte{})
	/**/ tx.Append(1, []byte{0x0A}, 0)
	/**/ tx.Append(2, []byte{0x0B}, 0)
	/**/ tx.Append(3, []byte{0x0C}, 0)
	/**/ tx.Append(4, []byte{0x0D}, 0)
	/*  */ tx.Append(5, []byte{0x1A}, 4)
	/*  */ tx.Append(6, []byte{0x1B}, 4)
	/*  */ tx.Append(7, []byte{0x1C}, 4)
	/*    */ tx.Append(8, []byte{0x2A}, 7)
	/*    */ tx.Append(9, []byte{0x2A}, 7)

	t.Run("front to back depth-first", func(t *testing.T) {

		distFirstSplit := map[uint64]int{}
		for i := 0; i < 10; i++ {
			rw := []uint64{}
			test.Ok(t, tx.Walk([]uint64{0}, tx.RevChildrenWRS, true, func(bid uint64, d []byte, m tangle.Meta) (err error) {
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
	s, err := tangle.NewGraph(42)
	test.Ok(t, err)

	tx := s.NewTransaction()
	defer checkCommit(t, tx)

	//start with equal distribution
	tx.Append(0, []byte{})
	tx.Append(1, []byte{0x0A}, 0)
	tx.Append(2, []byte{0x0B}, 0)
	tx.Append(3, []byte{0x0C}, 0)
	tx.Append(4, []byte{0x0D}, 0)

	t.Run("equal distribution selection", func(t *testing.T) {
		dist1th := map[uint64]int{}
		for i := 0; i < 100; i++ {
			ch := tx.ChildrenWRS(0)
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
			tx.Append(5+i, []byte{0xAA}, 2) //bias random walk
		}

		dist1th := map[uint64]int{}
		dist2th := map[uint64]int{}
		dist3th := map[uint64]int{}
		dist4th := map[uint64]int{}
		for i := 0; i < 4; i++ {
			ch := tx.ChildrenWRS(0)
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
	s, err := tangle.NewGraph(42)
	test.Ok(t, err)

	n := uint64(100) //insert this many blocks

	t.Run("should add 100 blocks in lineaur shape", func(t *testing.T) {
		tx := s.NewTransaction()
		defer checkCommit(t, tx)

		for i := uint64(0); i < n; i++ {
			if i > 0 {
				tx.Append(i, []byte{0x01}, i-1)
			} else {
				tx.Append(i, []byte{0x01})
			}
		}

		var visited []uint64
		var height uint64
		test.Ok(t, tx.Walk([]uint64{0}, tx.Children, false, func(bid uint64, d []byte, m tangle.Meta) (err error) {
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

	s, err := tangle.NewGraph(42)
	test.Ok(t, err)

	t.Run("add genesis", func(t *testing.T) {
		tx := s.NewTransaction()
		defer checkCommit(t, tx)

		tx.Append(math.MaxUint64, []byte{0x01})

		tips := tx.Tips()
		test.Equals(t, 1, len(tips))
		test.Equals(t, uint64(math.MaxUint64), tips[0])
		test.Equals(t, uint64(0), tx.Weight(math.MaxUint64))
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

					tx := s.NewTransaction()
					tx.Append(i, b, math.MaxUint64)

					err = tx.Commit()
					test.Ok(t, err)
				}(i)

				func(i uint64) { //read blocks
					tx := s.NewTransaction()
					defer checkCommit(t, tx)

					b2 := tx.Get(i)
					test.Equals(t, b, b2)
				}(i)
			}(i)
		}
	})

	t.Run("should correctly report parents and children", func(t *testing.T) {
		tx := s.NewTransaction()
		test.Ok(t, err)
		defer checkCommit(t, tx)

		test.Equals(t, int(n), len(tx.Tips())) //test tip
		test.Equals(t, 1, len(tx.Parents(0)))
		test.Equals(t, uint64(math.MaxUint64), tx.Parents(0)[0])
		test.Equals(t, 100, len(tx.Children(math.MaxUint64)))
		test.Equals(t, uint64(100), tx.Weight(math.MaxUint64))
	})

	t.Run("should correctly walk graph", func(t *testing.T) {
		tx := s.NewTransaction()
		defer checkCommit(t, tx)

		t.Run("front to back", func(t *testing.T) {
			var f2b []uint64 //walk front 2 back
			test.Ok(t, tx.Walk(tx.Tips(), tx.Parents, false, func(bid uint64, d []byte, m tangle.Meta) (err error) {
				f2b = append(f2b, bid)
				return
			}))

			test.Equals(t, int(n+1), len(f2b))
			test.Equals(t, uint64(math.MaxUint64), f2b[n]) //should have visited genesis last
		})

		t.Run("front to back depth-first", func(t *testing.T) {
			var f2b []uint64 //walk front 2 back
			test.Ok(t, tx.Walk(tx.Tips(), tx.Parents, true, func(bid uint64, d []byte, m tangle.Meta) (err error) {
				f2b = append(f2b, bid)
				return
			}))

			test.Equals(t, int(n+1), len(f2b))
			test.Equals(t, uint64(math.MaxUint64), f2b[1]) //should have visited as second
		})

		t.Run("back to front", func(t *testing.T) {
			var b2f []uint64 //walk back to front
			height := uint64(math.MaxUint64)
			test.Ok(t, tx.Walk([]uint64{math.MaxUint64}, tx.Children, false, func(bid uint64, d []byte, m tangle.Meta) (err error) {
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
			test.Equals(t, testErr, tx.Walk([]uint64{math.MaxUint64}, tx.Children, false, func(bid uint64, d []byte, m tangle.Meta) (err error) {
				errv = append(errv, bid)
				height = m.Height
				return testErr
			})) //should pass back walk error from func

			test.Equals(t, 1, len(errv))
			test.Equals(t, uint64(0), height)
		})
	})

}
