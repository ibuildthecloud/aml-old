package parser

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acorn-io/aml/parser/ast"
	"github.com/stretchr/testify/assert"
)

var cases = map[string]interface{}{
	"values.aml": ast.Value{
		Object: &ast.Object{
			Fields: []ast.Field{
				{
					Key: "int",
					Value: &ast.Value{
						Number: ptr(json.Number("4")),
					},
				},
				{
					Key: "fraction",
					Value: &ast.Value{
						Number: ptr(json.Number("4.0")),
					},
				},
				{
					Key: "exponent",
					Value: &ast.Value{
						Number: ptr(json.Number("4e10")),
					},
				},
				{
					Key: "string",
					Value: &ast.Value{
						String: ptr("h\t\ni"),
					},
				},
				{
					Key: "bool",
					Value: &ast.Value{
						Bool: ptr(true),
					},
				},
				{
					Key: "fbool",
					Value: &ast.Value{
						Bool: ptr(false),
					},
				},
				{
					Key: "null",
					Value: &ast.Value{
						Null: true,
					},
				},
				{
					Key: "string key",
					Value: &ast.Value{
						Bool: ptr(true),
					},
				},
				{
					Key: "string",
					Value: &ast.Value{
						Bool: ptr(true),
					},
				},
				{
					Key: "jsonarray_single",
					Value: &ast.Value{
						Array: &ast.Array{
							Values: []ast.Value{{String: ptr("string")}},
						},
					},
				},
				{
					Key: "jsonarray_two",
					Value: &ast.Value{
						Array: &ast.Array{
							Values: []ast.Value{{String: ptr("string")},
								{Number: ptr(json.Number("4"))}},
						},
					},
				},
				{
					Key: "jsonarray_empty",
					Value: &ast.Value{
						Array: &ast.Array{
							Values: nil,
						},
					},
				},
				{
					Key: "jsonarray_normal",
					Value: &ast.Value{
						Array: &ast.Array{
							Values: []ast.Value{{String: ptr("string")},
								{Number: ptr(json.Number("4"))}},
						},
					},
				},
				{
					Key: "jsonarray_trailing_object",
					Value: &ast.Value{
						Array: &ast.Array{
							Values: []ast.Value{{String: ptr("string")},
								{
									Object: &ast.Object{
										Fields: []ast.Field{
											{
												Key: "a",
												Value: &ast.Value{
													String: ptr("b"),
												},
											},
										},
									},
								},
							},
						},
					},
				},
				{
					Key: "jsonarray_trailing",
					Value: &ast.Value{
						Array: &ast.Array{
							Values: []ast.Value{{String: ptr("string")},
								{Number: ptr(json.Number("4"))}},
						},
					},
				},
				{
					Key: "list_comprehension",
					Value: &ast.Value{
						Array: &ast.Array{
							Comprehension: &ast.ArrayComprehension{
								Var1: "x",
								List: ast.Value{
									Expression: &ast.Expression{
										Selector: ast.Selector{
											Literal: &ast.Literal{
												Value: "foo",
											},
										},
									},
								},
								Object: ast.Value{
									Number: ptr(json.Number("4")),
								},
							},
						},
					},
				},
			},
		},
	},
	"value_string_invalid.aml": errors.New("no match found"),
	"brace_object_string_value.aml,brace_object_inline_value.aml": ast.Value{
		Object: &ast.Object{
			Fields: []ast.Field{
				{
					Key: "key",
					Value: &ast.Value{
						String: ptr("string"),
					},
				},
				{
					Key: "key",
					Value: &ast.Value{
						String: ptr("stri\nng2"),
					},
				},
			},
		},
	},
	"brace_object_nested_value.aml": ast.Value{
		Object: &ast.Object{
			Fields: []ast.Field{
				{
					Key: "key",
					Value: &ast.Value{
						String: ptr("string"),
					},
				},
				{
					Key: "key",
					Value: &ast.Value{
						Object: &ast.Object{
							Fields: []ast.Field{
								{
									Key:   "key",
									Value: &ast.Value{String: ptr("string2")},
								},
								{
									Key:   "key",
									Value: &ast.Value{String: ptr("string2")},
								},
							},
						},
					},
				},
			},
		},
	},
	"expressions.aml": ast.Value{
		Object: &ast.Object{
			Fields: []ast.Field{
				{
					Key: "mul_digit",
					Value: &ast.Value{
						Expression: &ast.Expression{
							Selector: ast.Selector{
								Value: &ast.Value{
									Number: ptr(json.Number("1")),
								},
							},
							Operator: []ast.Operator{
								{
									Op: "*",
									Selector: ast.Selector{
										Value: &ast.Value{
											Number: ptr(json.Number("2")),
										},
									},
								},
								{
									Op: "*",
									Selector: ast.Selector{
										Value: &ast.Value{
											Number: ptr(json.Number("3")),
										},
									},
								},
							},
						},
					},
				},
				{
					Key: "add_digit",
					Value: &ast.Value{
						Expression: &ast.Expression{
							Selector: ast.Selector{
								Value: &ast.Value{
									Number: ptr(json.Number("1")),
								},
							},
							Operator: []ast.Operator{
								{
									Op: "+",
									Selector: ast.Selector{
										Value: &ast.Value{
											Number: ptr(json.Number("2")),
										},
									},
								},
							},
						},
					},
				},
				{
					Key: "div_digit",
					Value: &ast.Value{
						Expression: &ast.Expression{
							Selector: ast.Selector{
								Value: &ast.Value{
									Number: ptr(json.Number("1")),
								},
							},
							Operator: []ast.Operator{
								{
									Op: "/",
									Selector: ast.Selector{
										Value: &ast.Value{
											Number: ptr(json.Number("2")),
										},
									},
								},
							},
						},
					},
				},
				{
					Key: "sub_digit",
					Value: &ast.Value{
						Expression: &ast.Expression{
							Selector: ast.Selector{
								Value: &ast.Value{
									Number: ptr(json.Number("1")),
								},
							},
							Operator: []ast.Operator{
								{
									Op: "-",
									Selector: ast.Selector{
										Value: &ast.Value{
											Number: ptr(json.Number("2")),
										},
									},
								},
							},
						},
					},
				},
				{
					Key: "obj_merge",
					Value: &ast.Value{
						Expression: &ast.Expression{
							Selector: ast.Selector{
								Value: &ast.Value{
									Object: &ast.Object{},
								},
							},
							Operator: []ast.Operator{
								{
									Op: "&",
									Selector: ast.Selector{
										Value: &ast.Value{
											Object: &ast.Object{},
										},
									},
								},
							},
						},
					},
				},
				{
					Key: "name_sel",
					Value: &ast.Value{
						Expression: &ast.Expression{
							Selector: ast.Selector{
								Literal: &ast.Literal{
									Value: "foo",
								},
								Lookup: []ast.Lookup{
									{
										Literal: &ast.Literal{
											Value: "bar",
										},
									},
									{
										Literal: &ast.Literal{
											Value: "baz",
										},
									},
								},
							},
						},
					},
				},
				{
					Key: "name_sel2",
					Value: &ast.Value{
						Expression: &ast.Expression{
							Selector: ast.Selector{
								Literal: &ast.Literal{
									Value: "foo",
								},
								Lookup: []ast.Lookup{
									{
										Literal: &ast.Literal{
											Value: "bar",
										},
									},
									{
										Number: ptr(json.Number("3")),
									},
									{
										Literal: &ast.Literal{
											Value: "baz",
										},
									},
									{
										Literal: &ast.Literal{
											Value: "key",
										},
									},
								},
							},
						},
					},
				},
				{
					Key: "method",
					Value: &ast.Value{
						Expression: &ast.Expression{
							Selector: ast.Selector{
								Parens: &ast.Parens{
									Value: ast.Value{
										Expression: &ast.Expression{
											Selector: ast.Selector{
												Value: &ast.Value{
													Object: &ast.Object{
														Fields: []ast.Field{
															{
																Key: "foo",
																Value: &ast.Value{
																	Expression: &ast.Expression{
																		Selector: ast.Selector{
																			Literal: &ast.Literal{
																				Value: "bar",
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
											Operator: []ast.Operator{
												{
													Op: "&",
													Selector: ast.Selector{
														Value: &ast.Value{
															Object: &ast.Object{
																Fields: []ast.Field{
																	{
																		Key: "baz",
																		Value: &ast.Value{
																			Number: ptr(json.Number("4")),
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
								Lookup: []ast.Lookup{
									{
										Literal: &ast.Literal{
											Value: "out",
										},
									},
									{
										Call: &ast.Call{
											Args: []ast.Value{
												{
													Number: ptr(json.Number("4")),
												},
												{
													Number: ptr(json.Number("3")),
												},
											},
										},
									},
								},
							},
						},
					},
				},
				{
					Key: "not",
					Value: &ast.Value{
						Expression: &ast.Expression{
							Selector: ast.Selector{
								Literal: &ast.Literal{
									Value: "foo",
								},
							},
							Operator: []ast.Operator{
								{
									Op: "*",
									Selector: ast.Selector{
										Not: true,
										Literal: &ast.Literal{
											Value: "bar",
										},
									},
								},
							},
						},
					},
				},
				{
					ForField: &ast.ForField{
						Var1: "x",
						Var2: ptr("y"),
						List: ast.Value{
							Array: &ast.Array{
								Values: []ast.Value{
									{
										Number: ptr(json.Number("1")),
									},
									{
										Number: ptr(json.Number("2")),
									},
								},
							},
						},
						Object: ast.Object{
							Fields: []ast.Field{
								{
									Key: "a",
									Value: &ast.Value{
										Number: ptr((json.Number("1"))),
									},
								},
							},
						},
					},
				},
			},
		},
	},
}

type clearPosition struct {
	t *testing.T
}

func (c clearPosition) Visit(obj interface{}) {
	if o, ok := obj.(*ast.Object); ok {
		assert.True(c.t, o.Position.IsSet())
		o.Position = ast.Position{}
	}
	if o, ok := obj.(*ast.Array); ok {
		assert.True(c.t, o.Position.IsSet())
		o.Position = ast.Position{}
	}
	if o, ok := obj.(*ast.Field); ok {
		assert.True(c.t, o.Position.IsSet())
		o.Position = ast.Position{}
	}
	if o, ok := obj.(*ast.Value); ok {
		assert.True(c.t, o.Position.IsSet())
		o.Position = ast.Position{}
	}
	if o, ok := obj.(*ast.ArrayComprehension); ok {
		assert.True(c.t, o.Position.IsSet())
		o.Position = ast.Position{}
	}
	if o, ok := obj.(*ast.Expression); ok {
		assert.True(c.t, o.Position.IsSet())
		o.Position = ast.Position{}
	}
	if o, ok := obj.(*ast.Selector); ok {
		assert.True(c.t, o.Position.IsSet())
		o.Position = ast.Position{}
	}
	if o, ok := obj.(*ast.Literal); ok {
		assert.True(c.t, o.Position.IsSet())
		o.Position = ast.Position{}
	}
	if o, ok := obj.(*ast.Lookup); ok {
		assert.True(c.t, o.Position.IsSet())
		o.Position = ast.Position{}
	}
	if o, ok := obj.(*ast.Parens); ok {
		assert.True(c.t, o.Position.IsSet())
		o.Position = ast.Position{}
	}
	if o, ok := obj.(*ast.Call); ok {
		assert.True(c.t, o.Position.IsSet())
		o.Position = ast.Position{}
	}
	if o, ok := obj.(*ast.ForField); ok {
		assert.True(c.t, o.Position.IsSet())
		o.Position = ast.Position{}
	}
}

func TestCases(t *testing.T) {
	const rootDir = "testdata"
	for name := range cases {
		for _, fname := range strings.Split(name, ",") {
			fname := fname
			t.Run(fname, func(t *testing.T) {
				exp := cases[name]
				file := filepath.Join(rootDir, fname)
				val, err := ParseFile(file, Debug(os.Getenv("AML_TEST_DEBUG") == "true"), Recover(false))
				if err != nil {
					if expErr, ok := exp.(error); ok {
						assert.Contains(t, err.Error(), expErr.Error())
					} else {
						t.Error(err)
					}
					return
				}
				v := val.(ast.Value)
				v.Accept(clearPosition{t})
				expStr, _ := json.MarshalIndent(exp, "", "  ")
				valStr, _ := json.MarshalIndent(v, "", "  ")
				assert.Equal(t, string(expStr), string(valStr))
			})
		}
	}
}

func ptr[T any](v T) *T {
	return &v
}
