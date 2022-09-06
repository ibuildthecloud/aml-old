package eval

import (
	"context"
	"fmt"

	"github.com/acorn-io/aml/parser/ast"
)

func callToKeyValue(ctx context.Context, scope *Scope, args *ast.Call) (result []KeyValue, _ error) {
	if args.Positional != nil {
		value, err := EvaluateArray(ctx, scope, args.Positional)
		if err != nil {
			return nil, err
		}
		for _, v := range value {
			result = append(result, KeyValue{
				Value: v,
			})
		}
	}
	if args.Named != nil {
		kvs, err := ToObject(scope, args.Named).KeyValues(ctx)
		if err != nil {
			return nil, err
		}
		for _, kv := range kvs {
			result = append(result, KeyValue{
				Key:   kv.Key,
				Value: kv.Value,
			})
		}
	}

	return
}

func callToFields(ctx context.Context, scope *Scope, argsDeclared Value, args []KeyValue) (result []*ast.Field, _ error) {
	t, err := argsDeclared.Type(ctx)
	if err != nil {
		return nil, err
	}

	if t != TypeObject {
		return nil, fmt.Errorf("invalid function, args key is of type %s not %s", t, TypeObject)
	}

	obj := argsDeclared.(ObjectValue)
	kvs, err := obj.KeyValues(ctx)
	if err != nil {
		return nil, err
	}

	for i, v := range args {
		if v.Key != "" {
			result = append(result, &ast.Field{
				StaticKey:   v.Key,
				StaticValue: v.Value,
			})
			continue
		}
		if len(kvs) <= i {
			return nil, fmt.Errorf("function accepts %d args and recieved %d", len(kvs), len(args))
		}
		result = append(result, &ast.Field{
			StaticKey:   kvs[i].Key,
			StaticValue: v.Value,
		})
	}

	return result, nil
}

func (o *ObjectReference) argsToFields(ctx context.Context, scope *Scope, args []KeyValue) (result []*ast.Field, _ error) {
	argsDeclared, ok, err := o.Lookup(ctx, ArgsName)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return callToFields(ctx, scope, argsDeclared, args)
}

func (o *ObjectReference) Call(ctx context.Context, scope *Scope, pos ast.Position, args []KeyValue) (_ Value, err error) {
	defer func() {
		err = wrapErr(o.Position, err)
	}()
	tick(ctx)

	fields, err := o.argsToFields(ctx, scope, args)
	if err != nil {
		return nil, err
	}

	call := &ObjectReference{
		Position: pos,
		Scope:    o.Scope,
		Fields: []*ast.Field{
			{
				StaticKey: "args",
				StaticValue: &ObjectReference{
					Position: pos,
					Scope:    o.Scope,
					Fields:   fields,
				},
			},
		},
	}

	obj, err := MergeObjects(ctx, o, call)
	if err != nil {
		return nil, err
	}
	ret, ok, err := obj.Lookup(ctx, ReturnName)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("invalid function missing return key")
	}
	return ret, err
}
