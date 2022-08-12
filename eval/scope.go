package eval

import (
	"errors"
)

type ErrKeyNotFound struct {
	Key string
}

func (e *ErrKeyNotFound) Error() string {
	return "key not found: " + e.Key
}

type Scope struct {
	Parent    *Scope
	Reference Reference
}

func (s *Scope) Lookup(key string) (Reference, error) {
	var (
		ref Reference
		err error = &ErrKeyNotFound{
			Key: key,
		}
	)
	if s.Reference != nil {
		ref, err = s.Reference.Lookup(key)
	}
	if k := (*ErrKeyNotFound)(nil); errors.As(err, &k) && s.Parent != nil {
		return s.Parent.Lookup(key)
	}
	return ref, err
}

func (s *Scope) Push(ref Reference) *Scope {
	return &Scope{
		Parent:    s,
		Reference: ref,
	}
}
