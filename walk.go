package tree

import "errors"

type WalkFunc func(path []string, n Node) error

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

var (
	SkipNode = errors.New("skip this node")
	SkipAll  = errors.New("stop the walk")
)
