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

    Names     []string (stored, in lexical order)
    Pointers  []int64  (pointer to the end (&Size + 1) of each child node record)
    NameSizes []uint64 (lengths of each name, stored as variable-length integers)
    Data      []byte
    Sizes     []uint64 (size of NamesSizes and Data sections, stored as variable-length integers; zeros are omitted)
    Size      uint8  (lower 5 bits: size of the Sizes field, bit 6: size Data > 0, bit 7: size NameSizes > 0)

NB: All slices are stored without separators.

#### type Branch

```go
type Branch []nameNode
```

Branch is a collection of named Nodes.

#### func (*Branch) Add

```go
func (b *Branch) Add(name string, node Node) error
```
Add adds a named Node to the branch.

No locking takes place, so all children should be added before using the Branch
Node.

#### func (Branch) Child

```go
func (b Branch) Child(name string) (Node, error)
```
Child attempts to retrieve a child Node corresponding to the given name.

If no child matches the given name, the returned error will be of type
ChildNotFoundError.

#### func (Branch) Children

```go
func (b Branch) Children() iter.Seq2[string, Node]
```
Children returns an iterator that loops through all of the child Nodes.

#### func (Branch) Data

```go
func (Branch) Data() []byte
```
Data returns the Nodes data.

#### func (Branch) DataLen

```go
func (Branch) DataLen() int64
```
DataLen will always return 0 for a Branch Node.

#### func (Branch) NumChildren

```go
func (b Branch) NumChildren() int
```
NumChildren returns the number of child Nodes that are attached to this Node.

#### func (Branch) WriteTo

```go
func (Branch) WriteTo(_ io.Writer) (int64, error)
```
WriteTo always returns 0, nil for a Branch Node.

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

#### func  NewChildrenError

```go
func NewChildrenError(err error) ChildrenError
```
NewChildrenError wraps an error to give it the methods of a Node.

#### func (ChildrenError) Children

```go
func (ChildrenError) Children() iter.Seq2[string, Node]
```
Children always returns an empty iterator.

#### func (ChildrenError) Unwrap

```go
func (c ChildrenError) Unwrap() error
```
Unwrap returns the wrapped error.

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

#### type Leaf

```go
type Leaf []byte
```

Leaf represents a childless Node that contains only data.

The Lead itself is a byte-slice.

#### func (Leaf) Child

```go
func (Leaf) Child(name string) (Node, error)
```
Child will always return nil with a ChildNotFoundError error for a Leaf Node.

#### func (Leaf) Children

```go
func (Leaf) Children() iter.Seq2[string, Node]
```
Children will return an empty iterator for Leaf Nodes.

#### func (Leaf) Data

```go
func (l Leaf) Data() []byte
```
Data returns the Nodes data.

#### func (Leaf) DataLen

```go
func (l Leaf) DataLen() int64
```
DataLen returns the length of the data stored on this Node.

#### func (Leaf) NumChildren

```go
func (Leaf) NumChildren() int
```
NumChildren will always return 0 for a Leaf Node.

#### func (Leaf) WriteTo

```go
func (l Leaf) WriteTo(w io.Writer) (int64, error)
```
WriteTo will pass the Nodes data to the given io.Writer as a single byte-slice.

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

#### func (*MemTree) ChildNames

```go
func (m *MemTree) ChildNames() iter.Seq[string]
```
ChildNames returns an iterator that loops through the names of the child Nodes.

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

#### func (*MemTree) DataLen

```go
func (m *MemTree) DataLen() int64
```
DataLen returns the length of the data stored on this Node.

#### func (*MemTree) NumChildren

```go
func (m *MemTree) NumChildren() int
```
NumChildren returns the number of child Nodes that are attached to this Node.

#### func (*MemTree) WriteTo

```go
func (m *MemTree) WriteTo(w io.Writer) (int64, error)
```
WriteTo will pass the Nodes data to the given io.Writer as a single byte-slice.

#### type Node

```go
type Node interface {
	// Children returns an iterator that yields a (unique) name and Node for each
	// of the child nodes.
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

#### type Roots

```go
type Roots []multiNode
```


#### func  Merge

```go
func Merge(nodes ...Node) (Roots, error)
```
Merge combines the children from multiple nodes, merging same named children
similarly.

Changes made to the Nodes after merging will not be recognised.

#### func (Roots) Child

```go
func (r Roots) Child(name string) (Node, error)
```
Child attempts to retrieve a child Node corresponding to the given name.

If no child matches the given name, the returned error will be of type
ChildNotFoundError.

#### func (Roots) Children

```go
func (r Roots) Children() iter.Seq2[string, Node]
```
Children returns an iterator that loops through all of the child Nodes.

Any errors will be expressed with a final Node of underlying type ChildrenError.

#### func (Roots) Data

```go
func (Roots) Data() []byte
```
Data will always return nil for a Roots Node.

#### func (Roots) DataLen

```go
func (Roots) DataLen() int64
```
DataLen will always return 0 for a Roots Node.

#### func (Roots) NumChildren

```go
func (r Roots) NumChildren() int
```
NumChildren returns the number of child Nodes that are attached to this Node.

#### func (Roots) WriteTo

```go
func (Roots) WriteTo(_ io.Writer) (int64, error)
```
WriteTo always return 0, nil for a Roots Node.

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
Child attempts to retrieve a child Node corresponding to the given name.

If no child matches the given name, the returned error will be of type
ChildNotFoundError.

#### func (*Tree) Children

```go
func (t *Tree) Children() iter.Seq2[string, Node]
```
Children returns an iterator that loops through all of the child Nodes.

Read errors will be expressed with a final Node of underlying type
ChildrenError.

#### func (*Tree) DataLen

```go
func (t *Tree) DataLen() (int64, error)
```
DataLen returns the length of the data stored on this Node.

#### func (*Tree) NumChildren

```go
func (t *Tree) NumChildren() (int, error)
```
NumChildren returns the number of child Nodes that are attached to this Node.

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
