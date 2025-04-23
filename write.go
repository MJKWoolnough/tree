package tree

import (
	"bytes"
	"errors"
	"io"
	"iter"
	"slices"
	"sort"
	"unsafe"

	"vimagination.zapto.org/byteio"
)

type Node interface {
	Children() iter.Seq2[string, Node]
	io.WriterTo
}

func WriteTree(w io.Writer, root Node) error {
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

func writeNode(w *byteio.StickyLittleEndianWriter, node Node) {
	var c children

	for name, childNode := range node.Children() {
		cn := child{name: name}
		childPos, found := slices.BinarySearchFunc(c, cn, func(a, b child) int {
			return bytes.Compare(unsafe.Slice(unsafe.StringData(a.name), len(a.name)), unsafe.Slice(unsafe.StringData(b.name), len(b.name)))
		})

		if found {
			w.Err = ErrDuplicateChildName

			return
		}

		start := w.Count

		writeNode(w, childNode)

		if w.Err != nil {
			return
		}

		cn.pos = w.Count
		if start == cn.pos {
			cn.pos = 0
		}

		c = slices.Insert(c, childPos, cn)
	}

	start := w.Count

	sizeChildren := writeChildren(w, c)

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

		w.WriteUintX(uint64(sizeChildren))
		w.WriteUintX(uint64(startSizes - startData))
		w.WriteUint8(uint8(w.Count - startSizes))
	}
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

// Errors
var (
	ErrDuplicateChildName = errors.New("duplicate child name")
)
