package tangle

import (
	"fmt"
	"io"
	"sort"
	"sync/atomic"
)

//Tangle is our consensus data structure
type Tangle struct {
	graph   *Graph
	store   *Store
	genesis []uint64
	idc     uint64
}

//NewTangle initiates a tangle
func NewTangle() (t *Tangle) {
	t = &Tangle{graph: NewGraph(42), store: NewStore()}

	tx := t.store.NewTransaction()
	defer t.mustCommit(tx)
	t.genesis = []uint64{ //add 2 genesis blocks
		t.receiveBlock(tx, []byte{0x01}),
		t.receiveBlock(tx, []byte{0x02}),
	}

	return
}

//Genesis blocks begin the tangle
func (t *Tangle) Genesis() []uint64 {
	return t.genesis
}

//Draw the tangle, mainly for debugging purposes
func (t *Tangle) Draw(w io.Writer) (err error) {
	tx := t.store.NewTransaction()
	defer t.mustCommit(tx)

	fmt.Fprintln(w, `digraph {`)
	if err := t.graph.Walk(tx, t.genesis, t.graph.RevChildrenWRS, true, func(id uint64, data []byte, m Meta, la []uint64) error {
		fmt.Fprintf(w, "\t"+`"%d" [shape=box];`+"\n", id)

		for _, l := range la {
			fmt.Fprintf(w, "\t"+`"%d" -> "%d";`+"\n", id, l)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to walk: %v", err)
	}

	fmt.Fprintln(w, `}`)
	return
}

//SelectTips will peform the tip selection until we have 'n' unique or ran the
//algorithm 'max' times whatever happens first
func (t *Tangle) SelectTips(n, max int) (tips []uint64) {
	tx := t.store.NewTransaction()
	defer t.mustCommit(tx)
	return t.selectTips(tx, n, max)
}

func (t *Tangle) selectTips(tx *StoreTx, n, max int) (tips []uint64) {
	utips := map[uint64]struct{}{}
	for i := 0; i < max; i++ {
		if len(utips) >= n {
			break
		}

		//perform a dept-first children traveral with weighted selection
		if err := t.graph.Walk(tx, t.genesis, t.graph.RevChildrenWRS, true, func(id uint64, data []byte, m Meta, la []uint64) error {
			//@TODO perform validation
			//@TODO also add tips that are not completely on the front line
			if len(la) == 0 {
				utips[id] = struct{}{} //add as tip
			}

			return nil
		}); err != nil {
			panic("failed the walk to find a tip: " + err.Error())
		}
	}

	for t := range utips {
		tips = append(tips, t)
	}

	sort.Slice(tips, func(i, j int) bool { return tips[i] < tips[j] })
	return
}

//ReceiveBlock with take data and
func (t *Tangle) ReceiveBlock(d []byte, parents ...uint64) (id uint64) {
	tx := t.store.NewTransaction()
	defer t.mustCommit(tx)
	return t.receiveBlock(tx, d, parents...)
}

func (t *Tangle) receiveBlock(tx *StoreTx, d []byte, parents ...uint64) (id uint64) {
	//@TODO add deduplication and verification
	//@TODO this seems racy, id should probably be retrieved from storage layer
	atomic.AddUint64(&t.idc, 1)
	id = atomic.LoadUint64(&t.idc)
	t.graph.Append(tx, id, d, parents...)
	return
}

func (t *Tangle) mustCommit(tx *StoreTx) {
	err := tx.Commit()
	if err != nil { //@TODO handle this propertly
		panic("failed to commit: " + err.Error())
	}
}
