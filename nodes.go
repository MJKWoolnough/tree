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

type multiNode struct {
	name  string
	nodes []Node
}

type Roots []multiNode

func Merge(nodes ...Node) (Roots, error) {
	var (
		b map[string][]Node
		r Roots
	)

	for _, node := range nodes {
		for name, child := range node.Children() {
			if ce, ok := child.(ChildrenError); ok {
				return nil, ce.error
			}

			b[name] = append(b[name], child)
		}
	}

	for name, nodes := range b {
		pos, _ := r.childPos(name)

		r = slices.Insert(r, pos, multiNode{name: name, nodes: nodes})
	}

	return r, nil
}

func (r Roots) childPos(name string) (int, bool) {
	return slices.BinarySearchFunc(r, multiNode{name: name}, func(a, b multiNode) int {
		return strings.Compare(a.name, b.name)
	})
}

func (r Roots) Children() iter.Seq2[string, Node] {
	return func(yield func(string, Node) bool) {
		for _, children := range r {
			if len(children.nodes) == 1 {
				if !yield(children.name, children.nodes[0]) {
					return
				}

				continue
			}

			roots, err := Merge(children.nodes...)
			if err != nil {
				yield(children.name, ChildrenError{err})

				return
			}

			yield(children.name, roots)
		}
	}
}

func (Roots) WriteTo(_ io.Writer) (int64, error) {
	return 0, nil
}

func (r Roots) Child(name string) (Node, error) {
	pos, exists := r.childPos(name)
	if !exists {
		return nil, ChildNotFoundError(name)
	}

	if len(r[pos].nodes) == 1 {
		return r[pos].nodes[0], nil
	}

	return Merge(r[pos].nodes...)
}

func (Roots) Data() []byte {
	return nil
}

func (Roots) DataLen() int64 {
	return 0
}

func (r Roots) NumChildren() int {
	return len(r)
}
