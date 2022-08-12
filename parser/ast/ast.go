package ast

import (
	"encoding/json"
	"fmt"
)

type Object struct {
	Position Position
	Fields   []*Field
}

type Field struct {
	Position Position
	Key      string
	Value    *Value
}

type IfField struct {
	Position  Position
	Condition *Expression
	Object    *Object
}

type Position struct {
	Line   int
	Col    int
	Offset int
}

func (p Position) String() string {
	return fmt.Sprintf("%d:%d:%d", p.Line, p.Col, p.Offset)
}

func (p Position) IsSet() bool {
	return p.Line != 0 || p.Col != 0 || p.Offset != 0
}

type Value struct {
	Position   Position
	Array      *Array
	Object     *Object
	String     *string
	Number     *json.Number
	Bool       *bool
	Null       bool
	Expression *Expression
}

type Op struct {
	Position Position
	Op       string
}

type Operator struct {
	Position Position
	Op       *Op
	Selector *Selector
}

type Literal struct {
	Position Position
	Value    string
}

type Expression struct {
	Position Position
	Selector *Selector
	Operator []*Operator
}

type Lookup struct {
	Position Position
	Literal  *Literal
}

type Selector struct {
	Not      bool
	Position Position
	Value    *Value
	Literal  *Literal
	Parens   *Parens
	Lookup   []*Lookup
}

type Array struct {
	Position Position
	Values   []*Value
}

type Parens struct {
	Position
	Value *Value
}
