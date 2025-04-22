package tree

import (
	"bytes"
	"reflect"
	"testing"
)

func TestOpenAt(t *testing.T) {
	for n, test := range [...]node{
		{},
		{
			data: []byte("ABC"),
		},
		{
			children: []node{
				{
					name: "",
				},
			},
		},
		{
			children: []node{
				{
					name: "Child1",
				},
			},
		},
		{
			children: []node{
				{
					name: "Child1",
					data: []byte("123"),
				},
			},
		},
		{
			children: []node{
				{
					name: "Child2",
					data: []byte("456"),
				},
			},
		},
		{
			children: []node{
				{
					name: "Child2",
					data: []byte("456"),
				},
			},
			data: []byte("ABC"),
		},
	} {
		var buf bytes.Buffer

		WriteTree(&buf, &test)

		tree := readTree(OpenAt(bytes.NewReader(buf.Bytes()), int64(buf.Len())))

		if !reflect.DeepEqual(test, tree) {
			t.Errorf("test %d: no match", n+1)
		}
	}
}

func readTree(t Node) node {
	var n node

	for name, child := range t.Children() {
		childNode := readTree(child)
		childNode.name = name

		n.children = append(n.children, childNode)
	}

	var buf bytes.Buffer

	t.WriteTo(&buf)

	if buf.Len() > 0 {
		n.data = buf.Bytes()
	}

	return n
}
