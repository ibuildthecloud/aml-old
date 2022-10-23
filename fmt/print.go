package fmt

import (
	"io"
	"regexp"
	"strings"

	"github.com/acorn-io/aml/parser/ast"
)

var (
	identifiedRegexp = regexp.MustCompile("^[_a-zA-Z][_a-zA-Z0-9]*$")
	mathOp           = map[string]bool{
		"+": true,
		"-": true,
		"/": true,
		"*": true,
	}
)

type formatContext struct {
	indent   string
	top      bool
	comments map[int]ast.CommentGroups
	out      *writer
}

type writer struct {
	err     error
	content bool
	out     io.Writer
}

func (w *writer) write(s string) {
	if w.err != nil {
		return
	}
	c, err := w.out.Write([]byte(s))
	if err != nil {
		w.err = err
	}
	if c > 0 {
		w.content = true
	}
}

func Print(out io.Writer, value *ast.Value) error {
	writer := &writer{
		out: out,
	}
	printValue(formatContext{
		comments: value.Comments,
		out:      writer,
		top:      true,
	}, *value)
	return writer.err
}

func isNumber(selector *ast.Selector) bool {
	return selector.Value != nil && selector.Value.Number != nil
}

func printExpression(c formatContext, expr *ast.Expression) {
	if expr.Selector != nil {
		printSelector(c, expr.Selector)
	}
	for _, op := range expr.Operator {
		if mathOp[op.Op.Op] && isNumber(op.Selector) {
			c.out.write(op.Op.Op)
		} else {
			c.out.write(" ")
			c.out.write(op.Op.Op)
			c.out.write(" ")
		}
		printSelector(c, op.Selector)
	}
}

func printCall(c formatContext, call ast.Call) {
	c.out.write("(")
	i := 0
	if call.Positional != nil {
		for _, arg := range call.Positional.Values {
			if i > 0 {
				c.out.write(", ")
			}
			printValue(c, *arg)
			i++
		}
	}
	if call.Named != nil {
		for _, field := range call.Named.Fields {
			if i > 0 {
				c.out.write(", ")
			}
			printField(c, *field)
			i++
		}
	}
	c.out.write(")")
}

func printKey(c formatContext, key ast.Key) {
	if key.Name == nil {
		return
	}
	if key.Match {
		c.out.write("[~=")
	}
	if !key.Match && len(key.Name.Parts) == 1 && key.Name.Parts[0].String != nil && identifiedRegexp.MatchString(*key.Name.Parts[0].String) {
		c.out.write(*key.Name.Parts[0].String)
	} else {
		printString(c, *key.Name)
	}
	if key.Match {
		c.out.write("]")
	}
	c.out.write(": ")
}

func printFor(c formatContext, forField ast.For) {
	c.out.write("for ")
	if forField.IndexVar != "" {
		c.out.write(forField.IndexVar)
		c.out.write(", ")
	}
	c.out.write(forField.ValueVar)
	c.out.write(" in ")
	printExpression(c, forField.Array)
	c.out.write(" ")
	printObject(c, *forField.Object)
	if forField.Condition != nil {
		c.out.write(" if ")
		printExpression(c, forField.Condition)
	}
}

func printIf(c formatContext, ifField ast.If) {
	if ifField.Condition != nil {
		c.out.write("if ")
		printExpression(c, ifField.Condition)
		c.out.write(" ")
	}

	printObject(c, *ifField.Object)

	if ifField.Else != nil {
		c.out.write(" else ")
		printIf(c, *ifField.Else)
	}
}

func printComments(c formatContext, pos ast.Position) {
	comments := c.comments[pos.Offset]
	if len(comments.Lines) == 0 && strings.Contains(comments.Text, "\n") {
		c.out.write("\n")
	}
	for _, line := range comments.Lines {
		if c.out.content {
			c.out.write("\n")
		}
		for _, line := range strings.Split(line, "\n") {
			c.out.write("//")
			c.out.write(line)
			c.out.write("\n")
		}
	}
}

