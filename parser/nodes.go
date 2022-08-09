package parser

import (
	"encoding/json"

	"github.com/acorn-io/aml/parser/ast"
)

func newSelector(n, v, d interface{}, c *current) (interface{}, error) {
	sel := ast.Selector{
		Position: toPos(c),
		Not:      n != nil,
	}
	if v, ok := v.(ast.Value); ok {
		sel.Value = &v
	}
	if l, ok := v.(ast.Literal); ok {
		sel.Literal = &l
	}
	if l, ok := v.([]interface{}); ok && len(l) > 0 {
		sel.Parens = &ast.Parens{
			Position: toPos(c),
			Value:    l[2].(ast.Value),
		}
	}
	for _, item := range toSlice(d) {
		s := item.(ast.Lookup)
		sel.Lookup = append(sel.Lookup, s)
	}
	return sel, nil
}

func newCallLookup(head, tail interface{}, c *current) (interface{}, error) {
	lookup := ast.Lookup{
		Position: toPos(c),
		Call: &ast.Call{
			Position: toPos(c),
		},
	}

	if head != nil {
		lookup.Call.Args = append(lookup.Call.Args, head.(ast.Value))
	}
	for _, item := range toSlice(tail) {
		lookup.Call.Args = append(lookup.Call.Args, item.(ast.Value))
	}

	return lookup, nil
}

func newDotLookup(v interface{}, c *current) (interface{}, error) {
	a := ast.Lookup{
		Position: toPos(c),
	}
	if l, ok := v.(ast.Literal); ok {
		a.Literal = &l
	}
	return a, nil
}

func newIndexLookup(v interface{}, c *current) (interface{}, error) {
	a := ast.Lookup{
		Position: toPos(c),
	}
	if v, ok := v.(ast.Value); ok {
		a.Number = v.Number
		if v.String != nil {
			a.Literal = &ast.Literal{
				Position: v.Position,
				Value:    *v.String,
			}
		}
	}

	return a, nil
}

func newObject(f interface{}, c *current) (interface{}, error) {
	b := ast.Object{
		Position: toPos(c),
	}
	for _, f := range toSlice(f) {
		b.Fields = append(b.Fields, f.(ast.Field))
	}
	return ast.Value{
		Position: toPos(c),
		Object:   &b,
	}, nil
}

func newExpression(op, rest interface{}, c *current) (interface{}, error) {
	sel := op.(ast.Selector)
	restSlice := toSlice(rest)

	if sel.Value != nil && len(sel.Lookup) == 0 && len(restSlice) == 0 {
		return *sel.Value, nil
	}
	expr := ast.Expression{
		Position: toPos(c),
		Selector: sel,
	}
	for _, item := range restSlice {
		rest := toSlice(item)
		expr.Operator = append(expr.Operator, ast.Operator{
			Op:       rest[1].(string),
			Selector: rest[3].(ast.Selector),
		})
	}
	return ast.Value{
		Position:   toPos(c),
		Expression: &expr,
	}, nil
}

func newForField(v1, v2, e, v interface{}, c *current) (interface{}, error) {
	a := ast.Field{
		Position: toPos(c),
		ForField: &ast.ForField{
			Position: toPos(c),
			Var1:     v1.(ast.Literal).Value,
			List:     e.(ast.Value),
			Object:   *v.(ast.Value).Object,
		},
	}
	if v2 != nil {
		s := toSlice(v2)[2].(ast.Literal).Value
		a.ForField.Var2 = &s
	}
	return a, nil
}

func newField(key, value interface{}, c *current) (interface{}, error) {
	val := value.(ast.Value)
	f := ast.Field{
		Position: toPos(c),
		Value:    &val,
	}
	if k, ok := key.(ast.Value); ok {
		f.Key = *k.String
	}
	if k, ok := key.(ast.Literal); ok {
		f.Key = k.Value
	}
	return f, nil
}

func toPos(c *current) ast.Position {
	return ast.Position{
		Line:   c.pos.line,
		Col:    c.pos.col,
		Offset: c.pos.offset,
	}
}

func newBool(c *current) (interface{}, error) {
	b := c.text[0] == 't'
	return ast.Value{
		Position: toPos(c),
		Bool:     &b,
	}, nil
}

func newNull(c *current) (interface{}, error) {
	return ast.Value{
		Position: toPos(c),
		Null:     true,
	}, nil
}

func newNumber(c *current) (interface{}, error) {
	num := json.Number(c.text)
	return ast.Value{
		Position: toPos(c),
		Number:   &num,
	}, nil
}

func newIdentifier(c *current) (interface{}, error) {
	s, err := unquote(c.text)
	return ast.Literal{
		Position: toPos(c),
		Value:    s,
	}, err
}

func toString(c *current) (interface{}, error) {
	return string(c.text), nil
}

func newQuotedString(c *current) (interface{}, error) {
	s, err := unquote(c.text[1 : len(c.text)-1])
	return ast.Value{
		Position: toPos(c),
		String:   &s,
	}, err
}

func noop(v interface{}) (interface{}, error) {
	return v, nil
}

func newArrayComprehension(v1, v2, e, v interface{}, c *current) (interface{}, error) {
	a := ast.Array{
		Position: toPos(c),
		Comprehension: &ast.ArrayComprehension{
			Position: toPos(c),
			Var1:     v1.(ast.Literal).Value,
			List:     e.(ast.Value),
			Object:   v.(ast.Value),
		},
	}
	if v2 != nil {
		s := toSlice(v2)[2].(ast.Literal).Value
		a.Comprehension.Var2 = &s
	}
	return ast.Value{
		Position: toPos(c),
		Array:    &a,
	}, nil
}

func newElementList(head, tail interface{}, c *current) (interface{}, error) {
	result := ast.Value{
		Position: toPos(c),
		Array: &ast.Array{
			Position: toPos(c),
		},
	}
	for _, value := range toSlice(head) {
		result.Array.Values = append(result.Array.Values, value.(ast.Value))
	}
	if tail != nil {
		result.Array.Values = append(result.Array.Values, tail.(ast.Value))
	}
	return result, nil
}

func unquote(b []byte) (string, error) {
	var str string
	d := append([]byte{'"'}, append(b, '"')...)
	return str, json.Unmarshal(d, &str)

}

func toSlice(v interface{}) []interface{} {
	if v == nil {
		return nil
	}
	return v.([]interface{})
}
