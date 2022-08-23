package ast

import (
	"fmt"
)

type Object struct {
	Position Position
	Fields   []*Field
}

type Key struct {
	Position Position
	Name     *String
	Match    bool
}

type Field struct {
	Position Position
	Key      Key

	Let      bool
	Embedded bool
	Value    *Value
	If       *If
	For      *For

	StaticKey   string
	StaticValue interface{}
}

type If struct {
	Condition *Expression
	Object    *Object
	Else      *If
}

type For struct {
	Position Position
	IndexVar string
	ValueVar string
	Array    *Expression
	Object   *Object
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

type String struct {
	Position  Position
	Parts     []StringPart
	Multiline bool
}

type StringPart struct {
	String     *string
	Expression *Expression
}

type Number string

type CommentGroups struct {
	Position Position
	End      int
	Lines    []string
}

type Value struct {
	Comments          map[int]CommentGroups
	Position          Position
	Array             *Array
	Object            *Object
	String            *String
	Number            *Number
	Bool              *bool
	Null              bool
	Expression        *Expression
	ListComprehension *For
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
	Index    *Expression
	Start    *Expression
	End      *Expression
	Call     *Call
}

type Call struct {
	Position Position
	Args     *Value
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
