package tangle

//Iter is a block id iterator
type Iter struct {
	v []uint64
	c uint64
}

//NewIter create an id iterator
func NewIter(ids ...uint64) *Iter {
	return &Iter{v: ids}
}

//Append values, calling next will return these at the end now
func (i *Iter) Append(v ...uint64) {
	i.v = append(i.v, v...)
}

//Prepend values, calling next will immediately start returning them
func (i *Iter) Prepend(v ...uint64) {
	i.v = append(v, i.v...)
}

//Next advances the iterator, returns false when done
func (i *Iter) Next() (b bool) {
	if len(i.v) == 0 {
		return false
	}

	i.c, i.v = i.v[0], i.v[1:]
	return len(i.v) >= 0
}

//Curr returns the current iter position
func (i *Iter) Curr() (v uint64) {

	return i.c
}
