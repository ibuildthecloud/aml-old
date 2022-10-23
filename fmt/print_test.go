package fmt

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/acorn-io/aml/parser"
	"github.com/acorn-io/aml/parser/ast"
	"github.com/stretchr/testify/assert"
)

func TestFmt(t *testing.T) {
	const dir = "./testdata"
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		t.Run(strings.TrimSuffix(file.Name(), ".test"), func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(dir, file.Name()))
			if err != nil {
				t.Fatal(err)
			}
			expected := string(data)
			i := strings.Index(expected, "---")
			input := expected[:i]
			expected = expected[i+4:]
			expected = strings.ReplaceAll(expected, "    ", "\t")

			node, err := parser.Parse(file.Name(), []byte(input), parser.GlobalStore("source", file.Name()))
			if err != nil {
				t.Fatal(err)
			}
			out := &bytes.Buffer{}
			err = Print(out, node.(*ast.Value))
			if err != nil {
				t.Fatal(err)
			}
			fmt.Println(out.String())
			assert.Equal(t, expected, out.String())
		})
	}
}
