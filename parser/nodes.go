package parser

import (
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
		Token:    op.(ast.Token),
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

func toCall(positional, named interface{}, c *current) (interface{}, error) {
	call := &ast.Call{
		Position: toPos(c),
	}
	if positional != nil {
		call.Positional = positional.(*ast.Value).Array
	}
	if named != nil {
		call.Named = named.(*ast.Value).Object
	}

	return &ast.Lookup{
		Position: toPos(c),
		Call:     call,
	}, nil
}

func toDotLookup(literal interface{}, c *current) (interface{}, error) {
	return &ast.Lookup{
		Position: toPos(c),
		Literal:  literal.(*ast.Literal),
	}, nil
}

func toIndexLookup(expr interface{}, c *current) (interface{}, error) {
	return &ast.Lookup{
		Position: toPos(c),
		Index:    expr.(*ast.Value).Expression,
	}, nil
}

func toSliceLookup(start, end interface{}, c *current) (interface{}, error) {
	return &ast.Lookup{
		Position: toPos(c),
		Start:    start.(*ast.Value).Expression,
		End:      end.(*ast.Value).Expression,
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
	case *ast.Literal:
		sel.Literal = v
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

func toForField(id1, id2, expr, obj, comma interface{}, c *current) (interface{}, error) {
	v, err := toListComprehension(ast.Token{}, id1, id2, expr, obj, nil, ast.Token{}, c)
	if err != nil {
		return nil, err
	}
	return &ast.Field{
		Position: toPos(c),
		For:      v.(*ast.Value).ListComprehension,
		Comma:    toComma(comma),
	}, nil
}

func toListComprehension(lbracket, id1, id2, expr, obj, ifCond, rbracket any, c *current) (interface{}, error) {
	var (
		valueVar = id1.(*ast.Literal)
		indexVar *ast.Literal
		cond     *ast.Expression
	)
	if id2 != nil {
		indexVar = valueVar
		valueVar = toSlice(id2)[1].(*ast.Literal)
	}
	if ifCond != nil {
		cond = toSlice(ifCond)[1].(*ast.Value).Expression
	}
	return &ast.Value{
		Position: toPos(c),
		ListComprehension: &ast.For{
			Condition: cond,
			Position:  toPos(c),
			IndexVar:  indexVar,
			ValueVar:  valueVar,
			Array:     expr.(*ast.Value).Expression,
			Object:    obj.(*ast.Value).Object,
			LBracket:  lbracket.(ast.Token),
			RBracket:  rbracket.(ast.Token),
		},
	}, nil
}

func toElse(objOrIf interface{}, c *current) (interface{}, error) {
	switch v := objOrIf.(type) {
	case *ast.Value:
		return &ast.If{
			Object: v.Object,
		}, nil
	case *ast.Field:
		return v.If, nil
	}
	return nil, fmt.Errorf("invalid else block %T", objOrIf)
}

func toIfField(condition, obj, ifElse interface{}, c *current) (interface{}, error) {
	f := &ast.Field{
		Position: toPos(c),
		If: &ast.If{
			Condition: condition.(*ast.Value).Expression,
			Object:    obj.(*ast.Value).Object,
		},
	}
	if ifElse != nil {
		f.If.Else = ifElse.(*ast.If)
	}
	return f, nil
}

func toComma(comma interface{}) *ast.Token {
	v, ok := comma.(ast.Token)
	if ok {
		return &v
	}
	return nil
}

func toLetField(let, key, sep, value, comma interface{}, c *current) (interface{}, error) {
	k, err := toKey(key, c)
	if err != nil {
		return nil, err
	}
	field := &ast.Field{
		Position:  toPos(c),
		Key:       *k,
		Value:     value.(*ast.Value),
		Separator: sep.(ast.Token),
		Comma:     toComma(comma),
	}

	letValue := let.(ast.Token)
	field.Let = &letValue

	return field, nil
}

func toFieldField(key, value, comma interface{}, c *current) (interface{}, error) {
	field := value.(*ast.Field)
	return &ast.Field{
		Position: toPos(c),
		Key:      *key.(*ast.Key),
		Value: &ast.Value{
			Position: field.Position,
			Object: &ast.Object{
				Position: field.Position,
				Fields:   []*ast.Field{field},
			},
		},
		Comma: toComma(comma),
	}, nil
}

func toKeyMatch(v, rbrace interface{}, c *current) (*ast.Key, error) {
	key, err := toKey(v, c)
	if err != nil {
		return nil, err
	}
	tok := rbrace.(ast.Token)
	key.Match = &tok
	return key, nil
}

func toKey(key interface{}, c *current) (*ast.Key, error) {
	switch v := key.(type) {
	case *ast.Literal:
		return &ast.Key{
			Position:   v.Position,
			Identifier: v,
		}, nil
	case *ast.Value:
		return &ast.Key{
			Position: v.Position,
			String:   v.String,
		}, nil
	case *ast.Key:
		return v, nil
	default:
		panic("unknown key")
	}
}

func toField(key, value, comma interface{}, c *current) (interface{}, error) {
	return &ast.Field{
		Position: toPos(c),
		Key:      *key.(*ast.Key),
		Value:    value.(*ast.Value),
		Comma:    toComma(comma),
	}, nil
}

func toString(chars, token interface{}, c *current) (interface{}, error) {
	ret := &ast.String{
		Position:   toPos(c),
		Whitespace: token.(ast.Token).Whitespace,
	}
	buf := &strings.Builder{}
	for _, v := range toSlice(chars) {
		s := toSlice(v)
		i := 1
		if len(s) == 2 {
			ret.Multiline = true
		} else {
			i = 2
		}
		switch v := s[i].(type) {
		case string:
			buf.WriteString(v)
		case *ast.Value:
			if buf.Len() > 0 {
				str := buf.String()
				ret.Parts = append(ret.Parts, ast.StringPart{
					String: &str,
				})
				buf = &strings.Builder{}
			}
			ret.Parts = append(ret.Parts, ast.StringPart{
				Expression: v.Expression,
			})
		}
	}
	if buf.Len() > 0 {
		str := buf.String()
		ret.Parts = append(ret.Parts, ast.StringPart{
			String: &str,
		})
	}
	return &ast.Value{
		Position: toPos(c),
		String:   ret,
	}, nil
}

func toChar(c *current) (interface{}, error) {
	text := string(c.text)
	if text == "\n" || text == "\r" {
		return text, nil
	}
	return strconv.Unquote(fmt.Sprintf("'%s'", c.text))
}

func toNamedArgs(fields interface{}, c *current) (interface{}, error) {
	return toObject(ast.Token{}, fields, ast.Token{}, c)
}

func toObject(lbrace, fields, rbrace interface{}, c *current) (interface{}, error) {
	obj := &ast.Value{
		Position: lbrace.(ast.Token).Position,
		Object: &ast.Object{
			Position: lbrace.(ast.Token).Position,
			LBrace:   lbrace.(ast.Token),
			RBrace:   rbrace.(ast.Token),
		},
	}

	for i, field := range toSlice(fields) {
		if i == 0 && obj.Position.Offset == 0 {
			obj.Position = field.(*ast.Field).Position
			obj.Object.Position = field.(*ast.Field).Position
		}
		obj.Object.Fields = append(obj.Object.Fields, field.(*ast.Field))
	}

	return obj, nil
}

func toNull(v1 interface{}, c *current) (interface{}, error) {
	null := v1.(ast.Token)
	return &ast.Value{
		Position: null.Position,
		Null: &ast.Null{
			Position:   null.Position,
			Whitespace: null.Whitespace,
		},
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

	if slice, ok := head.([]interface{}); ok {
		array.Values = append(array.Values, slice[1].(*ast.Value))
	} else {
		array.Values = append(array.Values, head.(*ast.Value))
	}

	for _, item := range toSlice(tail) {
		itemSlice := toSlice(item)
		if len(itemSlice) == 2 {
			array.Values = append(array.Values, itemSlice[1].(*ast.Value))
		} else if len(itemSlice) == 3 {
			array.Values = append(array.Values, itemSlice[2].(*ast.Value))
		}
	}

	return ret, nil
}

func toEmbeddedField(v, comma interface{}, c *current) (interface{}, error) {
	field := &ast.Field{
		Position: toPos(c),
		Key:      ast.Key{},
		Embedded: true,
		Value:    v.(*ast.Value),
		Comma:    toComma(comma),
	}
	return field, nil
}

func toNumber(c *current) (interface{}, error) {
	n := ast.Number(strings.TrimSpace(strings.Split(string(c.text), "//")[0]))
	return &ast.Value{
		Position: toPos(c),
		Number:   &n,
	}, nil
}

func toBool(v interface{}, _ *current) (interface{}, error) {
	tok := v.(ast.Token)
	return &ast.Value{
		Position: tok.Position,
		Bool: &ast.Bool{
			Position:   tok.Position,
			Value:      tok.Value == "true",
			Whitespace: tok.Whitespace,
		},
	}, nil
}

func space(c *current) (interface{}, error) {
	return &ast.Space{
		Position: toPos(c),
		String:   string(c.text),
	}, nil
}

func comment(c *current) (interface{}, error) {
	return ast.Comment{
		Position: toPos(c),
		String:   string(c.text),
	}, nil
}

func toWhitespace(elems interface{}, c *current) (interface{}, error) {
	result := ast.Whitespace{
		Position: toPos(c),
	}
	for _, elem := range toSlice(elems) {
		switch v := elem.(type) {
		case ast.Space:
			space := v
			result.Elements = append(result.Elements, ast.WhitespaceElement{
				Space: &space,
			})
		case ast.Comment:
			comment := v
			result.Elements = append(result.Elements, ast.WhitespaceElement{
				Comment: &comment,
			})
		}
	}
	return result, nil
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
	s, _ := c.globalStore["source"].(string)
	return ast.Position{
		Source: s,
		Line:   c.pos.line,
		Col:    c.pos.col,
		Offset: c.pos.offset,
	}
}

func charsToString(chars interface{}) string {
	result := &strings.Builder{}
	switch v := chars.(type) {
	case []interface{}:
		for _, chars := range toSlice(v) {
			for _, c := range chars.([]uint8) {
				result.WriteByte(c)
			}
		}
	case []uint8:
		for _, c := range v {
			result.WriteByte(c)
		}
	}
	return result.String()
}

func identifier(start, end, whitespace interface{}, c *current) (*ast.Literal, error) {
	return &ast.Literal{
		Position:   toPos(c),
		Value:      charsToString(start) + charsToString(end),
		Whitespace: whitespace.(ast.Whitespace),
	}, nil
}

func token(s, whitespace interface{}, c *current) (ast.Token, error) {
	switch v := whitespace.(type) {
	case ast.Whitespace:
		return ast.Token{
			Position:   toPos(c),
			Value:      string(s.([]byte)),
			Whitespace: v,
		}, nil
	}
	wh, err := toWhitespace(whitespace, c)
	return ast.Token{
		Position:   toPos(c),
		Value:      string(s.([]byte)),
		Whitespace: wh.(ast.Whitespace),
	}, err
}
