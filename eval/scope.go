package eval

import (
	"context"

	"github.com/acorn-io/aml/parser/ast"
)

type Scope struct {
	Parent    *Scope
	Reference Reference
	Name      string
}

func (s *Scope) Push(ref Reference, name string) *Scope {
	return &Scope{
		Parent:    s,
		Reference: ref,
		Name:      name,
	}
}

type ScopeLookup struct {
	Position ast.Position
	Scope    *Scope
	Key      string
}

func (s ScopeLookup) ResolveKey(ctx context.Context, key string) (Value, error) {
	//TODO implement me
	panic("implement me")
}

func (s ScopeLookup) Resolve(ctx context.Context) (Value, error) {
	//TODO implement me
	panic("implement me")
}
