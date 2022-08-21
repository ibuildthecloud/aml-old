package eval

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acorn-io/aml/parser"
	"github.com/acorn-io/aml/parser/ast"
	"github.com/stretchr/testify/assert"
)

func TestEval(t *testing.T) {
	const dir = "./testdata/eval"
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		t.Run(strings.TrimSuffix(file.Name(), ".test"), func(t *testing.T) {
			bytes, err := ioutil.ReadFile(filepath.Join(dir, file.Name()))
			if err != nil {
				t.Fatal(err)
			}
			expected := string(bytes)
			i := strings.Index(expected, "---")
			input := expected[:i]
			expected = expected[i+4:]

			node, err := parser.Parse(file.Name(), []byte(input))
			if err != nil {
				t.Fatal(err)
			}
			v, err := ToValue(context.Background(), &Scope{}, node.(*ast.Value))
			if err != nil {
				t.Fatal(err)
			}
			iface, err := v.Interface(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			result, err := json.MarshalIndent(iface, "", "    ")
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, strings.TrimSpace(expected), string(result))
		})
	}
}
