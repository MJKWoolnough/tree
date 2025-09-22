package tree_test

import (
	"bytes"
	"fmt"

	"vimagination.zapto.org/tree"
)

func Example() {
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
