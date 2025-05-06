package tree

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

var (
	openTests = [...]node{
		{}, // 1
		{ // 2
			data: []byte("ABC"),
		},
		{ // 3
			children: []node{
				{
					name: "",
				},
			},
		},
		{ // 4
			children: []node{
				{
					name: "Child1",
				},
			},
		},
		{ // 5
			children: []node{
				{
					name: "Child1",
					data: []byte("123"),
				},
			},
		},
		{ // 6
			children: []node{
				{
					name: "Child2",
					data: []byte("456"),
				},
			},
		},
		{ // 7
			children: []node{
				{
					name: "Child1",
					data: []byte("123"),
				},
				{
					name: "Child2",
					data: []byte("456"),
				},
			},
		},
		{ // 8
			children: []node{
				{
					name: "Child2",
					data: []byte("456"),
				},
			},
			data: []byte("ABC"),
		},
	}
	testChild = &node{
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
	}
	childTests = [...]struct {
		key    []string
		data   []byte
		errors []error
	}{
		{ // 1
			data: []byte("MNO"),
		},
		{ // 2
			key:    []string{"A1"},
			data:   []byte("123"),
			errors: []error{nil},
		},
		{ // 3
			key:    []string{"A1", "B1"},
			data:   []byte("456"),
			errors: []error{nil, nil},
		},
		{ // 4
			key:    []string{"A1", "B2"},
			data:   []byte("789"),
			errors: []error{nil, nil},
		},
		{ // 5
			key:    []string{"A2", "B2"},
			data:   []byte("JKL"),
			errors: []error{nil, nil},
		},
		{ // 6
			key:    []string{"A2", "B3"},
			errors: []error{nil, ChildNotFoundError("B3")},
		},
		{ // 7
			key:    []string{"A2", "B2", "C1"},
			errors: []error{nil, nil, ChildNotFoundError("C1")},
		},
	}
)

func TestOpenAt(t *testing.T) {
	for n, test := range openTests {
		var buf bytes.Buffer

		Serialise(&buf, &test)

		tree := readTree(OpenAt(bytes.NewReader(buf.Bytes()), int64(buf.Len())))

		if !reflect.DeepEqual(test, tree) {
			t.Errorf("test %d: no match", n+1)
		}
	}
}

func TestOpenFile(t *testing.T) {
	tmp := t.TempDir()

	for n, test := range openTests {
		path := filepath.Join(tmp, strconv.Itoa(n))

		f, err := os.Create(path)
		if err != nil {
			t.Fatalf("test %d: unexpected error creating file (%s): %s", n+1, path, err)
		}

		if err = Serialise(f, &test); err != nil {
			t.Fatalf("test %d: unexpected error serialising tree (%s): %s", n+1, path, err)
		}

		if err = f.Close(); err != nil {
			t.Fatalf("test %d: unexpected error closing file (%s): %s", n+1, path, err)
		}

		node, err := OpenFile(path)
		if err != nil {
			t.Fatalf("test %d: unexpected error opening tree (%s): %s", n+1, path, err)
		}

		tree := readTree(node)

		if !reflect.DeepEqual(test, tree) {
			t.Errorf("test %d: no match", n+1)
		}

		node.Close()
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

	Serialise(&buf, testChild)

Loop:
	for n, test := range childTests {
		node := OpenAt(bytes.NewReader(buf.Bytes()), int64(buf.Len()))

		for m := range test.key {
			child, err := node.Child(test.key[m])
			if !reflect.DeepEqual(err, test.errors[m]) {
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
