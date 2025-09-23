package tree

import (
	"bytes"
	"errors"
	"reflect"
	"slices"
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

func TestMemChildNames(t *testing.T) {
	var buf bytes.Buffer

	Serialise(&buf, testChild)

	node, err := OpenMem(buf.Bytes())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	childNames := slices.Collect(node.ChildNames())

	if len(childNames) != 2 {
		t.Fatalf("expecting 2 children, got %d", len(childNames))
	}

	for n, expected := range [...]string{"A1", "A2"} {
		if expected != childNames[n] {
			t.Errorf("test %d: expecting name %q, got %q", n+1, expected, childNames[n])
		}
	}
}

func TestLargeTreeMem(t *testing.T) {
	tree := genLargeTree(5)
	tree.name = ""

	var buf bytes.Buffer

	Serialise(&buf, &tree)

	mem, _ := OpenMem(buf.Bytes())
	read := readTree(mem)

	if !reflect.DeepEqual(&tree, &read) {
		t.Errorf("did not read what we wrote")
	}
}
