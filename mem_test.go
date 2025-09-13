package tree

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
)

func TestOpenMem(t *testing.T) {
	for n, test := range openTests {
		var buf bytes.Buffer

		Serialise(&buf, &test)

		node, err := OpenMem(buf.Bytes())
		if err != nil {
			t.Fatalf("test %d: unexpected error: %s", n+1, err)
		}

		tree := readTree(node)

		if !reflect.DeepEqual(test, tree) {
			t.Errorf("test %d: no match", n+1)
		}
	}
}

func TestMemChild(t *testing.T) {
	var buf bytes.Buffer

	Serialise(&buf, testChild)

Loop:
	for n, test := range childTests {
		node, err := OpenMem(buf.Bytes())
		if err != nil {
			t.Fatalf("test %d: unexpected error: %s", n+1, err)
		}

		for m := range test.key {
			child, err := node.Child(test.key[m])
			if !errors.Is(err, test.errors[m]) {
				t.Errorf("test %d.%d: expecting error %v, got %v", n+1, m+1, test.errors[m], err)
			}

			if err != nil {
				continue Loop
			}

			node = child
		}

		var data bytes.Buffer

		node.WriteTo(&data)

		if !bytes.Equal(data.Bytes(), test.data) {
			t.Errorf("test %d: expecting data %q, got %q", n+1, test.data, data.Bytes())
		} else if numChildren := node.NumChildren(); numChildren != test.numChildren {
			t.Errorf("test %d: expecting %d children, got %d", n+1, test.numChildren, numChildren)
		} else if dataLen := node.DataLen(); dataLen != test.dataLen {
			t.Errorf("test %d: expecting %d bytes of data, got %d", n+1, test.dataLen, dataLen)
		}
	}
}
