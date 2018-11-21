package store

import (
	"sync"

	tangle "tangle/tangle2"
)

//Simple persists graph data
type Simple struct {
	meta map[uint64]tangle.Meta //keep (local) metadata about blocks
	tips map[uint64]struct{}    //keep orphan blocks as tips
	data map[uint64][]byte      //holds block data
	p2c  map[uint64][]uint64    //map parent -> child
	c2p  map[uint64][]uint64    //map children -> parents

	mu sync.RWMutex
}

//NewSimple initiates the store
func NewSimple() (s *Simple) {
	s = &Simple{
		meta: make(map[uint64]tangle.Meta),
		tips: make(map[uint64]struct{}),
		data: make(map[uint64][]byte),
		p2c:  make(map[uint64][]uint64),
		c2p:  make(map[uint64][]uint64),
	}

	return
}

//NewTransaction starts a store transaction
func (s *Simple) NewTransaction(update bool) tangle.StoreTx {
	tx := &SimpleTx{s: s, update: update}
	if tx.update {
		tx.s.mu.Lock()
	} else {
		tx.s.mu.RLock()
	}

	return tx
}

//SimpleTx is an atomic interaction with the graph store
type SimpleTx struct {
	s      *Simple
	update bool
}

//GetMeta gets a blocks metadata
func (tx *SimpleTx) GetMeta(id uint64) (m tangle.Meta, ok bool) {
	m, ok = tx.s.meta[id]
	return
}

//GetData gets data of a given node
func (tx *SimpleTx) GetData(id uint64) (d []byte, ok bool) {
	d, ok = tx.s.data[id]
	return
}

//GetTips gets the current tips
func (tx *SimpleTx) GetTips() map[uint64]struct{} {
	return tx.s.tips
}

//SetTip sets the provided id as a tip
func (tx *SimpleTx) SetTip(id uint64) {
	tx.s.tips[id] = struct{}{}
}

//SetData sets the block data
func (tx *SimpleTx) SetData(id uint64, d []byte) {
	tx.s.data[id] = d
}

//SetMeta sets the metadata for a block
func (tx *SimpleTx) SetMeta(id uint64, m tangle.Meta) {
	tx.s.meta[id] = m
}

//DelTip deletes the 'id' as tip
func (tx *SimpleTx) DelTip(id uint64) {
	delete(tx.s.tips, id)
}

//GetP2c gets the parent to child edges
func (tx *SimpleTx) GetP2c(id uint64) []uint64 {
	return tx.s.p2c[id]
}

//SetP2c sets the parent to child edges
func (tx *SimpleTx) SetP2c(id uint64, p2c []uint64) {
	tx.s.p2c[id] = p2c
}

//GetC2p gets the child to parent edges
func (tx *SimpleTx) GetC2p(id uint64) []uint64 {
	return tx.s.c2p[id]
}

//SetC2p sets the child to parent edges
func (tx *SimpleTx) SetC2p(id uint64, c2p []uint64) {
	tx.s.c2p[id] = c2p
}

//Commit the store transaction
func (tx *SimpleTx) Commit() (err error) {
	if tx.update {
		tx.s.mu.Unlock()
	} else {
		tx.s.mu.RUnlock()
	}

	return
}
