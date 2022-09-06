package eval

import (
	"fmt"
	"strings"

	"github.com/acorn-io/aml/parser/ast"
)

type ErrPositionWrapped struct {
	Pos ast.Position
	Err error
}

func (e *ErrPositionWrapped) Error() string {
	next := fmt.Sprintf("%v", e.Err)
	prefixArrow := fmt.Sprintf("[%s]:%d:%d->", e.Pos.Source, e.Pos.Line, e.Pos.Col)
	prefixEnd := fmt.Sprintf("[%s]:%d:%d: ", e.Pos.Source, e.Pos.Line, e.Pos.Col)
	if strings.HasPrefix(next, prefixArrow) || strings.HasPrefix(next, prefixEnd) {
		return next
	}
	if len(next) > 0 && !strings.ContainsAny(next[0:1], "123456789") {
		return prefixEnd + next
	}
	return prefixArrow + next
}

func (e *ErrPositionWrapped) Unwrap() error {
	return e.Err
}

type ErrInvalidCondition struct {
	Err error
}

func (e *ErrInvalidCondition) Error() string {
	return fmt.Sprintf("invalid condition: %v", e.Err)
}

func (e *ErrInvalidCondition) Unwrap() error {
	return e.Err
}

func wrapErr(pos ast.Position, err error) error {
	if err == nil || pos.Offset == 0 {
		return err
	}
	return &ErrPositionWrapped{
		Pos: pos,
		Err: err,
	}
}
