package tree

import (
	"bytes"
	"io"
	"iter"
	"reflect"
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

func TestSerialise(t *testing.T) {
	for n, test := range [...]struct {
		Input  Node
		Output []byte
		Error  error
	}{
		{
			Input: &node{},
		}, // 1
		{ // 2
			Input: &node{
				data: []byte("ABC"),
			},
			Output: []byte{'A', 'B', 'C', 3, 0x20 | 1},
		},
		{ // 3
			Input: &node{
				children: []node{
					{
						name: "Child1",
						data: []byte("123"),
					},
				},
			},
			Output: []byte{'1', '2', '3', 3, 0x20 | 1, 'C', 'h', 'i', 'l', 'd', '1', 5, 6 << 3, 1, 0x40 | 1},
		},
		{ // 4
			Input: &node{
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
			Output: []byte{'1', '2', '3', 3, 0x20 | 1, 'q', 'w', 'e', 'r', 't', 'y', 6, 0x20 | 1, 'C', 'h', 'i', 'l', 'd', '1', 'c', 'h', 'i', 'l', 'd', '-', '2', 5, 13, 6 << 3, 7 << 3, 2, 0x40 | 1},
		},
		{ // 5
			Input: &node{
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
			Output: []byte{'q', 'w', 'e', 'r', 't', 'y', 6, 0x20 | 1, '1', '2', '3', 3, 0x20 | 1, 'C', 'h', 'i', 'l', 'd', '1', 'c', 'h', 'i', 'l', 'd', '-', '2', 13, 8, 6 << 3, 7 << 3, 'a', 'b', 'c', 2, 3, 0x60 | 2},
		},
		{ // 6
			Input: &node{
				children: []node{
					{
						name: "Child1",
					},
					{
						name: "Child1",
					},
				},
			},
			Error: DuplicateChildError{"Child1"},
		},
		{ // 7
			Input: &node{
				children: []node{
					{
						name: "Child1",
						children: []node{
							{
								name: "SubChild1",
							},
							{
								name: "SubChild1",
							},
						},
					},
					{
						name: "Child2",
					},
				},
			},
			Error: DuplicateChildError{"Child1", "SubChild1"},
		},
		{ // 8
			Input: errorWriter{},
			Error: io.ErrShortWrite,
		},
	} {
		var buf bytes.Buffer

		if err := Serialise(&buf, test.Input); !reflect.DeepEqual(err, test.Error) {
			t.Errorf("test %d: expected error %v, got %v", n+1, test.Error, err)
		} else if written := buf.Bytes(); !bytes.Equal(written, test.Output) {
			t.Errorf("test %d: expecting to have written %v, wrote %v", n+1, test.Output, written)
		}
	}
}

type errorWriter struct{}

func (errorWriter) Children() iter.Seq2[string, Node] {
	return noChildren
}

func (errorWriter) WriteTo(_ io.Writer) (int64, error) {
	return 0, io.ErrShortWrite
}
