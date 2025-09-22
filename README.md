# tree

[![CI](https://github.com/MJKWoolnough/tree/actions/workflows/go-checks.yml/badge.svg)](https://github.com/MJKWoolnough/tree/actions)
[![Go Reference](https://pkg.go.dev/badge/vimagination.zapto.org/tree.svg)](https://pkg.go.dev/vimagination.zapto.org/tree)
[![Go Report Card](https://goreportcard.com/badge/vimagination.zapto.org/tree)](https://goreportcard.com/report/vimagination.zapto.org/tree)

--
    import "vimagination.zapto.org/tree"

Package tree implements a tree serialiser and reader. Usable as a tree-based, [WORM](https://en.wikipedia.org/wiki/Write_once_read_many), key-value store.

## Highlights

 - Serialise trees using built-in data types `Branch` and `Leaf`, or any implementation of the two method `Node` interface.
 - Can read trees from files, with `OpenFile`, from a bytes-slice with `OpenMemAt`, or from any `io.ReaderAt`, with `OpenAt`.
 - Can store data on any node, be it a branch or a leaf node.

## Usage

```go
package main

import (
	"bytes"
	"fmt"

	"vimagination.zapto.org/tree"
)

func main() {
	var (
		buf, readBuf bytes.Buffer
		root         tree.Branch
		branch       tree.Branch
	)

	root.Add("child1", tree.Leaf([]byte("Hello")))
	root.Add("child2", tree.Leaf([]byte("World")))
	root.Add("branch1", &branch)

	branch.Add("childA", tree.Leaf([]byte("Foo")))
	branch.Add("childB", tree.Leaf([]byte("Bar")))

	tree.Serialise(&buf, root)

	t := tree.OpenAt(bytes.NewReader(buf.Bytes()), int64(buf.Len()))

	child1, _ := t.Child("child1")

	child1.WriteTo(&readBuf)

	fmt.Printf("child 1 data: %q\n", readBuf.Bytes())

	readBuf.Reset()

	branch1, _ := t.Child("branch1")
	childB, _ := branch1.Child("childB")

	childB.WriteTo(&readBuf)

	fmt.Printf("child B data: %q\n", readBuf.Bytes())

	// Output:
	// child 1 data: "Hello"
	// child B data: "Bar"
}

```

# Binary Format

┌───────────────────────────────────────────────────────────┐
│ Node Data                                                 │
├───────────────────────────────────────────────────────────┤
│ Names Section                                             │
│  ├─ Name0: Name of Child0 node (bytes)                    │
│  ├─ Name1: Name of Child1 node (bytes)                    │
│  └─ …                                                     │
├───────────────────────────────────────────────────────────┤
│ Pointers Section                                          │
│  ├─ Pointer0: int64 offset to end of Child0 node (varint) │
│  ├─ Pointer1: int64 offset to end of Child1 node (varint) │
│  └─ …                                                     │
├───────────────────────────────────────────────────────────┤
│ NameSizes Section                                         │
│  ├─ Size of Name0 << 3 & Size of Ptr0 (varint)            │
│  ├─ Size of Name1 << 3 & Size of Ptr1 (varint)            │
│  └─ …                                                     │
├───────────────────────────────────────────────────────────┤
│ Data Section                                              │
│  └─ Bytes of the data stored on this node                 │
├───────────────────────────────────────────────────────────┤
│ Sizes Section                                             │
│  ├─ Size of NameSizes section (varint); only if > 0       │
│  └─ Size of Data section (varint); only if > 0            │
├───────────────────────────────────────────────────────────┤
│ Size Flags (uint8)                                        │
│  ├─ Bits 0-5: Size of the Sizes section in bytes          │
│  ├─ Bit 6: 1 when there size of data > 0; 0 otherwise     │
│  └─ Bit 7: 1 when there are children; 0 otherwise         │
└───────────────────────────────────────────────────────────┘

NB: Pointers to leaf nodes with no data will be 0.

## Documentation

Full API docs can be found at:

https://pkg.go.dev/vimagination.zapto.org/tree
