package tree

import (
	"bytes"
	"errors"
	"io"
	"iter"
	"testing"
)

type node struct {
	name     string
	children []node
	data     []byte
}

func (nd *node) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(nd.data)

	return int64(n), err
}

func (nd *node) Children() iter.Seq2[string, Node] {
	return func(yield func(string, Node) bool) {
		for _, child := range nd.children {
			if !yield(child.name, &child) {
				break
			}
		}
	}
}

func TestWriteTree(t *testing.T) {
	for n, test := range [...]struct {
		Input  node
		Output []byte
		Error  error
	}{
		{},
		{
			Input: node{
				data: []byte("ABC"),
			},
			Output: []byte{'A', 'B', 'C', 0, 3, 2},
		},
		{
			Input: node{
				children: []node{
					{
						name: "Child1",
						data: []byte("123"),
					},
				},
			},
			Output: []byte{'1', '2', '3', 0, 3, 2, 'C', 'h', 'i', 'l', 'd', '1', 6, 0, 0, 0, 0, 0, 0, 0, 6, 1, 0, 2},
		},
		{
			Input: node{
				children: []node{
					{
						name: "Child1",
						data: []byte("123"),
					},
					{
						name: "child-2",
						data: []byte("qwerty"),
					},
				},
			},
			Output: []byte{'1', '2', '3', 0, 3, 2, 'q', 'w', 'e', 'r', 't', 'y', 0, 6, 2, 'C', 'h', 'i', 'l', 'd', '1', 'c', 'h', 'i', 'l', 'd', '-', '2', 6, 0, 0, 0, 0, 0, 0, 0, 15, 0, 0, 0, 0, 0, 0, 0, 6, 7, 2, 0, 2},
		},
		{
			Input: node{
				children: []node{
					{
						name: "child-2",
						data: []byte("qwerty"),
					},
					{
						name: "Child1",
						data: []byte("123"),
					},
				},
				data: []byte("abc"),
			},
			Output: []byte{'q', 'w', 'e', 'r', 't', 'y', 0, 6, 2, '1', '2', '3', 0, 3, 2, 'C', 'h', 'i', 'l', 'd', '1', 'c', 'h', 'i', 'l', 'd', '-', '2', 15, 0, 0, 0, 0, 0, 0, 0, 9, 0, 0, 0, 0, 0, 0, 0, 6, 7, 'a', 'b', 'c', 2, 3, 2},
		},
		{
			Input: node{
				children: []node{
					{
						name: "Child1",
					},
					{
						name: "Child1",
					},
				},
			},
			Error: ErrDuplicateChildName,
		},
	} {
		var buf bytes.Buffer

		if err := WriteTree(&buf, &test.Input); !errors.Is(err, test.Error) {
			t.Errorf("test %d: expected error %q, got %q", n+1, test.Error, err)
		} else if written := buf.Bytes(); !bytes.Equal(written, test.Output) {
			t.Errorf("test %d: expecting to have written %v, wrote %v", n+1, test.Output, written)
		}
	}
}
