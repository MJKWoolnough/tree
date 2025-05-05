package tree

import (
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
}
