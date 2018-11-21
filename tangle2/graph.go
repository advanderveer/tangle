package tangle

import (
	"errors"
	"math/rand"
	"sort"
	"sync"
)

var (
	//ErrSkipNext can returned to stop the walk
	ErrSkipNext = errors.New("skip next")
)

//Weight returns the weight of the provided block, if the provided block does'nt exist it panics
func (g *Graph) Weight(tx StoreTx, id uint64) (w uint64) {
	m, ok := tx.GetMeta(id)
	if !ok {
		panic("block doesn't exist")
	}

	return m.Weight
}

//Tips returns all blocks without parents
func (g *Graph) Tips(tx StoreTx) (tips []uint64) {
	for t := range tx.GetTips() {
		tips = append(tips, t)
	}

	return
}

//Append a new block to the DAG
func (g *Graph) Append(tx StoreTx, id uint64, data []byte, parents ...uint64) {
	_, ok := tx.GetData(id)
	if ok {
		panic("block already exists")
	}

	//set data and make new tips
	tx.SetData(id, data)
	tx.SetTip(id)

	//update edges and tipsier
	var height uint64
	for _, pid := range parents {
		pmeta, ok := tx.GetMeta(pid)
		if !ok {
			panic("parent (meta) doesn't exist")
		}

		//adjust new block height to be max of parent height
		nheight := pmeta.Height + 1
		if nheight > height {
			height = nheight
		}

		//if parent was part of tips, it is no longer
		if _, ok := tx.GetTips()[pid]; ok {
			tx.DelTip(pid)
		}

		//update edges
		tx.SetP2c(pid, append(tx.GetP2c(pid), id))
		tx.SetC2p(id, append(tx.GetC2p(id), pid))
	}

	//update weights for each block (in)directly referenced
	if err := g.Walk(tx, parents, g.Parents, false, func(id uint64, data []byte, m Meta, la []uint64) error {
		m.Weight++
		tx.SetMeta(id, m)
		return nil
	}); err != nil {
		panic("failed to update weights: " + err.Error())
	}

	//set this blocks meta
	tx.SetMeta(id, Meta{Height: height})
}

type nextFunc func(tx StoreTx, id uint64) []uint64                    //determine the next nodes
type walkFunc func(id uint64, data []byte, m Meta, la []uint64) error //execute for each node

//Walk the graph
func (g *Graph) Walk(tx StoreTx, f []uint64, nf nextFunc, depthFirst bool, wf walkFunc) (err error) {
	visited := make(map[uint64]struct{})
	frontier := NewIter(f...)

	for frontier.Next() {
		bid := frontier.Curr()
		if _, ok := visited[bid]; ok {
			continue
		}

		b, _ := tx.GetData(bid)
		if b == nil {
			panic("block doesn't exist")
		}

		m, _ := tx.GetMeta(bid)  //current blocks's meta
		lookahead := nf(tx, bid) //curernt block's lookahead

		err = wf(bid, b, m, lookahead)
		if err == ErrSkipNext {
			err = nil
			continue
		} else if err != nil {
			return err //return user error unmodified
		}

		visited[bid] = struct{}{}
		for _, n := range lookahead {
			if depthFirst {
				frontier.Prepend(n)
			} else {
				frontier.Append(n)
			}
		}
	}

	return
}

//Parents returns the parents of a given block
func (g *Graph) Parents(tx StoreTx, id uint64) (parents []uint64) {
	parents = tx.GetC2p(id)
	return
}

//Children returns the children of a given block
func (g *Graph) Children(tx StoreTx, id uint64) (children []uint64) {
	children = tx.GetP2c(id)
	return
}

//RevChildrenWRS returns the reversed randomly weighted shuffled children ids
func (g *Graph) RevChildrenWRS(tx StoreTx, id uint64) (children []uint64) {
	a := g.ChildrenWRS(tx, id)
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}

	return a
}

//ChildrenWRS returns childres randomly shuffled by their weighted (Weigthed Random Shuffle)
func (g *Graph) ChildrenWRS(tx StoreTx, id uint64) (children []uint64) {
	var (
		ids     []uint64
		weights []uint64
	)

	//@TODO concurrent randomness is hard, refactor this to not need locking down
	//else we created race conditions somehow
	g.rmu.Lock()

	children = g.Children(tx, id) //get children and sort to make deterministic
	sort.Slice(children, func(i, j int) bool {
		return children[i] < children[j]
	})

	for _, id := range children {
		ids = append(ids, id)
		weights = append(weights, g.Weight(tx, id)+1) //no zero weights allowed
	}

	sh := weightedShuffle(g.rnd, ids, weights)
	g.rmu.Unlock() //@TODO remove me
	return sh
}

//Get will return a block by its id or return nil if not found
func (g *Graph) Get(tx StoreTx, id uint64) (data []byte) {
	data, _ = tx.GetData(id)
	return data
}

//Meta information about a block
type Meta struct {
	Weight uint64
	Height uint64
}

//Graph stores blocks
type Graph struct {
	seed int64
	rnd  *rand.Rand
	rmu  sync.Mutex
}

//NewGraph initates a store
func NewGraph(seed int64) (g *Graph) {
	g = &Graph{
		seed: seed,
		rnd:  rand.New(rand.NewSource(seed)),
	}

	return
}
