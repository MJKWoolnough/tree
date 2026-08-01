package tree

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"testing"
)

func TestRoots(t *testing.T) {
	var branches [7]Branch

	branches[0].Add("Leaf1", Leaf("ABC"))
	branches[0].Add("Leaf2", Leaf("DEF"))
	branches[1].Add("Leaf1", Leaf("GHI"))
	branches[2].Add("BranchA", branches[0])
	branches[2].Add("BranchB", branches[1])
	branches[3].Add("Parent", branches[2])

	branches[4].Add("Leaf3", Leaf("JKL"))
	branches[5].Add("BranchA", branches[4])
	branches[6].Add("Parent", branches[5])

	tree, err := Merge(branches[3], branches[6])
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	expected := node{
		children: []node{
			{
				name: "Parent",
				children: []node{
					{
						name: "BranchA",
						children: []node{
							{
								name: "Leaf1",
								data: []byte("ABC"),
							},
							{
								name: "Leaf2",
								data: []byte("DEF"),
							},
							{
								name: "Leaf3",
								data: []byte("JKL"),
							},
						},
					},
					{
						name: "BranchB",
						children: []node{
							{
								name: "Leaf1",
								data: []byte("GHI"),
							},
						},
					},
				},
			},
		},
	}

	if read := readTree(tree); !reflect.DeepEqual(read, expected) {
		t.Errorf("no match")
	}

	if _, err := branches[2].Child("BranchA"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if _, err := branches[2].Child("BranchZ"); !errors.Is(err, ChildNotFoundError("BranchZ")) {
		t.Errorf("expected ChildNotFoundError(BranchZ), got %v", err)
	}

	if _, err := tree.Child("Parent"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if _, err := tree.Child("BranchZ"); !errors.Is(err, ChildNotFoundError("BranchZ")) {
		t.Errorf("expected ChildNotFoundError(BranchZ), got %v", err)
	}
}

func TestChild(t *testing.T) {
	var buf bytes.Buffer

	branch := Branch{nameNode{Name: "Child", Node: Leaf{}}}

	Serialise(&buf, Branch{nameNode{Name: "Child", Node: Leaf{}}})

	f, _ := os.CreateTemp(t.TempDir(), "")

	f.Write(buf.Bytes())
	f.Close()

	m, _ := OpenMem(buf.Bytes())
	tree, _ := OpenFile(f.Name())
	root, _ := Merge(branch, m)

	for n, test := range [...]Node{m, tree, branch, root} {
		if _, err := Child(test, "Child"); err != nil {
			t.Errorf("test %d: unexpected error: %v", n+1, err)
		}

		if _, err := Child(test, "NoChild"); !errors.Is(err, ChildNotFoundError("NoChild")) {
			t.Errorf("test %d: expected ChildNotFoundError(NoChild), got %v", n+1, err)
		}
	}
}
