package tree

import (
	"bytes"
	"errors"
	"io"
	"sort"
	"unsafe"

	"vimagination.zapto.org/byteio"
)

type Tree struct {
	r                                io.ReaderAt
	children, names, ptrs, data, ptr int64
	nameData                         [][2]int64
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
	if t.nameData == nil {
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
			return nil, sr.Err
		}

		t.nameData = nameData
		t.names = t.children + sr.Count
	}

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
		return nil, err
	}

	if !found {
		return nil, ErrNotFound
	}

	sr := byteio.LittleEndianReader{Reader: io.NewSectionReader(t.r, t.ptrs+int64(pos)*8, 8)}

	childPtr, _, err := sr.ReadInt64()
	if err != nil {
		return nil, err
	}

	return OpenAt(t.r, childPtr)
}

// Errors
var (
	ErrNotFound = errors.New("child not found")
)
