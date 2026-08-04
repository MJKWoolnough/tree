package tree

import (
	"errors"
	"reflect"
	"slices"
	"strings"
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
					{"AABA", Leaf("")},
					{"AABB", Leaf("")},
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

func TestFlatten(t *testing.T) {
	tree := Branch{
		{"A", Branch{
			{"AA", Branch{
				{"AAA", Branch{
					{"AAAA", Leaf("")},
					{"AAAB", Leaf("")},
				}},
				{"AAB", Branch{
					{"AABA", Leaf("")},
					{"AABB", Leaf("")},
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

	expectation := map[string]Node{
		"A":             navigate(t, tree, []string{"A"}),
		"A/AA":          navigate(t, tree, []string{"A", "AA"}),
		"A/AA/AAA":      navigate(t, tree, []string{"A", "AA", "AAA"}),
		"A/AA/AAA/AAAA": navigate(t, tree, []string{"A", "AA", "AAA", "AAAA"}),
		"A/AA/AAA/AAAB": navigate(t, tree, []string{"A", "AA", "AAA", "AAAB"}),
		"A/AA/AAB":      navigate(t, tree, []string{"A", "AA", "AAB"}),
		"A/AA/AAB/AABA": navigate(t, tree, []string{"A", "AA", "AAB", "AABA"}),
	}
	got := map[string]Node{}

	for name, n := range Flatten(tree) {
		got[strings.Join(name, "/")] = n

		if len(got) == 7 {
			break
		}
	}

	if !reflect.DeepEqual(got, expectation) {
		t.Errorf("expected map %v, got %v", expectation, got)
	}
}

func navigate(t *testing.T, n Node, path []string) Node {
	node, err := Navigate(n, slices.Values(path))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	return node
}

func TestFilter(t *testing.T) {
	tree := Branch{
		{"A", Branch{
			{"AA", Branch{
				{"AAA", Branch{
					{"AAAA", Leaf("")},
					{"AAAB", Leaf("")},
				}},
				{"AAB", Branch{
					{"AABA", Leaf("")},
					{"AABB", Leaf("")},
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

	var leafs [][]string

	for name := range Filter(tree, func(_ []string, n Node) int {
		_, ok := n.(Leaf)

		if ok {
			return 1
		}

		return 0
	}) {
		leafs = append(leafs, name)
	}

	expectation := [][]string{
		{"A", "AA", "AAA", "AAAA"},
		{"A", "AA", "AAA", "AAAB"},
		{"A", "AA", "AAB", "AABA"},
		{"A", "AA", "AAB", "AABB"},
		{"A", "AB", "ABA"},
		{"A", "AC", "ACA"},
		{"A", "AC", "ACB"},
		{"B"},
	}

	if !reflect.DeepEqual(leafs, expectation) {
		t.Errorf("expecting leafs %v, got %v", expectation, leafs)
	}
}
