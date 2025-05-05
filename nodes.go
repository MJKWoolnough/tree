package tree

import (
	"io"
	"iter"
	"slices"
	"strings"
)

// Leaf represents a childless Node that contains only data.
//
// The Lead itself is a byte-slice.
type Leaf []byte

// Children will return an empty iterator for Leaf Nodes.
func (Leaf) Children() iter.Seq2[string, Node] {
	return noChildren
}

// WriteTo will pass the Nodes data to the given io.Writer as a single
// byte-slice.
func (l Leaf) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(l)

	return int64(n), err
}

// Data returns the Nodes data.
func (l Leaf) Data() []byte {
	return l
}

// DataLen returns the length of the data stored on this Node.
func (l Leaf) DataLen() int64 {
	return int64(len(l))
}

// Child will always return nil with a ChildNotFoundError error for a Leaf Node.
func (Leaf) Child(name string) (Node, error) {
	return nil, ChildNotFoundError(name)
}

// NumChildren will always return 0 for a Leaf Node.
func (Leaf) NumChildren() int {
	return 0
}

type nameNode struct {
	Name string
	Node
}

func (n nameNode) compare(m nameNode) int {
	return strings.Compare(n.Name, m.Name)
}

// Branch is a collection of named Nodes.
type Branch []nameNode

// Add adds a named Node to the branch.
//
// No locking takes place, so all children should be added before using the
// Branch Node.
func (b *Branch) Add(name string, node Node) error {
	pos, exists := slices.BinarySearchFunc(*b, nameNode{Name: name}, nameNode.compare)
	if exists {
		return DuplicateChildError{name}
	}

	*b = slices.Insert(*b, pos, nameNode{Name: name, Node: node})

	return nil
}

// Children returns an iterator that loops through all of the child Nodes.
func (b Branch) Children() iter.Seq2[string, Node] {
	return func(yield func(string, Node) bool) {
		for _, nn := range b {
			if !yield(nn.Name, nn.Node) {
				break
			}
		}
	}
}

// WriteTo always returns 0, nil for a Branch Node.
func (Branch) WriteTo(_ io.Writer) (int64, error) {
	return 0, nil
}

// Child attempts to retrieve a child Node corresponding to the given name.
//
// If no child matches the given name, the returned error will be of type
// ChildNotFoundError.
func (b Branch) Child(name string) (Node, error) {
	pos, exists := slices.BinarySearchFunc(b, nameNode{Name: name}, func(a, b nameNode) int {
		return strings.Compare(a.Name, b.Name)
	})

	if !exists {
		return nil, ChildNotFoundError(name)
	}

	return b[pos], nil
}

// Data returns the Nodes data.
func (Branch) Data() []byte {
	return nil
}

// DataLen will always return 0 for a Branch Node.
func (Branch) DataLen() int64 {
	return 0
}

// NumChildren returns the number of child Nodes that are attached to this Node.
func (b Branch) NumChildren() int {
	return len(b)
}

type multiNode struct {
	name  string
	nodes []Node
}

func (m multiNode) compare(n multiNode) int {
	return strings.Compare(m.name, n.name)
}

type Roots []multiNode

// Merge combines the children from multiple nodes, merging same named
// children similarly.
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
	return slices.BinarySearchFunc(r, multiNode{name: name}, multiNode.compare)
}

// Children returns an iterator that loops through all of the child Nodes.
//
// Any errors will be expressed with a final Node of underlying type
// ChildrenError.
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

// WriteTo always return 0, nil for a Roots Node.
func (Roots) WriteTo(_ io.Writer) (int64, error) {
	return 0, nil
}

// Child attempts to retrieve a child Node corresponding to the given name.
//
// If no child matches the given name, the returned error will be of type
// ChildNotFoundError.
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

// Data will always return nil for a Roots Node.
func (Roots) Data() []byte {
	return nil
}

// DataLen will always return 0 for a Roots Node.
func (Roots) DataLen() int64 {
	return 0
}

// NumChildren returns the number of child Nodes that are attached to this Node.
func (r Roots) NumChildren() int {
	return len(r)
}
