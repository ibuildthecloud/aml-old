package parser

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/acorn-io/aml/parser/ast"
)

func toInput(input interface{}, c *current) (interface{}, error) {
	if v, ok := input.(ast.Value); ok {
		return v, nil
	}
	ret := &ast.Value{
		Position: toPos(c),
		Object: &ast.Object{
			Position: toPos(c),
			Fields:   nil,
		},
	}

	for _, field := range toSlice(input) {
		ret.Object.Fields = append(ret.Object.Fields, field.(*ast.Field))
	}

	return ret, nil
}

func newOp(op interface{}, c *current) (interface{}, error) {
	return &ast.Op{
		Position: toPos(c),
		Op:       op.(string),
	}, nil
}

func toExpression(sel, ops interface{}, c *current) (interface{}, error) {
	expr := &ast.Expression{
		Position: toPos(c),
		Selector: sel.(*ast.Selector),
		Operator: nil,
	}

	for _, op := range toSlice(ops) {
		opSlice := toSlice(op)
		expr.Operator = append(expr.Operator, &ast.Operator{
			Position: toPos(c),
			Op:       opSlice[0].(*ast.Op),
			Selector: opSlice[1].(*ast.Selector),
		})
	}

	return &ast.Value{
		Position:   toPos(c),
		Expression: expr,
	}, nil
}

func toDotLookup(literal interface{}, c *current) (interface{}, error) {
	return &ast.Lookup{
		Position: toPos(c),
		Literal: &ast.Literal{
			Position: toPos(c),
			Value:    literal.(string),
		},
	}, nil
}

func toParens(value interface{}, c *current) (interface{}, error) {
	return &ast.Parens{
		Position: toPos(c),
		Value:    value.(*ast.Value),
	}, nil
}

func toSelector(not, value, lookup interface{}, c *current) (interface{}, error) {
	sel := &ast.Selector{
		Position: toPos(c),
		Not:      not != nil,
		Value:    nil,
		Literal:  nil,
		Parens:   nil,
		Lookup:   nil,
	}

	switch v := value.(type) {
	case string:
		sel.Literal = &ast.Literal{
			Position: toPos(c),
			Value:    v,
		}
	case *ast.Parens:
		sel.Parens = v
	case *ast.Value:
		sel.Value = v
	}

	for _, lookup := range toSlice(lookup) {
		sel.Lookup = append(sel.Lookup, lookup.(*ast.Lookup))
	}

	return sel, nil
}

func toIf(condition, obj interface{}, c *current) (interface{}, error) {
	return &ast.IfField{
		Position:  toPos(c),
		Condition: condition.(*ast.Expression),
		Object:    obj.(*ast.Value).Object,
	}, nil
}

func toField(key, value interface{}, c *current) (interface{}, error) {
	return &ast.Field{
		Position: toPos(c),
		Key:      key.(string),
		Value:    value.(*ast.Value),
	}, nil
}

func toString(chars interface{}, c *current) (interface{}, error) {
	buf := &strings.Builder{}
	for _, v := range toSlice(chars) {
		s := toSlice(v)
		buf.WriteString(s[1].(string))
	}
	s := buf.String()
	return &ast.Value{
		Position: toPos(c),
		String:   &s,
	}, nil
}

func toChar(c *current) (interface{}, error) {
	return strconv.Unquote(fmt.Sprintf("'%s'", c.text))
}

func toObject(fields interface{}, c *current) (interface{}, error) {
	obj := &ast.Value{
		Position: toPos(c),
		Object: &ast.Object{
			Position: toPos(c),
			Fields:   nil,
		},
	}

	for _, field := range toSlice(fields) {
		obj.Object.Fields = append(obj.Object.Fields, field.(*ast.Field))
	}

	return obj, nil
}

func toNull(c *current) (interface{}, error) {
	return &ast.Value{
		Position: toPos(c),
		Null:     true,
	}, nil
}

func toArray(head, tail interface{}, c *current) (interface{}, error) {
	array := &ast.Array{
		Position: toPos(c),
	}

	ret := &ast.Value{
		Position: toPos(c),
		Array:    array,
	}

	if head == nil {
		return ret, nil
	}

	array.Values = append(array.Values, head.(*ast.Value))

	for _, item := range toSlice(tail) {
		itemSlice := toSlice(item)
		if len(itemSlice) == 2 {
			array.Values = append(array.Values, itemSlice[1].(*ast.Value))
		}
	}

	return ret, nil
}

func toNumber(c *current) (interface{}, error) {
	n := json.Number(strings.TrimSpace(string(c.text)))
	return &ast.Value{
		Position: toPos(c),
		Number:   &n,
	}, nil
}

func toBool(v interface{}, c *current) (interface{}, error) {
	b := v.(string) == "true"
	return &ast.Value{
		Position: toPos(c),
		Bool:     &b,
	}, nil
}

func currentString(c *current) (interface{}, error) {
	return strings.TrimSpace(string(c.text)), nil
}

func noop(v interface{}) (interface{}, error) {
	return v, nil
}

func toSlice(v interface{}) []interface{} {
	if v == nil {
		return nil
	}
	return v.([]interface{})
}

func toPos(c *current) ast.Position {
	return ast.Position{
		Line:   c.pos.line,
		Col:    c.pos.col,
		Offset: c.pos.offset,
	}
}
