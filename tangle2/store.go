package tangle

import "sync"

//Store persists graph data
type Store struct {
	meta map[uint64]Meta     //keep (local) metadata about blocks
	tips map[uint64]struct{} //keep orphan blocks as tips
	data map[uint64][]byte   //holds block data
	p2c  map[uint64][]uint64 //map parent -> child
	c2p  map[uint64][]uint64 //map children -> parents
	mu   sync.Mutex
}

//NewStore initiates the store
func NewStore() (s *Store) {
	s = &Store{
		meta: make(map[uint64]Meta),
		tips: make(map[uint64]struct{}),
		data: make(map[uint64][]byte),
		p2c:  make(map[uint64][]uint64),
		c2p:  make(map[uint64][]uint64),
	}
	return
}

//NewTransaction starts a store transaction
func (s *Store) NewTransaction() (tx *StoreTx) {
	s.mu.Lock()         //unlocked by commiting the transaction
	tx = &StoreTx{s: s} //@TODO seed with random bytes
	return
}

//StoreTx is an atomic interaction with the graph store
type StoreTx struct {
	s *Store
}

func (tx *StoreTx) getMeta(id uint64) (m Meta, ok bool) {
	m, ok = tx.s.meta[id]
	return
}

func (tx *StoreTx) getData(id uint64) (d []byte, ok bool) {
	d, ok = tx.s.data[id]
	return
}

func (tx *StoreTx) getTips() map[uint64]struct{} {
	return tx.s.tips
}

func (tx *StoreTx) setTip(id uint64) {
	tx.s.tips[id] = struct{}{}
}

func (tx *StoreTx) setData(id uint64, d []byte) {
	tx.s.data[id] = d
}

func (tx *StoreTx) setMeta(id uint64, m Meta) {
	tx.s.meta[id] = m
}

func (tx *StoreTx) delTip(id uint64) {
	delete(tx.s.tips, id)
}

func (tx *StoreTx) getP2c(id uint64) []uint64 {
	return tx.s.p2c[id]
}

func (tx *StoreTx) setP2c(id uint64, p2c []uint64) {
	tx.s.p2c[id] = p2c
}

func (tx *StoreTx) getC2p(id uint64) []uint64 {
	return tx.s.c2p[id]
}

func (tx *StoreTx) setC2p(id uint64, c2p []uint64) {
	tx.s.c2p[id] = c2p
}

//Commit the graph data
func (tx *StoreTx) Commit() (err error) {
	tx.s.mu.Unlock()
	return
}
