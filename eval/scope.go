package eval

import (
	"context"
	"fmt"

	"github.com/acorn-io/aml/parser/ast"
)

type ErrKeyNotFound struct {
	Key string
}

func (e *ErrKeyNotFound) Error() string {
	return "key not found: " + e.Key
}

type Scope struct {
	Parent          *Scope
	SecondaryParent *Scope
	Value           Value
	cycleVars       map[string]bool
}

func NewScope(val Value) *Scope {
	return &Scope{Value: val}
}

func (s *Scope) Disallow(keys ...*ast.Literal) *Scope {
	cycleVars := map[string]bool{}
	for _, key := range keys {
		if key != nil {
			cycleVars[key.Value] = true
		}
	}
	return &Scope{
		Parent:          s.Parent,
		SecondaryParent: s.SecondaryParent,
		Value:           s.Value,
		cycleVars:       cycleVars,
	}
}

func (s *Scope) Merge(newParent *Scope) *Scope {
	if s == newParent {
		return s
	}
	return &Scope{
		Parent:          newParent,
		SecondaryParent: s,
	}
}

func (s *Scope) Lookup(ctx context.Context, key string) (Value, bool, error) {
	var (
		val, parentVal, secondaryParentVal Value
		ok, parentOk, secondayParentOk     bool
		err                                error
	)
	tick(ctx)

	if s.cycleVars[key] {
		return nil, false, fmt.Errorf("cycle looking up key %s", key)
	}

	if s.Value != nil {
		val, ok, err = s.Value.Lookup(ctx, key)
		if err != nil {
			return nil, false, err
		}
		if ok {
			return val, true, nil
		}
	}

	if s.Parent != nil {
		parentVal, parentOk, err = s.Parent.Lookup(ctx, key)
		if err != nil {
			return nil, false, err
		}
	}
	if s.SecondaryParent != nil {
		secondaryParentVal, secondayParentOk, err = s.SecondaryParent.Lookup(ctx, key)
		if err != nil {
			return nil, false, err
		}
	}

	if !parentOk {
		return secondaryParentVal, secondayParentOk, nil
	} else if !secondayParentOk {
		return parentVal, parentOk, nil
	}

	v, err := Merge(ctx, ast.Position{}, secondaryParentVal, parentVal)
	return v, true, err
}

func (s *Scope) Push(val Value) *Scope {
	return &Scope{
		Parent: s,
		Value:  val,
	}
}
