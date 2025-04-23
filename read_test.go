package tree

import (
	"bytes"
	"errors"
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

func TestChild(t *testing.T) {
	var buf bytes.Buffer

	WriteTree(&buf, &node{
		children: []node{
			{
				name: "A1",
				data: []byte("123"),
				children: []node{
					{
						name: "B1",
						data: []byte("456"),
					},
					{
						name: "B2",
						data: []byte("789"),
					},
					{
						name: "B3",
						data: []byte("ABC"),
					},
				},
			},
			{
				name: "A2",
				data: []byte("DEF"),
				children: []node{
					{
						name: "B1",
						data: []byte("GHI"),
					},
					{
						name: "B2",
						data: []byte("JKL"),
					},
				},
			},
		},
		data: []byte("MNO"),
	})

Loop:
	for n, test := range [...]struct {
		key    []string
		data   []byte
		errors []error
	}{
		{
			data: []byte("MNO"),
		},
		{
			key:    []string{"A1"},
			data:   []byte("123"),
			errors: []error{nil},
		},
		{
			key:    []string{"A1", "B1"},
			data:   []byte("456"),
			errors: []error{nil, nil},
		},
		{
			key:    []string{"A1", "B2"},
			data:   []byte("789"),
			errors: []error{nil, nil},
		},
		{
			key:    []string{"A2", "B2"},
			data:   []byte("JKL"),
			errors: []error{nil, nil},
		},
		{
			key:    []string{"A2", "B3"},
			data:   []byte("JKL"),
			errors: []error{nil, ErrNotFound},
		},
		{
			key:    []string{"A2", "B2", "C1"},
			data:   []byte("JKL"),
			errors: []error{nil, nil, ErrNotFound},
		},
	} {
		node := OpenAt(bytes.NewReader(buf.Bytes()), int64(buf.Len()))

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
		}
	}
}
