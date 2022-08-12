package eval

import (
	"context"
	"fmt"

	"github.com/acorn-io/aml/parser/ast"
)

type Expression struct {
	Position ast.Position
	Scope    *Scope

	expr *ast.Expression
	ref  Reference
}

func (e *Expression) Type() (Type, error) {
	if err := e.process(); err != nil {
		return "", err
	}
	return e.ref.Type()
}

type opChain struct {
	Op *ast.Op
	Reference
}

func processMerge(chain []opChain) (result []opChain) {
	for _, op := range chain {
		switch op.Op.Op {
		case "&":
			result[len(result)-1].Reference = &Merge{
				Position: op.Op.Position,
				Left:     result[len(result)-1].Reference,
				Right:    op.Reference,
			}
		default:
			result = append(result, op)
		}
	}
	return result
}

func processAnd(chain []opChain) (result []opChain) {
	for _, op := range chain {
		switch op.Op.Op {
		case "&&":
			fallthrough
		case "||":
			result[len(result)-1].Reference = &BinaryOp{
				Position: op.Op.Position,
				Op:       op.Op.Op,
				Left:     result[len(result)-1].Reference,
				Right:    op.Reference,
			}
		default:
			result = append(result, op)
		}
	}
	return result
}

func processAdd(chain []opChain) (result []opChain) {
	for _, op := range chain {
		switch op.Op.Op {
		case "+":
			fallthrough
		case "-":
			result[len(result)-1].Reference = &BinaryOp{
				Position: op.Op.Position,
				Op:       op.Op.Op,
				Left:     result[len(result)-1].Reference,
				Right:    op.Reference,
			}
		default:
			result = append(result, op)
		}
	}
	return result
}

func processMul(chain []opChain) (result []opChain) {
	for _, op := range chain {
		switch op.Op.Op {
		case "*":
			fallthrough
		case "/":
			result[len(result)-1].Reference = &BinaryOp{
				Position: op.Op.Position,
				Op:       op.Op.Op,
				Left:     result[len(result)-1].Reference,
				Right:    op.Reference,
			}
		default:
			result = append(result, op)
		}
	}
	return result
}

func processOps(chain []opChain) (Reference, error) {
	chain = processMerge(chain)
	chain = processMul(chain)
	chain = processAdd(chain)
	chain = processAnd(chain)
	if len(chain) > 1 {
		return nil, fmt.Errorf("invalid op chain non recognized op: %s", chain[1].Op.Op)
	}
	return chain[0].Reference, nil
}

func (e *Expression) process() error {
	if e.ref != nil {
		return nil
	}
	ref, err := selectorToReference(e.Scope, e.expr.Selector)
	if err != nil {
		return err
	}

	chain := []opChain{{Reference: ref, Op: &ast.Op{Position: e.expr.Selector.Position}}}
	for _, op := range e.expr.Operator {
		opRef, err := selectorToReference(e.Scope, op.Selector)
		if err != nil {
			return wrapErr(op.Selector.Position, err)
		}
		chain = append(chain, opChain{
			Op:        op.Op,
			Reference: opRef,
		})
	}

	ref, err = processOps(chain)
	if err != nil {
		return err
	}

	e.ref = ref
	return nil
}

func (e *Expression) Lookup(key string) (_ Reference, err error) {
	defer func() {
		err = wrapErr(e.Position, err)
	}()
	if err := e.process(); err != nil {
		return nil, err
	}
	return e.ref.Lookup(key)
}

func (e *Expression) Resolve(ctx context.Context) (_ Value, err error) {
	defer func() {
		err = wrapErr(e.Position, err)
	}()
	if err := e.process(); err != nil {
		return nil, err
	}
	return e.ref.Resolve(ctx)
}

func selectorToReference(scope *Scope, sel *ast.Selector) (base Reference, err error) {
	if sel.Literal != nil {
		base, err = scope.Lookup(sel.Literal.Value)
		if err != nil {
			return nil, wrapErr(sel.Literal.Position, err)
		}
	} else if sel.Value != nil {
		base = toReference(scope, sel.Value)
	} else if sel.Parens != nil {
		base = toReference(scope, sel.Parens.Value)
	} else {
		return nil, wrapErr(sel.Position, fmt.Errorf("invalid selector no selection set"))
	}

	for _, l := range sel.Lookup {
		base, err = base.Lookup(l.Literal.Value)
		if err != nil {
			return nil, wrapErr(l.Literal.Position, fmt.Errorf("failed to find key %s: %w", l.Literal.Value, err))
		}
	}

	if sel.Not {
		base = &Not{
			Position: sel.Position,
			ref:      base,
		}
	}

	return base, nil
}