func printField(c formatContext, field ast.Field) {
	printComments(c, field.Position)
	switch {
	case field.If != nil:
		printIf(c, *field.If)
	case field.For != nil:
		printFor(c, *field.For)
	case field.Value != nil:
		if !field.Embedded {
			printKey(c, field.Key)
		}
		printValue(c, *field.Value)
	}
}

func printLookup(c formatContext, lookup ast.Lookup) {
	switch {
	case lookup.Index != nil:
		c.out.write("[")
		printExpression(c, lookup.Index)
		c.out.write("]")
	case lookup.Literal != nil:
		c.out.write(".")
		c.out.write(lookup.Literal.Value)
	case lookup.Call != nil:
		printCall(c, *lookup.Call)
	case lookup.End != nil && lookup.Start != nil:
		c.out.write("[")
		printExpression(c, lookup.Start)
		c.out.write(":")
		printExpression(c, lookup.End)
		c.out.write("]")
	}
}

func printSelector(c formatContext, sel *ast.Selector) {
	if sel.Not {
		c.out.write("!")
	}
	switch {
	case sel.Literal != nil:
		c.out.write(sel.Literal.Value)
	case sel.Value != nil:
		printValue(c, *sel.Value)
	case sel.Parens != nil:
		c.out.write("(")
		printValue(c, *sel.Parens.Value)
		c.out.write(")")
	}

	for _, lookup := range sel.Lookup {
		printLookup(c, *lookup)
	}
}

func printString(c formatContext, value ast.String) {
	if value.Multiline {
		c.out.write(`"""`)
	} else {
		c.out.write(`"`)
	}

	for _, part := range value.Parts {
		if part.String != nil {
			c.out.write(*part.String)
		}
		if part.Expression != nil {
			c.out.write("\\(")
			printExpression(c, part.Expression)
			c.out.write(")")
		}
	}

	if value.Multiline {
		c.out.write(`"""`)
	} else {
		c.out.write(`"`)
	}
}

func printArray(c formatContext, array ast.Array) {
	newC := c
	newC.indent += "\t"

	c.out.write("[")
	if len(array.Values) == 1 {
		printValue(c, *array.Values[0])
	} else {
		for i, val := range array.Values {
			c.out.write("\n")
			c.out.write(newC.indent)
			printValue(newC, *val)
			if i < len(array.Values)-1 {
				c.out.write(",")
			}
		}
		if len(array.Values) > 0 {
			c.out.write("\n")
			c.out.write(c.indent)
		}
	}
	c.out.write("]")
}

func printListComprehension(c formatContext, list ast.For) {
	c.out.write("[ ")
	printFor(c, list)
	c.out.write("]")
}

func printValue(c formatContext, value ast.Value) {
	switch {
	case value.Null:
		c.out.write("null")
	case value.String != nil:
		printString(c, *value.String)
	case value.Array != nil:
		printArray(c, *value.Array)
	case value.Object != nil:
		printObject(c, *value.Object)
	case value.Expression != nil:
		printExpression(c, value.Expression)
	case value.Bool != nil:
		if *value.Bool {
			c.out.write("true")
		} else {
			c.out.write("false")
		}
	case value.Number != nil:
		c.out.write((string)(*value.Number))
	case value.ListComprehension != nil:
		printListComprehension(c, *value.ListComprehension)
	}
}

func printObject(c formatContext, value ast.Object) {
	if len(value.Fields) == 0 {
		c.out.write("{}")
		return
	}

	if !c.top && len(value.Fields) == 1 && value.Position.Line == value.Fields[0].Position.Line {
		c.out.write("{ ")
		printField(c, *value.Fields[0])
		c.out.write(" }")
		return
	}

	newC := c
	newC.top = false
	if !c.top {
		c.out.write("{\n")
		newC.indent += "\t"
	}
	for _, field := range value.Fields {
		c.out.write(newC.indent)
		printField(newC, *field)
		c.out.write("\n")
	}
	if !c.top {
		c.out.write(c.indent)
		c.out.write("}")
	}
}
