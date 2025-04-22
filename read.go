package tree

import (
	"bytes"
	"errors"
	"io"
	"iter"
	"sort"
	"strings"
	"sync"
	"unsafe"

	"vimagination.zapto.org/byteio"
)

type Tree struct {
	r                         io.ReaderAt
	children, ptrs, data, ptr int64

	mu       sync.Mutex
	names    int64
	nameData [][2]int64
	err      error
}

func OpenAt(r io.ReaderAt, pos int64) *Tree {
	if pos == 0 {
		r = nil
	}

	return &Tree{r: r, ptr: pos, data: -1}
}

func (t *Tree) WriteTo(w io.Writer) (int64, error) {
	r, err := t.Reader()
	if err != nil {
		return 0, err
	}

	return io.Copy(w, r)
}

func (t *Tree) Child(name string) (*Tree, error) {
	if t.r == nil {
		return nil, ErrNotFound
	}

	if err := t.init(); err != nil {
		return nil, err
	}

	if t.children == t.data {
		return nil, ErrNotFound
	}

	pos, err := t.getChildIndex(name)
	if err != nil {
		return nil, err
	}

	sr := byteio.LittleEndianReader{Reader: io.NewSectionReader(t.r, t.ptrs+pos*8, 8)}

	childPtr, _, err := sr.ReadInt64()
	if err != nil {
		return nil, err
	}

	return OpenAt(t.r, childPtr), nil
}

func (t *Tree) init() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if err := t.initData(); err != nil {
		return err
	}

	return t.initChildren()
}

func (t *Tree) initJustData() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.initData()
}

func (t *Tree) initData() error {
	if t.data > -1 {
		return nil
	}

	sr := byteio.StickyLittleEndianReader{Reader: io.NewSectionReader(t.r, t.ptr-1, 1)}
	sizes := int64(sr.ReadUint8())
	sr.Reader = io.NewSectionReader(t.r, t.ptr-1-sizes, sizes)
	childrenSize := int64(sr.ReadUintX())
	dataSize := int64(sr.ReadUintX())

	if sr.Err != nil {
		return sr.Err
	}

	t.ptr -= 1 + sizes
	t.data = t.ptr - dataSize
	t.children = t.data - childrenSize

	return nil
}

func (t *Tree) initChildren() error {
	if t.nameData != nil {
		return nil
	}

	var nameData [][2]int64
	var start int64

	sr := byteio.StickyLittleEndianReader{Reader: io.NewSectionReader(t.r, t.children, t.data-t.children)}

	for {
		l := int64(sr.ReadUintX())
		if l == 0 && len(nameData) > 0 {
			break
		}

		nameData = append(nameData, [2]int64{start, l})
		start += l
	}

	if sr.Err != nil {
		return sr.Err
	}

	t.nameData = nameData
	t.names = t.children + sr.Count
	t.ptrs = t.names + start

	return sr.Err
}

func (t *Tree) getChildIndex(name string) (int64, error) {
	nameBytes := unsafe.Slice(unsafe.StringData(name), len(name))

	var err error

	pos, found := sort.Find(len(t.nameData), func(i int) int {
		tName := make([]byte, t.nameData[i][1])

		_, err = io.ReadFull(io.NewSectionReader(t.r, t.nameData[i][0], int64(len(tName))), tName)
		if err != nil {
			return 0
		}

		return bytes.Compare(nameBytes, tName)
	})

	if err != nil {
		return 0, err
	}

	if !found {
		return 0, ErrNotFound
	}

	return int64(pos), nil
}

func noChildren(_ func(string, Node) bool) {}

func (t *Tree) Children() iter.Seq2[string, Node] {
	if t.r == nil {
		return noChildren
	}

	if err := t.init(); err != nil {
		t.err = err

		return noChildren
	}

	if t.children == t.data {
		return noChildren
	}

	return t.iterChildren
}

func (t *Tree) iterChildren(yield func(string, Node) bool) {
	var sb strings.Builder

	nameReader := io.NewSectionReader(t.r, t.names, t.ptrs-t.names)
	ptrReader := byteio.LittleEndianReader{Reader: io.NewSectionReader(t.r, t.ptrs, t.data-t.ptrs)}

	for _, child := range t.nameData {
		_, err := io.CopyN(&sb, nameReader, child[1])
		if err != nil {
			t.err = err

			return
		}

		ptr, _, err := ptrReader.ReadInt64()
		if err != nil {
			t.err = err

			return
		}

		if !yield(sb.String(), OpenAt(t.r, ptr)) {
			return
		}

		sb.Reset()
	}
}

func (t *Tree) Err() error {
	return t.err
}

type empty struct{}

func (empty) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (t *Tree) Reader() (io.Reader, error) {
	if t.r == nil {
		return empty{}, nil
	}

	if err := t.initJustData(); err != nil {
		return nil, err
	}

	return io.NewSectionReader(t.r, t.data, t.ptr-t.data), nil
}

// Errors
var (
	ErrNotFound = errors.New("child not found")
)
