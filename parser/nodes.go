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
		Comments: map[int]ast.CommentGroups{},
	}

	for _, field := range toSlice(input) {
		ret.Object.Fields = append(ret.Object.Fields, field.(*ast.Field))
	}

	comments, _ := c.state["comments"].([]ast.CommentGroups)
	for _, comment := range comments {
		ret.Comments[comment.End] = comment
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

func toCall(val interface{}, c *current) (interface{}, error) {
	return &ast.Lookup{
		Call: &ast.Call{
			Position: toPos(c),
			Args:     val.(*ast.Value),
		},
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

func toIndexLookup(expr interface{}, c *current) (interface{}, error) {
	return &ast.Lookup{
		Position: toPos(c),
		Index:    expr.(*ast.Value).Expression,
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

func toForField(id1, id2, expr, obj interface{}, c *current) (interface{}, error) {
	v, err := toListComprehension(id1, id2, expr, obj, c)
	if err != nil {
		return nil, err
	}
	return &ast.Field{
		Position: toPos(c),
		For:      v.(*ast.Value).ListComprehension,
	}, nil
}

func toListComprehension(id1, id2, expr, obj interface{}, c *current) (interface{}, error) {
	var (
		valueVar = id1.(string)
		indexVar = ""
	)
	if id2 != nil {
		indexVar = valueVar
		valueVar = toSlice(id2)[1].(string)
	}
	return &ast.Value{
		Position: toPos(c),
		ListComprehension: &ast.For{
			Position: toPos(c),
			IndexVar: indexVar,
			ValueVar: valueVar,
			Array:    expr.(*ast.Value).Expression,
			Object:   obj.(*ast.Value).Object,
		},
	}, nil
}

func toIfField(condition, obj interface{}, c *current) (interface{}, error) {
	return &ast.Field{
		Position: toPos(c),
		If: &ast.If{
			Condition: condition.(*ast.Value).Expression,
			Object:    obj.(*ast.Value).Object,
		},
	}, nil
}

func toLetField(key, value interface{}, c *current) (interface{}, error) {
	k, err := toKey(key, c)
	if err != nil {
		return nil, err
	}
	return &ast.Field{
		Position: toPos(c),
		Let:      true,
		Key:      *k,
		Value:    value.(*ast.Value),
	}, nil
}

func toFieldField(key, value interface{}, c *current) (interface{}, error) {
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
	}, nil
}

func toKeyMatch(v interface{}, c *current) (*ast.Key, error) {
	key, err := toKey(v, c)
	if err != nil {
		return nil, err
	}
	key.Match = true
	return key, nil
}

func toKey(key interface{}, c *current) (*ast.Key, error) {
	var keyName *ast.String
	switch v := key.(type) {
	case string:
		keyName = toASTString(toPos(c), v, c)
	case *ast.Value:
		keyName = v.String
	case *ast.Key:
		return v, nil
	}
	return &ast.Key{
		Position: toPos(c),
		Name:     keyName,
	}, nil
}

func toField(key, value interface{}, c *current) (interface{}, error) {
	return &ast.Field{
		Position: toPos(c),
		Key:      *key.(*ast.Key),
		Value:    value.(*ast.Value),
	}, nil
}

func toString(chars interface{}, c *current) (interface{}, error) {
	ret := &ast.String{
		Position: toPos(c),
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

func comments(c *current) []string {
	v, _ := c.state["comments"].([]string)
	return v
}

func toEmbeddedField(v interface{}, c *current) (interface{}, error) {
	return &ast.Field{
		Position: toPos(c),
		Key:      ast.Key{},
		Embedded: true,
		Value:    v.(*ast.Value),
	}, nil
}

func toNumber(c *current) (interface{}, error) {
	n := ast.Number(strings.TrimSpace(strings.Split(string(c.text), "//")[0]))
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

type spacePosition struct {
	Position ast.Position
	Text     string
}

func space(c *current) (interface{}, error) {
	return &spacePosition{
		Position: toPos(c),
		Text:     string(c.text),
	}, nil
}

func whitespace(w interface{}, c *current) error {
	var (
		comments     []string
		currentBlock []string
		position     ast.Position
		end          int
	)
	for i, c := range toSlice(w) {
		sp, ok := c.(*spacePosition)
		if !ok {
			continue
		}
		if i == 0 {
			position = sp.Position
		}
		end = sp.Position.Offset + len(sp.Text)
		trimmedText := strings.TrimSpace(sp.Text)
		if strings.HasPrefix(trimmedText, "//") {
			currentBlock = append(currentBlock, strings.TrimPrefix(trimmedText, "//"))
		} else if sp.Text == "\n" && len(currentBlock) > 0 {
			comments = append(comments, strings.Join(currentBlock, "\n"))
			currentBlock = nil
		}
	}

	if len(currentBlock) > 0 {
		comments = append(comments, strings.Join(currentBlock, "\n"))
	}
	if len(comments) == 0 {
		return nil
	}
	v, _ := c.state["comments"].([]ast.CommentGroups)
	newV := make([]ast.CommentGroups, len(v), len(v)+1)
	copy(newV, v)
	c.state["comments"] = append(newV, ast.CommentGroups{
		Position: position,
		End:      end,
		Lines:    comments,
	})
	return nil
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

func toASTString(pos ast.Position, s string, c *current) *ast.String {
	return &ast.String{
		Position: pos,
		Parts: []ast.StringPart{
			{
				String: &s,
			},
		},
	}
}
