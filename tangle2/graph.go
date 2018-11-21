package tangle

import (
	"errors"
	"math/rand"
	"sort"
)

var (
	//ErrSkipNext can returned to stop the walk
	ErrSkipNext = errors.New("skip next")
)

//GraphTx provides an atomic iteraction on our graph
type GraphTx struct {
	rnd *rand.Rand
	stx *StoreTx
}

//Weight returns the weight of the provided block, if the provided block does'nt exist it panics
func (tx *GraphTx) Weight(id uint64) (w uint64) {
	m, ok := tx.stx.getMeta(id)
	if !ok {
		panic("block doesn't exist")
	}

	return m.Weight
}

//Tips returns all blocks without parents
func (tx *GraphTx) Tips() (tips []uint64) {
	for t := range tx.stx.getTips() {
		tips = append(tips, t)
	}

	return
}

//Append a new block to the DAG
func (tx *GraphTx) Append(id uint64, data []byte, parents ...uint64) {
	_, ok := tx.stx.getData(id)
	if ok {
		panic("block already exists")
	}

	//set data and make new tips
	tx.stx.setData(id, data)
	tx.stx.setTip(id)

	//update edges and tipsier
	var height uint64
	for _, pid := range parents {
		pmeta, ok := tx.stx.getMeta(pid)
		if !ok {
			panic("parent (meta) doesn't exist")
		}

		//adjust new block height to be max of parent height
		nheight := pmeta.Height + 1
		if nheight > height {
			height = nheight
		}

		//if parent was part of tips, it is no longer
		if _, ok := tx.stx.getTips()[pid]; ok {
			tx.stx.delTip(pid)
		}

		//update edges
		tx.stx.setP2c(pid, append(tx.stx.getP2c(pid), id))
		tx.stx.setC2p(id, append(tx.stx.getC2p(id), pid))
	}

	//update weights for each block (in)directly referenced
	if err := tx.Walk(parents, tx.Parents, false, func(id uint64, data []byte, m Meta) error {
		m.Weight++
		tx.stx.setMeta(id, m)
		return nil
	}); err != nil {
		panic("failed to update weights: " + err.Error())
	}

	//set this blocks meta
	tx.stx.setMeta(id, Meta{Height: height})
}

type nextFunc func(id uint64) []uint64                   //determine the next nodes
type walkFunc func(id uint64, data []byte, m Meta) error //execute for each node

//Walk the graph
func (tx *GraphTx) Walk(f []uint64, nf nextFunc, depthFirst bool, wf walkFunc) (err error) {
	visited := make(map[uint64]struct{})
	frontier := NewIter(f...)

	for frontier.Next() {
		bid := frontier.Curr()
		if _, ok := visited[bid]; ok {
			continue
		}

		b, _ := tx.stx.getData(bid)
		if b == nil {
			panic("block doesn't exist")
		}

		m, _ := tx.stx.getMeta(bid)
		err = wf(bid, b, m)
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
func (tx *GraphTx) Parents(id uint64) (parents []uint64) {
	parents = tx.stx.getC2p(id)
	return
}

//Children returns the children of a given block
func (tx *GraphTx) Children(id uint64) (children []uint64) {
	children = tx.stx.getP2c(id)
	return
}

//RevChildrenWRS returns the reversed randomly weighted shuffled children ids
func (tx *GraphTx) RevChildrenWRS(id uint64) (children []uint64) {
	a := tx.ChildrenWRS(id)
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}

	return a
}

//ChildrenWRS returns childres randomly shuffled by their weighted (Weigthed Random Shuffle)
func (tx *GraphTx) ChildrenWRS(id uint64) (children []uint64) {
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
func (tx *GraphTx) Get(id uint64) (data []byte) {
	if err := tx.Walk([]uint64{id}, nil, false, func(bid uint64, d []byte, m Meta) (err error) {
		data = d
		return ErrSkipNext
	}); err != nil {
		panic("error while walking for single node: " + err.Error())
	}

	return
}

//Meta information about a block
type Meta struct {
	Weight uint64
	Height uint64
}

//Graph stores blocks
type Graph struct {
	seed  int64
	store *Store
}

//NewGraph initates a store
func NewGraph(seed int64) (s *Graph, err error) {
	s = &Graph{
		seed:  seed,
		store: NewStore(),
	}

	return
}

//NewTransaction creates ACID store transaction
func (s *Graph) NewTransaction() (tx *GraphTx) {
	tx = &GraphTx{
		rnd: rand.New(rand.NewSource(s.seed)),
		stx: s.store.NewTransaction(),
	}

	return
}

//Commit the transaction
func (tx *GraphTx) Commit() (err error) {
	return tx.stx.Commit()
}
