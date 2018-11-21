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

//Tx provides an atomic iteraction on our block store
type Tx struct {
	s   *Graph
	rnd *rand.Rand
}

//Weight returns the weight of the provided block, if the provided block does'nt exist it panics
func (tx *Tx) Weight(id uint64) (w uint64) {
	m, ok := tx.s.meta[id]
	if !ok {
		panic("block doesn't exist")
	}

	return m.Weight
}

//Tips returns all blocks without parents
func (tx *Tx) Tips() (tips []uint64) {
	for t := range tx.s.tips {
		tips = append(tips, t)
	}

	return
}

//Append a new block to the DAG
func (tx *Tx) Append(id uint64, data []byte, parents ...uint64) {
	_, ok := tx.s.data[id]
	if ok {
		panic("block already exists")
	}

	//set data and make new tips
	tx.s.data[id] = data
	tx.s.tips[id] = struct{}{}

	//update edges and tipsier
	var height uint64
	for _, pid := range parents {
		pmeta, ok := tx.s.meta[pid]
		if !ok {
			panic("parent (meta) doesn't exist")
		}

		//adjust new block height to be max of parent height
		nheight := pmeta.Height + 1
		if nheight > height {
			height = nheight
		}

		//if parent was part of tips, it is no longer
		if _, ok := tx.s.tips[pid]; ok {
			delete(tx.s.tips, pid)
		}

		//update edges
		tx.s.p2c[pid] = append(tx.s.p2c[pid], id)
		tx.s.c2p[id] = append(tx.s.c2p[id], pid)
	}

	//update weights for each block (in)directly referenced
	if err := tx.Walk(parents, tx.Parents, false, func(id uint64, data []byte, m Meta) error {
		m.Weight++
		tx.s.meta[id] = m
		return nil
	}); err != nil {
		panic("failed to update weights: " + err.Error())
	}

	//set this blocks meta
	tx.s.meta[id] = Meta{Height: height}
}

type nextFunc func(id uint64) []uint64                   //determine the next nodes
type walkFunc func(id uint64, data []byte, m Meta) error //execute for each node

//Walk the graph
func (tx *Tx) Walk(f []uint64, nf nextFunc, depthFirst bool, wf walkFunc) (err error) {
	visited := make(map[uint64]struct{})
	frontier := NewIter(f...)

	for frontier.Next() {
		bid := frontier.Curr()
		if _, ok := visited[bid]; ok {
			continue
		}

		b := tx.s.data[bid]
		if b == nil {
			panic("block doesn't exist")
		}

		err = wf(bid, b, tx.s.meta[bid])
		if err == ErrSkipNext {
			err = nil
			continue
		} else if err != nil {
			return err //return user error unmodified
		}

		visited[bid] = struct{}{}
		for _, n := range nf(bid) {
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
func (tx *Tx) Parents(id uint64) (parents []uint64) {
	parents = tx.s.c2p[id]
	return
}

//Children returns the children of a given block
func (tx *Tx) Children(id uint64) (children []uint64) {
	children = tx.s.p2c[id]
	return
}

//RevChildrenWRS returns the reversed randomly weighted shuffled children ids
func (tx *Tx) RevChildrenWRS(id uint64) (children []uint64) {
	a := tx.ChildrenWRS(id)
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}

	return a
}

//ChildrenWRS returns childres randomly shuffled by their weighted (Weigthed Random Shuffle)
func (tx *Tx) ChildrenWRS(id uint64) (children []uint64) {
	var (
		ids     []uint64
		weights []uint64
	)

	children = tx.Children(id) //get children and sort to make deterministic
	sort.Slice(children, func(i, j int) bool { return children[i] < children[j] })
	for _, id := range children {
		ids = append(ids, id)
		weights = append(weights, tx.Weight(id)+1) //no zero weights allowed
	}

	sh := weightedShuffle(tx.rnd, ids, weights)
	return sh
}

//Get will return a block by its id or return nil if not found
func (tx *Tx) Get(id uint64) (data []byte) {
	if err := tx.Walk([]uint64{id}, nil, false, func(bid uint64, d []byte, m Meta) (err error) {
		data = d
		return ErrSkipNext
	}); err != nil {
		panic("error while walking for single node: " + err.Error())
	}

	return
}

//Commit the transaction
func (tx *Tx) Commit() (err error) {
	tx.s.mu.Unlock()
	return
}

//Meta information about a block
type Meta struct {
	Weight uint64
	Height uint64
}

//Graph stores blocks
type Graph struct {
	meta map[uint64]Meta     //keep (local) metadata about blocks
	tips map[uint64]struct{} //keep orphan blocks as tips
	data map[uint64][]byte   //holds block data
	p2c  map[uint64][]uint64 //map parent -> child
	c2p  map[uint64][]uint64 //map children -> parents
	mu   sync.Mutex
	seed int64
}

//NewGraph initates a store
func NewGraph(seed int64) (s *Graph, err error) {
	s = &Graph{
		meta: make(map[uint64]Meta),
		tips: make(map[uint64]struct{}),
		data: make(map[uint64][]byte),
		p2c:  make(map[uint64][]uint64),
		c2p:  make(map[uint64][]uint64),
		seed: seed,
	}

	return
}

//NewTransaction creates ACID store transaction
func (s *Graph) NewTransaction() (tx *Tx) {
	s.mu.Lock()                                           //unlocked by commiting the transaction
	tx = &Tx{s: s, rnd: rand.New(rand.NewSource(s.seed))} //@TODO seed with random bytes
	return
}
