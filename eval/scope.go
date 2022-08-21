package eval

import (
	"context"
)

type ErrKeyNotFound struct {
	Key string
}

func (e *ErrKeyNotFound) Error() string {
	return "key not found: " + e.Key
}

type Scope struct {
	Parent *Scope
	Value  Value
}

func (s *Scope) Merge(other *Scope) *Scope {

}

func (s *Scope) Lookup(ctx context.Context, key string) (Value, error) {
	var (
		val Value
		ok  bool
		err error = &ErrKeyNotFound{
			Key: key,
		}
	)
	tick(ctx)
	if s.Value != nil {
		val, ok, err = s.Value.Lookup(ctx, key)
		if err != nil {
			return nil, err
		}
	}
	if !ok && s.Parent != nil {
		return s.Parent.Lookup(ctx, key)
	}
	return val, err
}

func (s *Scope) Push(val Value) *Scope {
	return &Scope{
		Parent: s,
		Value:  val,
	}
}
