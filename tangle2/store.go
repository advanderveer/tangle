package tangle

//Store provides persistent storage
type Store interface {
	NewTransaction(update bool) StoreTx
}

//StoreTx provides ACID interactions with the store
type StoreTx interface {
	GetMeta(id uint64) (m Meta, ok bool)
	GetData(id uint64) (d []byte, ok bool)
	GetTips() map[uint64]struct{}
	SetTip(id uint64)
	SetData(id uint64, d []byte)
	SetMeta(id uint64, m Meta)
	DelTip(id uint64)
	GetP2c(id uint64) []uint64
	SetP2c(id uint64, p2c []uint64)
	GetC2p(id uint64) []uint64
	SetC2p(id uint64, c2p []uint64)
	Commit() (err error)
}
