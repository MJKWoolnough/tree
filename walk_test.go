package tree

import (
	"errors"
	"reflect"
	"slices"
	"testing"
)

func TestWalk(t *testing.T) {
	tree := Branch{
		{"A", Branch{
			{"AA", Branch{
				{"AAA", Branch{
					{"AAAA", Leaf("")},
					{"AAAB", Leaf("")},
				}},
				{"AAB", Branch{
					{"AAAA", Leaf("")},
					{"AAAB", Leaf("")},
				}},
			}},
			{"AB", Branch{
				{"ABA", Leaf("")},
			}},
			{"AC", Branch{
				{"ACA", Leaf("")},
				{"ACB", Leaf("")},
			}},
		}},
		{"B", Leaf("")},
	}

	var paths [][]string

	if err := Walk(&tree, func(path []string, node Node) error {
		paths = append(paths, slices.Clone(path))

		if len(path) > 2 {
			return SkipNode
		}

		if len(paths) >= 7 {
			return SkipAll
		}

		return nil
	}); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	expectation := [][]string{
		{"A"},
		{"A", "AA"},
		{"A", "AA", "AAA"},
		{"A", "AA", "AAB"},
		{"A", "AB"},
		{"A", "AB", "ABA"},
		{"A", "AC"},
	}

	if !reflect.DeepEqual(paths, expectation) {
		t.Errorf("expecting paths %v, got %v", expectation, paths)
	}

	paths = paths[:0]

	customError := errors.New("custom")

	if err := Walk(&tree, func(path []string, node Node) error {
		paths = append(paths, slices.Clone(path))

		if len(path) > 2 {
			return customError
		}

		return nil
	}); err != customError {
		t.Errorf("expecting error %s, got %s", customError, err)
	}
}
