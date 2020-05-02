package main

import (
	rbt "github.com/emirpasic/gods/trees/redblacktree"
)

// RedBlackTreeExtended to demonstrate how to extend a RedBlackTree to include new functions
type RedBlackTreeExtended struct {
	*rbt.Tree
}

// GetMin gets the min value and flag if found
func (tree *RedBlackTreeExtended) GetMin() (value interface{}, found bool) {
	node, found := tree.getMinFromNode(tree.Root)
	if node != nil {
		return node.Value, found
	}
	return nil, false
}

// GetMax gets the max value and flag if found
func (tree *RedBlackTreeExtended) GetMax() (value interface{}, found bool) {
	node, found := tree.getMaxFromNode(tree.Root)
	if node != nil {
		return node.Value, found
	}
	return nil, false
}

// RemoveMin removes the min value and flag if found
func (tree *RedBlackTreeExtended) RemoveMin() (value interface{}, deleted bool) {
	node, found := tree.getMinFromNode(tree.Root)
	if found {
		tree.Remove(node.Key)
		return node.Value, found
	}
	return nil, false
}

// RemoveMax removes the max value and flag if found
func (tree *RedBlackTreeExtended) RemoveMax() (value interface{}, deleted bool) {
	node, found := tree.getMaxFromNode(tree.Root)
	if found {
		tree.Remove(node.Key)
		return node.Value, found
	}
	return nil, false
}

func (tree *RedBlackTreeExtended) getMinFromNode(node *rbt.Node) (foundNode *rbt.Node, found bool) {
	if node == nil {
		return nil, false
	}
	if node.Left == nil {
		return node, true
	}
	return tree.getMinFromNode(node.Left)
}

func (tree *RedBlackTreeExtended) getMaxFromNode(node *rbt.Node) (foundNode *rbt.Node, found bool) {
	if node == nil {
		return nil, false
	}
	if node.Right == nil {
		return node, true
	}
	return tree.getMaxFromNode(node.Right)
}
