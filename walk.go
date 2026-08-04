package tree

import (
	"errors"
	"iter"
	"slices"
)

// WalkFunc is the type of the function called by Walk to visit each Node.
//
// The path argument contains the names of the Nodes leading to the current
// Node.
//
// The returned error controls how the Walk continues.
//
// The SkipNode error will prevent walking the children of the current Node.
//
// The SkipAll error will cancel the remaining walk, but return a nil error
// from Walk.
//
// Any other error will be returned via the Walk function.
type WalkFunc func(path []string, n Node) error

// Walk recursively walks the tree hierarchy, calling the supplied function for
// each Node visited.
//
// See the WalkFunc type for information on the arguments and how the returned
// error is handled.
func Walk(n Node, fn WalkFunc) error {
	if err := walk(n, fn, nil); err != SkipAll {
		return err
	}

	return nil
}

func walk(n Node, fn WalkFunc, path []string) error {
	for name, child := range n.Children() {
		cp := append(path, name)

		switch err := fn(cp, n); err {
		default:
			return err
		case nil:
			if err := walk(child, fn, cp); err != nil {
				return err
			}
		case SkipNode:
		}
	}

	return nil
}

func Flatten(n Node) iter.Seq2[[]string, Node] {
	return func(yield func([]string, Node) bool) {
		Walk(n, func(path []string, n Node) error {
			if !yield(slices.Clone(path), n) {
				return SkipAll
			}

			return nil
		})
	}
}

var (
	// SkipNode tells the Walk to not iterate through the current Nodes children.
	SkipNode = errors.New("skip this node")

	// SkipAll tells the Walk to stop and return a nil error.
	SkipAll = errors.New("stop the walk")
)
