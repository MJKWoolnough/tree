package tree

import (
	"bytes"
	"io"
	"iter"
	"slices"
	"unsafe"

	"vimagination.zapto.org/byteio"
)

// MemTree represents a tree backed by an in-memory byte slice.
type MemTree struct {
	tree  []byte
	data  []byte
	names []string
	ptrs  [][]byte
}

// OpenMem opens a Tree from the given byte slice.
func OpenMem(data []byte) (*MemTree, error) {
	return OpenMemAt(data, int64(len(data)))
}

// OpenMemAt opens a Tree from the given byte slice, using the given Node
// pointer instead of using the length of the data.
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

	if len(nameData) == 0 {
		return nil
	}

	lastName := nameData[len(nameData)-1]
	ptrs := start - lastName.ptrStart - int64(lastName.ptrLength)
	namesStart := ptrs - lastName.nameStart - lastName.nameLength
	m.names = make([]string, len(nameData))
	m.ptrs = make([][]byte, len(nameData))

	for n, name := range nameData {
		m.names[n] = unsafe.String(&data[namesStart+name.nameStart], name.nameLength)
		m.ptrs[n] = data[ptrs : ptrs+int64(name.ptrLength)]
		ptrs += int64(name.ptrLength)
	}

	return nil
}

// WriteTo will pass the Nodes data to the given io.Writer as a single
// byte-slice.
func (m *MemTree) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(m.data)

	return int64(n), err
}

// Data returns the Nodes data.
func (m *MemTree) Data() []byte {
	return m.data
}

// Child attempts to retrieve a child Node corresponding to the given name.
//
// If no child matches the given name, the returned error will be of type
// ChildNotFoundError.
func (m *MemTree) Child(name string) (*MemTree, error) {
	pos, found := slices.BinarySearch(m.names, name)
	if !found {
		return nil, ChildNotFoundError(name)
	}

	ptr, err := readPointer(m.ptrs[pos])
	if err != nil {
		return nil, err
	}

	return OpenMemAt(m.tree, ptr)
}

func readPointer(ptr []byte) (int64, error) {
	ler := byteio.LittleEndianReader{Reader: bytes.NewReader(ptr)}

	return readChildPointer(&ler, uint8(len(ptr)))
}

// Children returns an iterator that loops through all of the child Nodes.
//
// Read errors will be expressed with a final Node of underlying type
// ChildrenError.
func (m *MemTree) Children() iter.Seq2[string, Node] {
	return func(yield func(string, Node) bool) {
		for n, name := range m.names {
			ptr, err := readPointer(m.ptrs[n])
			if err != nil {
				yield(name, ChildrenError{err})

				return
			}

			tree, err := OpenMemAt(m.tree, ptr)
			if err != nil {
				yield(name, ChildrenError{err})

				return
			}

			if !yield(name, tree) {
				break
			}
		}
	}
}

// ChildNames returns an iterator that loops through the names of the child
// Nodes.
func (m *MemTree) ChildNames() iter.Seq[string] {
	return slices.Values(m.names)
}

// DataLen returns the length of the data stored on this Node.
func (m *MemTree) DataLen() int64 {
	return int64(len(m.data))
}

// NumChildren returns the number of child Nodes that are attached to this Node.
func (m *MemTree) NumChildren() int {
	return len(m.names)
}
