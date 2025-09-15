package tree

import (
	"bufio"
	"bytes"
	"io"
	"iter"
	"os"
	"sort"
	"strings"
	"sync"
	"unsafe"

	"vimagination.zapto.org/byteio"
)

// Tree represents a Node of a tree backed by an io.ReaderAt.
type Tree struct {
	r                         io.ReaderAt
	children, ptrs, data, ptr int64

	mu       sync.Mutex
	nameData []childNameSizes
}

// OffsetReaderAt is a wrapper around the io.ReaderAt interface that will shift
// the read position by the set Offset.
type OffsetReaderAt struct {
	io.ReaderAt
	Offset int64
}

// ReadAt implements the io.ReaderAt interface, but shifts the offset by adding
// OffsetReaderAt.Offset.
func (o *OffsetReaderAt) ReadAt(p []byte, offset int64) (int, error) {
	return o.ReaderAt.ReadAt(p, offset-o.Offset)
}

// OpenAt opens a Tree from the given io.ReaderAt.
//
// The pos should be the length of the data underlying the io.ReaderAt, or a
// specific Node pointer address within the data.
func OpenAt(r io.ReaderAt, pos int64) *Tree {
	if pos == 0 {
		r = nil
	}

	return &Tree{r: r, ptr: pos, data: -1}
}

// TreeCloser is a tree that includes a Close method for an opened file.
type TreeCloser struct {
	Tree
	io.Closer
}

// OpenFile opens a Tree from the given filename.
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

// WriteTo writes the Nodes data to the given writer.
func (t *Tree) WriteTo(w io.Writer) (int64, error) {
	r, err := t.Reader()
	if err != nil {
		return 0, err
	}

	return io.Copy(w, r)
}

// Child attempts to retrieve a child Node corresponding to the given name.
//
// If no child matches the given name, the returned error will be of type
// ChildNotFoundError.
func (t *Tree) Child(name string) (*Tree, error) {
	if t.r == nil {
		return nil, ChildNotFoundError(name)
	}

	if err := t.init(); err != nil {
		return nil, err
	}

	if len(t.nameData) == 0 {
		return nil, ChildNotFoundError(name)
	}

	pos, err := t.getChildIndex(name)
	if err != nil {
		return nil, err
	}

	child := t.nameData[pos]
	sr := byteio.LittleEndianReader{Reader: io.NewSectionReader(t.r, child.ptrStart, int64(child.ptrLength))}

	childPtr, err := readChildPointer(&sr, child.ptrLength)
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
	lastName := nameData[len(nameData)-1]
	t.ptrs = t.data - t.children - lastName.ptrStart - int64(lastName.ptrLength)
	namesStart := t.ptrs - lastName.nameStart - lastName.nameLength

	for n := range nameData {
		nameData[n].nameStart += namesStart
		nameData[n].ptrStart += t.ptrs
	}

	return nil
}

type childNameSizes struct {
	nameStart  int64
	nameLength int64
	ptrStart   int64
	ptrLength  uint8
}

func readChildNameSizes(r io.Reader, length int64) ([]childNameSizes, error) {
	var (
		nameData      []childNameSizes
		nextNameStart int64
		nextPtrStart  int64
	)

	sr := byteio.StickyLittleEndianReader{Reader: r}

	for sr.Count < length {
		ls := sr.ReadUintX()
		l := int64(ls >> 3)
		p := uint8(ls&7) + 1
		nameData = append(nameData, childNameSizes{
			nameStart:  nextNameStart,
			nameLength: l,
			ptrStart:   nextPtrStart,
			ptrLength:  p,
		})
		nextNameStart += l
		nextPtrStart += int64(p)
	}

	return nameData, sr.Err
}

