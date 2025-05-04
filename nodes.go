package tree

import (
	"io"
	"iter"
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
