package eval

import (
	"fmt"

	"github.com/acorn-io/aml/parser/ast"
)

type ErrPositionWrapped struct {
	Pos ast.Position
	Err error
}

func (e *ErrPositionWrapped) Error() string {
	return fmt.Sprintf("%s: %v", e.Pos, e.Err)
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
	if err == nil {
		return nil
	}
	return &ErrPositionWrapped{
		Pos: pos,
		Err: err,
	}
}
