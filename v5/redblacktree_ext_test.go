package dohProxy

import (
	"fmt"
	"github.com/emirpasic/gods/trees/redblacktree"
	"testing"
)

var tree RedBlackTreeExtended

func printTree(tree *RedBlackTreeExtended) {
	max, _ := tree.GetMax()
	min, _ := tree.GetMin()
	fmt.Printf("Value for max key: %v\n", max)
	fmt.Printf("Value for min key: %v\n", min)
	fmt.Printf("%v\n", tree)
}

func TestRedBlackTreeExtended(t *testing.T) {
	// RedBlackTreeExtendedExample main method on how to use the custom red-black tree above
	tree = RedBlackTreeExtended{redblacktree.NewWithIntComparator()}

	tree.Put(1, "a") // 1->x (in order)
	tree.Put(2, "b") // 1->x, 2->b (in order)
	tree.Put(3, "c") // 1->x, 2->b, 3->c (in order)
	tree.Put(4, "d") // 1->x, 2->b, 3->c, 4->d (in order)
	tree.Put(5, "e") // 1->x, 2->b, 3->c, 4->d, 5->e (in order)

	max, _ := tree.GetMax()
	min, _ := tree.GetMin()
	t.Logf("Value for max key: %v", max)
	t.Logf("Value for min key: %v", min)
	t.Logf("%v", tree)
	// Value for max key: e
	// Value for min key: a
	// RedBlackTree
	// │       ┌── 5
	// │   ┌── 4
	// │   │   └── 3
	// └── 2
	//     └── 1

	tree.RemoveMin() // 2->b, 3->c, 4->d, 5->e (in order)
	tree.RemoveMax() // 2->b, 3->c, 4->d (in order)
	tree.RemoveMin() // 3->c, 4->d (in order)
	tree.RemoveMax() // 3->c (in order)

	max, _ = tree.GetMax()
	min, _ = tree.GetMin()
	printTree(&tree)
	// Value for max key: c
	// Value for min key: c
	// RedBlackTree
	// └── 3
}
