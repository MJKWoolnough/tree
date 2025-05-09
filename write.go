// Package tree implements a tree serialiser and reader.
package tree // import "vimagination.zapto.org/tree"

import (
	"io"
	"iter"
	"slices"
	"sort"
	"strings"

	"vimagination.zapto.org/byteio"
)

// Node represents a single node in a Tree.
type Node interface {
	// Children returns an iterator that yields a (unique) name and Node for each
	// of the child nodes.
	//
	// Yielding the children in a lexically sorted order is recommended,
	// but not required.
	//
	// If an error occurs, the Node may be of type ChildrenError, which in
	// addition to being a Node also implements the error interface.
	Children() iter.Seq2[string, Node]

	// WriterTo accepts an io.Writer to which any data stored on the node will be
	// passed.
	io.WriterTo
}

// Serialise writes a tree structure to the given writer.
//
// The byte-format for each node is as follows:
//
//	Names     []string (stored, in lexical order)
//	Pointers  []int64  (pointer to the end (&Size + 1) of each child node record)
//	NameSizes []uint64 (lengths of each name, stored as variable-length integers)
//	Data      []byte
//	Sizes     []uint64 (size of NamesSizes and Data sections, stored as variable-length integers; zeros are omitted)
//	Size      uint8  (lower 5 bits: size of the Sizes field, bit 6: size Data > 0, bit 7: size NameSizes > 0)
//
// NB: All slices are stored without separators.
func Serialise(w io.Writer, root Node) error {
	sw := byteio.StickyLittleEndianWriter{Writer: w}

	writeNode(&sw, root)

	return sw.Err
}

type child struct {
	name string
	pos  int64
}

type children []child

func (c children) Less(i, j int) bool {
	return c[i].name < c[j].name
}

// DuplicateChildError is an error that records the duplicated child name.
type DuplicateChildError []string

// Error implements the error interface.
func (d DuplicateChildError) Error() string {
	return "duplicate child name: " + strings.Join(d, "/")
}

func writeNode(w *byteio.StickyLittleEndianWriter, node Node) {
	start, sizeChildren := getAndWriteChildren(w, node)

	if w.Err != nil {
		return
	}

	startData := w.Count

	if _, err := node.WriteTo(w); err != nil {
		w.Err = err

		return
	}

	if start != w.Count {
		startSizes := w.Count
		dataSize := startSizes - startData

		var toWrite uint8

		if sizeChildren > 0 {
			w.WriteUintX(uint64(sizeChildren))

			toWrite |= 0x40
		}

		if dataSize > 0 {
			w.WriteUintX(uint64(dataSize))

			toWrite |= 0x20
		}

		w.WriteUint8(toWrite | uint8(w.Count-startSizes))
	}
}

func getAndWriteChildren(w *byteio.StickyLittleEndianWriter, node Node) (int64, int64) {
	var c children

	for name, childNode := range node.Children() {
		cn := child{name: name}
		childPos, found := slices.BinarySearchFunc(c, cn, func(a, b child) int {
			return strings.Compare(a.name, b.name)
		})

		if found {
			w.Err = DuplicateChildError{name}

			return 0, 0
		}

		start := w.Count

		writeNode(w, childNode)

		if w.Err != nil {
			if dce, ok := w.Err.(DuplicateChildError); ok {
				w.Err = slices.Insert(dce, 0, name)
			}

			return 0, 0
		}

		cn.pos = w.Count
		if start == cn.pos {
			cn.pos = 0
		}

		c = slices.Insert(c, childPos, cn)
	}

	start := w.Count

	return start, writeChildren(w, c)
}

func writeChildren(w *byteio.StickyLittleEndianWriter, c children) int64 {
	if len(c) == 0 {
		return 0
	}

	sort.Slice(c, c.Less)

	for _, child := range c {
		w.WriteString(child.name)
	}

	for _, child := range c {
		w.WriteInt64(child.pos)
	}

	sizeStart := w.Count

	for _, child := range c {
		w.WriteUintX(uint64(len(child.name)))
	}

	return w.Count - sizeStart
}