func (t *Tree) getChildIndex(name string) (int64, error) {
	nameBytes := unsafe.Slice(unsafe.StringData(name), len(name))

	var err error

	pos, found := sort.Find(len(t.nameData), func(i int) int {
		tName := make([]byte, t.nameData[i].nameLength)

		_, err = io.ReadFull(io.NewSectionReader(t.r, t.nameData[i].nameStart, int64(len(tName))), tName)
		if err != nil {
			return 0
		}

		return bytes.Compare(nameBytes, tName)
	})

	if err != nil {
		return 0, err
	}

	if !found {
		return 0, ChildNotFoundError(name)
	}

	return int64(pos), nil
}

func noChildren(_ func(string, Node) bool) {}

// Children returns an iterator that loops through all of the child Nodes.
//
// Read errors will be expressed with a final Node of underlying type
// ChildrenError.
func (t *Tree) Children() iter.Seq2[string, Node] {
	if t.r == nil {
		return noChildren
	}

	if err := t.init(); err != nil {
		return func(yield func(string, Node) bool) { yield("", ChildrenError{err}) }
	}

	if len(t.nameData) == 0 {
		return noChildren
	}

	return t.iterChildren
}

func (t *Tree) iterChildren(yield func(string, Node) bool) {
	var sb strings.Builder

	namesStart := t.nameData[0].nameStart
	nameReader := io.NewSectionReader(t.r, namesStart, t.ptrs-namesStart)
	ptrReader := byteio.LittleEndianReader{Reader: io.NewSectionReader(t.r, t.ptrs, t.data-t.ptrs)}

	for _, child := range t.nameData {
		_, err := io.CopyN(&sb, nameReader, child.nameLength)
		if err != nil {
			yield("", ChildrenError{err})

			return
		}

		ptr, err := readChildPointer(&ptrReader, child.ptrLength)
		if err != nil {
			yield(sb.String(), ChildrenError{err})

			return
		}

		if !yield(sb.String(), OpenAt(t.r, ptr)) {
			return
		}

		sb.Reset()
	}
}

func readChildPointer(r *byteio.LittleEndianReader, size uint8) (int64, error) {
	switch size {
	case 1:
		n, _, err := r.ReadUint8()

		return int64(n), err
	case 2:
		n, _, err := r.ReadUint16()

		return int64(n), err
	case 3:
		n, _, err := r.ReadUint24()

		return int64(n), err
	case 4:
		n, _, err := r.ReadUint32()

		return int64(n), err
	case 5:
		n, _, err := r.ReadUint40()

		return int64(n), err
	case 6:
		n, _, err := r.ReadUint48()

		return int64(n), err
	case 7:
		n, _, err := r.ReadUint56()

		return int64(n), err
	default:
		n, _, err := r.ReadUint64()

		return int64(n), err
	}
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

// DataLen returns the length of the data stored on this Node.
func (t *Tree) DataLen() (int64, error) {
	if t.r == nil {
		return 0, nil
	}

	if err := t.initJustData(); err != nil {
		return 0, err
	}

	return t.ptr - t.data, nil
}

// NumChildren returns the number of child Nodes that are attached to this Node.
func (t *Tree) NumChildren() (int, error) {
	if t.r == nil {
		return 0, nil
	}

	if err := t.init(); err != nil {
		return 0, err
	}

	return len(t.nameData), nil
}

// ChildrenError is a Node and error type that is returned from the Children
// iterator.
//
// It has no children and any attempt to retrieve the data will result in the
// underlying error to be returned.
type ChildrenError struct {
	error
}

// NewChildrenError wraps an error to give it the methods of a Node.
func NewChildrenError(err error) ChildrenError {
	return ChildrenError{error: err}
}

// Children always returns an empty iterator.
func (ChildrenError) Children() iter.Seq2[string, Node] {
	return noChildren
}

// WriteTo always returns the underlying error.
func (c ChildrenError) WriteTo(_ io.Writer) (int64, error) {
	return 0, c.error
}

// Unwrap returns the wrapped error.
func (c ChildrenError) Unwrap() error {
	return c.error
}

// ChildNotFoundError contains the name of the child that could not be found.
type ChildNotFoundError string

// Error implements the error interface.
func (c ChildNotFoundError) Error() string {
	return "child not found: " + string(c)
}
