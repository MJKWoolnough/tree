package tree

import (
	"bytes"
	"io"
	"iter"
	"slices"
	"unsafe"

	"vimagination.zapto.org/byteio"
)

type MemTree struct {
	tree  []byte
	data  []byte
	names []string
	ptrs  [][]byte
	err   error
}

func OpenMemAt(data []byte, pos int64) (*MemTree, error) {
	if pos <= 0 {
		return &MemTree{}, nil
	}

	childrenSize, dataSize, sizes, err := readSizes(bytes.NewReader(data), pos)
	if err != nil {
		return nil, err
	}

	pos -= 1 + sizes
	dataStart := pos - dataSize
	m := &MemTree{
		tree: data,
		data: data[dataStart:pos],
	}

	if childrenSize > 0 {
		if err := m.loadChildren(data, dataStart-childrenSize, childrenSize); err != nil {
			return nil, err
		}
	}

	return m, nil
}

func (m *MemTree) loadChildren(data []byte, start, length int64) error {
	nameData, err := readChildNameSizes(bytes.NewReader(data[start:start+length]), length)
	if err != nil {
		return err
	}

	ptrs := start - int64(len(nameData))*8
	lastName := nameData[len(nameData)-1]
	namesStart := ptrs - lastName[0] - lastName[1]
	m.names = make([]string, len(nameData))
	m.ptrs = make([][]byte, len(nameData))

	for n, name := range nameData {
		m.names[n] = unsafe.String(&data[namesStart+name[0]], name[1])
		m.ptrs[n] = data[ptrs : ptrs+8]
		ptrs += 8
	}

	return nil
}

func OpenMem(data []byte) (*MemTree, error) {
	return OpenMemAt(data, int64(len(data)))
}

func (m *MemTree) WriteTo(w io.Writer) (int64, error) {
	d, err := m.Data()
	if err != nil {
		return 0, err
	}

	n, err := w.Write(d)

	return int64(n), err
}

func (m *MemTree) Data() ([]byte, error) {
	return m.data, nil
}

func (m *MemTree) Child(name string) (*MemTree, error) {
	pos, found := slices.BinarySearch(m.names, name)
	if !found {
		return nil, ErrNotFound
	}

	ptr, err := readPointer(m.ptrs[pos])
	if err != nil {
		return nil, err
	}

	return OpenMemAt(m.tree, ptr)
}

func readPointer(ptr []byte) (int64, error) {
	ler := byteio.LittleEndianReader{Reader: bytes.NewReader(ptr)}
	p, _, err := ler.ReadInt64()

	return p, err
}

func (m *MemTree) Children() iter.Seq2[string, Node] {
	return func(yield func(string, Node) bool) {
		for n, name := range m.names {
			ptr, err := readPointer(m.ptrs[n])
			if err != nil {
				m.err = err

				return
			}

			tree, err := OpenMemAt(m.tree, ptr)
			if err != nil {
				m.err = err

				return
			}

			if !yield(name, tree) {
				break
			}
		}
	}
}

func (m *MemTree) Err() error {
	return m.err
}
