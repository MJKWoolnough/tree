# tree
--
    import "vimagination.zapto.org/tree"

Package tree implements a tree serialiser and reader.

## Usage

#### func  Serialise

```go
func Serialise(w io.Writer, root Node) error
```
Serialise writes a tree structure to the given writer.

The byte-format for each node is as follows:

    Names     []string (stored, in lexical order, without seperators)
    Pointers  []int64  (pointer to the end (&Size + 1) of each child node record)
    NameSizes []uint64 (lengths of each name, stored as variable-length integers)
    Data      []byte
    Sizes     []uint64 (size of NamesSizes and Data sections, stored as variable-length integers)
    Size      []uint8  (size of the Sizes field)

#### type ChildNotFoundError

```go
type ChildNotFoundError string
```

ChildNotFoundError contains the name of the child that could not be found.

#### func (ChildNotFoundError) Error

```go
func (c ChildNotFoundError) Error() string
```
Error implements the error interface.

#### type ChildrenError

```go
type ChildrenError struct {
}
```

ChildrenError is a Node and error type that is returned from the Children
iterator.

It has no children and any attempt to retrieve the data will result in the
underlying error to be returned.

#### func (ChildrenError) Children

```go
func (ChildrenError) Children() iter.Seq2[string, Node]
```
Children always returns an empty iterator.

#### func (ChildrenError) WriteTo

```go
func (c ChildrenError) WriteTo(_ io.Writer) (int64, error)
```
WriteTo always returns the underlying error.

#### type DuplicateChildError

```go
type DuplicateChildError []string
```

DuplicateChildError is an error that records the duplicated child name.

#### func (DuplicateChildError) Error

```go
func (d DuplicateChildError) Error() string
```
Error implements the error interface.

#### type MemTree

```go
type MemTree struct {
}
```

MemTree represents a tree backed by an in-memory byte slice.

#### func  OpenMem

```go
func OpenMem(data []byte) (*MemTree, error)
```
OpenMem opens a Tree from the given byte slice.

#### func  OpenMemAt

```go
func OpenMemAt(data []byte, pos int64) (*MemTree, error)
```
OpenMemAt opens a Tree from the given byte slice, using the given Node pointer
instead of using the length of the data.

#### func (*MemTree) Child

```go
func (m *MemTree) Child(name string) (*MemTree, error)
```
Child attempts to retrieve a child Node corresponding to the given name.

If no child matches the given name, the returned error will be of type
ChildNotFoundError.

#### func (*MemTree) Children

```go
func (m *MemTree) Children() iter.Seq2[string, Node]
```
Children returns an iterator that loops through all of the child Nodes.

Read errors will be expressed with a final Node of underlying type
ChildrenError.

#### func (*MemTree) Data

```go
func (m *MemTree) Data() []byte
```
Data returns the Nodes data.

#### func (*MemTree) WriteTo

```go
func (m *MemTree) WriteTo(w io.Writer) (int64, error)
```
WriteTo will pass the Nodes data to the given io.Writer as a single byte-slice.

#### type Node

```go
type Node interface {
	// Children returns an iterator that yields a name and Node for each of the
	// child nodes.
	//
	// Yielding the children in a lexically sorted order is recommended,
	// but not required.
	//
	// If an error occurs, the Node may be of type ChildrenError, which in
	// addition to being a Node also implements the error interface.
	Children() iter.Seq2[string, Node]

	// WriterTo accepts an io.Writer to which any data stored on the node will be
	// passed.
	io.WriterTo
}
```

Node represents a single node in a Tree.

#### type Tree

```go
type Tree struct {
}
```

Tree represents a Node of a tree backed by an io.ReaderAt.

#### func  OpenAt

```go
func OpenAt(r io.ReaderAt, pos int64) *Tree
```
OpenAt opens a Tree from the given io.ReaderAt.

The pos should be the length of the data underlying the io.ReaderAt, or a
specific Node pointer address within the data.

#### func (*Tree) Child

```go
func (t *Tree) Child(name string) (*Tree, error)
```
Child attempts to retrive a child Node corresponding to the given name.

If no child matches the given name, the returned error will be of type
ChildNotFoundError.

#### func (*Tree) Children

```go
func (t *Tree) Children() iter.Seq2[string, Node]
```
Children returns an iterator that loops through all of the child Nodes.

Read errors will be expressed with a final Node of underlying type
ChildrenError.

#### func (*Tree) Reader

```go
func (t *Tree) Reader() (io.Reader, error)
```

#### func (*Tree) WriteTo

```go
func (t *Tree) WriteTo(w io.Writer) (int64, error)
```
WriteTo writes the Nodes data to the given writer.

#### type TreeCloser

```go
type TreeCloser struct {
	Tree
	io.Closer
}
```

TreeCloser is a tree that includes a Close method for an opened file.

#### func  OpenFile

```go
func OpenFile(path string) (*TreeCloser, error)
```
OpenFile opens a Tree from the given filename.
