package tree

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"iter"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"vimagination.zapto.org/byteio"
)

type Tree struct {
	r                         io.ReaderAt
	children, ptrs, data, ptr int64

	mu       sync.Mutex
	nameData [][2]int64
	err      atomic.Pointer[error]
}

func OpenAt(r io.ReaderAt, pos int64) *Tree {
	if pos == 0 {
		r = nil
	}

	return &Tree{r: r, ptr: pos, data: -1}
}

type TreeCloser struct {
	Tree
	io.Closer
}

func OpenFile(path string) (*TreeCloser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	pos, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	var (
		r io.ReaderAt = f
		c io.Closer   = f
	)

	if pos == 0 {
		f.Close()

		r = nil
		c = io.NopCloser(nil)
	}

	return &TreeCloser{
		Tree:   Tree{r: r, ptr: pos, data: -1},
		Closer: c,
	}, nil
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

	if len(t.nameData) == 0 {
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

	childrenSize, dataSize, sizes, err := readSizes(t.r, t.ptr)
	if err != nil {
		return err
	}

	t.ptr -= 1 + sizes
	t.data = t.ptr - dataSize
	t.children = childrenSize

	return nil
}

func readSizes(r io.ReaderAt, pos int64) (int64, int64, int64, error) {
	sr := byteio.StickyLittleEndianReader{Reader: io.NewSectionReader(r, pos-1, 1)}
	sizes := int64(sr.ReadUint8())
	hasChildren := sizes&0x40 > 0
	hasData := sizes&0x20 > 0
	sizes &= 0x1f

	sr.Reader = io.NewSectionReader(r, pos-1-sizes, sizes)

	var childrenSize, dataSize int64

	if hasChildren {
		childrenSize = int64(sr.ReadUintX())
	}

	if hasData {
		dataSize = int64(sr.ReadUintX())
	}

	return childrenSize, dataSize, sizes, sr.Err
}

func (t *Tree) initChildren() error {
	if t.nameData != nil || t.children == 0 {
		return nil
	}

	nameData, err := readChildNameSizes(bufio.NewReader(io.NewSectionReader(t.r, t.data-t.children, t.children)), t.children)
	if err != nil {
		return err
	}

	t.nameData = nameData
	t.ptrs = t.data - t.children - int64(len(nameData))*8
	lastName := nameData[len(nameData)-1]
	namesStart := t.ptrs - lastName[0] - lastName[1]

	for n := range nameData {
		nameData[n][0] += namesStart
	}

	return nil
}

func readChildNameSizes(r io.Reader, length int64) ([][2]int64, error) {
	var nameData [][2]int64
	var nextStart int64

	sr := byteio.StickyLittleEndianReader{Reader: r}

	for sr.Count < length {
		l := int64(sr.ReadUintX())
		nameData = append(nameData, [2]int64{nextStart, l})
		nextStart += l
	}

	return nameData, sr.Err
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
		t.err.Store(&err)

		return noChildren
	}

	if len(t.nameData) == 0 {
		return noChildren
	}

	return t.iterChildren
}

func (t *Tree) iterChildren(yield func(string, Node) bool) {
	var sb strings.Builder

	namesStart := t.nameData[0][0]
	nameReader := io.NewSectionReader(t.r, namesStart, t.ptrs-namesStart)
	ptrReader := byteio.LittleEndianReader{Reader: io.NewSectionReader(t.r, t.ptrs, t.data-t.ptrs)}

	for _, child := range t.nameData {
		_, err := io.CopyN(&sb, nameReader, child[1])
		if err != nil {
			t.err.Store(&err)

			return
		}

		ptr, _, err := ptrReader.ReadInt64()
		if err != nil {
			t.err.Store(&err)

			return
		}

		if !yield(sb.String(), OpenAt(t.r, ptr)) {
			return
		}

		sb.Reset()
	}
}

func (t *Tree) Err() error {
	err := t.err.Load()
	if err == nil {
		return nil
	}

	return *err
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
