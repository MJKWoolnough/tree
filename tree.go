package tree

import (
	"io"
	"iter"
	"sort"

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
		writeNode(w, childNode)

		if w.Err != nil {
			return
		}

		c = append(c, child{name: name, pos: w.Count})
	}

	startChildren := w.Count

	writeChildren(w, c)

	if w.Err != nil {
		return
	}

	startData := w.Count

	if _, err := node.WriteTo(w); err != nil {
		w.Err = err

		return
	}

	w.WriteUint64(uint64(startChildren))
	w.WriteUint64(uint64(startData))
}

func writeChildren(w *byteio.StickyLittleEndianWriter, c children) {
	sort.Slice(c, c.Less)

	for _, child := range c {
		w.WriteUintX(uint64(len(child.name)))
	}

	w.WriteUint8(0)

	for _, child := range c {
		w.WriteString(child.name)
	}

	for _, child := range c {
		w.WriteInt64(child.pos)
	}
}
