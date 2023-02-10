package eval

import (
	"context"
	"fmt"

	"github.com/acorn-io/aml/parser/ast"
)

type opChain struct {
	Op *ast.Op
	Value
}

type Acc func(ctx context.Context, scope *Scope, pos ast.Position, op string, left, right Value) (_ Value, err error)

func processOperators(ctx context.Context, scope *Scope, acc Acc, chain []opChain, operators ...string) (result []opChain, err error) {
	for _, op := range chain {
		found := false
		for _, operator := range operators {
			if op.Op.Token.Value == operator {
				result[len(result)-1].Value, err = acc(
					ctx,
					scope,
					op.Op.Position,
					op.Op.Token.Value,
					result[len(result)-1].Value,
					op.Value)
				found = true
				if err != nil {
					return nil, err
				}
			}
		}
		if !found {
			result = append(result, op)
		}
	}
	return result, nil
}

func processOps(ctx context.Context, scope *Scope, chain []opChain) (_ Value, err error) {
	chain, err = processOperators(ctx, scope, BinaryOp, chain, "*", "/")
	if err != nil {
		return nil, err
	}
	chain, err = processOperators(ctx, scope, BinaryOp, chain, "+", "-")
	if err != nil {
		return nil, err
	}
	chain, err = processOperators(ctx, scope, BinaryOp, chain, "==", "!=", "<", "<=", ">", ">=", "=~", "!~")
	if err != nil {
		return nil, err
	}
	chain, err = processOperators(ctx, scope, BinaryOp, chain, "&&")
	if err != nil {
		return nil, err
	}
	chain, err = processOperators(ctx, scope, BinaryOp, chain, "||")
	if err != nil {
		return nil, err
	}
	chain, err = processOperators(ctx, scope, merge, chain, "&")
	if err != nil {
		return nil, err
	}
	if len(chain) > 1 {
		return nil, fmt.Errorf("invalid op chain non recognized op: %s", chain[1].Op.Token.Value)
	}
	return chain[0].Value, nil
}

func EvaluateExpression(ctx context.Context, scope *Scope, expr *ast.Expression) (Value, error) {
	val, err := selectorToValue(ctx, scope, expr.Selector)
	if err != nil {
		return nil, err
	}

	chain := []opChain{{Value: val, Op: &ast.Op{Position: expr.Selector.Position}}}
	for _, op := range expr.Operator {
		opVal, err := selectorToValue(ctx, scope, op.Selector)
		if err != nil {
			return nil, wrapErr(op.Selector.Position, err)
		}
		chain = append(chain, opChain{
			Op:    op.Op,
			Value: opVal,
		})
	}

	val, err = processOps(ctx, scope, chain)
	if err != nil {
		return nil, err
	}

	return val, nil
}

func selectorToValue(ctx context.Context, scope *Scope, sel *ast.Selector) (base Value, err error) {
	if sel.Literal != nil {
		var ok bool
		base, ok, err = scope.Lookup(ctx, sel.Literal.Value)
		if err != nil {
			return nil, wrapErr(sel.Literal.Position, err)
		}
		if !ok {
			return nil, wrapErr(sel.Literal.Position, &ErrKeyNotFound{
				Key: sel.Literal.Value,
			})
		}
	} else if sel.Value != nil {
		base, err = ToValue(ctx, scope, sel.Value)
		if err != nil {
			return nil, wrapErr(sel.Value.Position, err)
		}
	} else if sel.Parens != nil {
		base, err = ToValue(ctx, scope, sel.Parens.Value)
		if err != nil {
			return nil, wrapErr(sel.Parens.Position, err)
		}
	} else {
		return nil, wrapErr(sel.Position, fmt.Errorf("invalid selector no selection set"))
	}

	var ok bool
	for _, l := range sel.Lookup {
		if l.Literal != nil {
			base, ok, err = base.Lookup(ctx, l.Literal.Value)
			if err != nil {
				return nil, wrapErr(l.Literal.Position, fmt.Errorf("failed to lookup key %s: %w", l.Literal.Value, err))
			}
			if !ok {
				return nil, wrapErr(l.Literal.Position, &ErrKeyNotFound{
					Key: l.Literal.Value,
				})
			}
		} else if l.Index != nil {
			v, err := EvaluateExpression(ctx, scope, l.Index)
			if err != nil {
				return nil, wrapErr(l.Index.Position, fmt.Errorf("failed to evaluate index: %w", err))
			}
			indexable, ok := base.(Indexable)
			if !ok {
				t, _ := base.Type(ctx)
				return nil, wrapErr(l.Index.Position, fmt.Errorf("target of type %s is not indexable", t))
			}
			base, err = indexable.Index(ctx, v)
			if err != nil {
				return nil, wrapErr(l.Index.Position, err)
			}
		} else if l.Start != nil {
			start, err := EvaluateExpression(ctx, scope, l.Start)
			if err != nil {
				return nil, wrapErr(l.Start.Position, fmt.Errorf("failed to evaluate slice start expression: %w", err))
			}
			end, err := EvaluateExpression(ctx, scope, l.End)
			if err != nil {
				return nil, wrapErr(l.End.Position, fmt.Errorf("failed to evaluate slice end expression: %w", err))
			}
			base, err = Slice(ctx, base, start, end)
			if err != nil {
				return nil, wrapErr(l.Start.Position, fmt.Errorf("failed to create slice: %w", err))
			}
		} else if l.Call != nil {
			callable, ok := base.(Callable)
			if !ok {
				t, _ := base.Type(ctx)
				return nil, wrapErr(l.Call.Position, fmt.Errorf("target of type %s is not callable", t))
			}
			kvs, err := callToKeyValue(ctx, scope, l.Call)
			if err != nil {
				return nil, err
			}
			base, err = callable.Call(ctx, scope, l.Call.Position, kvs)
			if err != nil {
				return nil, wrapErr(l.Call.Position, err)
			}
		}
	}

	if sel.Not {
		return Not(ctx, sel.Position, base)
	}

	return base, nil
}
