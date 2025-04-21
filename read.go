package tree

import (
	"bytes"
	"errors"
	"io"
	"sort"
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
}

func OpenAt(r io.ReaderAt, pos int64) (*Tree, error) {
	sr := byteio.StickyLittleEndianReader{Reader: io.NewSectionReader(r, pos-16, 16)}

	children := sr.ReadInt64()
	data := sr.ReadInt64()

	if sr.Err != nil {
		return nil, sr.Err
	}

	return &Tree{r: r, children: children, data: data, ptr: pos}, nil
}

func (t *Tree) WriteTo(w io.Writer) (int64, error) {
	return io.Copy(w, io.NewSectionReader(t.r, t.data, t.ptr-t.data-16))
}

func (t *Tree) Child(name string) (*Tree, error) {
	if err := t.initChildren(); err != nil {
		return nil, err
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

	return OpenAt(t.r, childPtr)
}

func (t *Tree) initChildren() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.nameData != nil {
		return nil
	}

	var nameData [][2]int64
	var start int64

	sr := byteio.StickyLittleEndianReader{Reader: io.NewSectionReader(t.r, t.children, t.data-t.children)}

	for {
		l := int64(sr.ReadUintX())
		if l == 0 {
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

	return nil
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

// Errors
var (
	ErrNotFound = errors.New("child not found")
)
