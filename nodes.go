package tree

import (
	"io"
	"iter"
	"slices"
	"strings"
)

type Leaf []byte

func (Leaf) Children() iter.Seq2[string, Node] {
	return noChildren
}

func (l Leaf) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(l)

	return int64(n), err
}

func (l Leaf) Data() []byte {
	return l
}

func (l Leaf) DataLen() int64 {
	return int64(len(l))
}

func (Leaf) Child(name string) (Node, error) {
	return nil, ChildNotFoundError(name)
}

func (Leaf) NumChildren() int {
	return 0
}

type nameNode struct {
	Name string
	Node
}

type Branch []nameNode

func (b *Branch) Add(name string, node Node) error {
	pos, exists := slices.BinarySearchFunc(*b, nameNode{Name: name}, func(a, b nameNode) int {
		return strings.Compare(a.Name, b.Name)
	})

	if exists {
		return DuplicateChildError{name}
	}

	*b = slices.Insert(*b, pos, nameNode{Name: name, Node: node})

	return nil
}

func (b Branch) Children() iter.Seq2[string, Node] {
	return func(yield func(string, Node) bool) {
		for _, nn := range b {
			if !yield(nn.Name, nn) {
				break
			}
		}
	}
}

func (Branch) WriteTo(_ io.Writer) (int64, error) {
	return 0, nil
}

func (b Branch) Child(name string) (Node, error) {
	pos, exists := slices.BinarySearchFunc(b, nameNode{Name: name}, func(a, b nameNode) int {
		return strings.Compare(a.Name, b.Name)
	})

	if !exists {
		return nil, ChildNotFoundError(name)
	}

	return b[pos], nil
}

func (Branch) Data() []byte {
	return nil
}

func (Branch) DataLen() int64 {
	return 0
}

func (b Branch) NumChildren() int {
	return len(b)
}
