package tree

import (
	"bytes"
	"io"
	"iter"
	"slices"
	"sort"
	"strings"
	"unsafe"

	"vimagination.zapto.org/byteio"
)

type Node interface {
	Children() iter.Seq2[string, Node]
	io.WriterTo
}

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

type DuplicateChildError []string

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
			return bytes.Compare(unsafe.Slice(unsafe.StringData(a.name), len(a.name)), unsafe.Slice(unsafe.StringData(b.name), len(b.name)))
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

	if errr, ok := node.(interface{ Err() error }); ok {
		if err := errr.Err(); err != nil {
			w.Err = err

			return 0, 0
		}
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
