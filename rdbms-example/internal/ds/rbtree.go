// rbtree.go — Red-Black Tree used for in-memory sorted result sets (ORDER BY).
//
// A Red-Black Tree is a self-balancing BST that guarantees O(log n) insert,
// delete, and search. Its balance invariants:
//  1. Every node is RED or BLACK.
//  2. The root is BLACK.
//  3. Every nil leaf is BLACK.
//  4. A RED node's children are both BLACK.
//  5. All paths from any node to its nil leaves have the same black-height.
//
// Implementation follows Cormen et al. "Introduction to Algorithms" (CLRS).
package ds

type rbColor bool

const (
	rbRed   rbColor = true
	rbBlack rbColor = false
)

// rbNode is a node in the Red-Black Tree.
type rbNode struct {
	key         int64
	vals        []any   // multiple rows can share the same ORDER-BY key
	left, right *rbNode
	parent      *rbNode
	color       rbColor
}

// RBTree is a Red-Black Tree keyed on int64 with any payload.
// Duplicate keys accumulate values in a slice.
type RBTree struct {
	root *rbNode
	nil_ *rbNode // sentinel nil node (always BLACK)
	size int
}

// NewRBTree returns an empty Red-Black Tree.
func NewRBTree() *RBTree {
	sentinel := &rbNode{color: rbBlack}
	return &RBTree{root: sentinel, nil_: sentinel}
}

// Size returns the number of distinct keys.
func (t *RBTree) Size() int { return t.size }

// Insert adds val under key. Duplicate keys accumulate values.
func (t *RBTree) Insert(key int64, val any) {
	// Find or create node.
	y := t.nil_
	x := t.root
	for x != t.nil_ {
		y = x
		switch {
		case key < x.key:
			x = x.left
		case key > x.key:
			x = x.right
		default:
			// Duplicate key: append value.
			x.vals = append(x.vals, val)
			return
		}
	}

	z := &rbNode{
		key:    key,
		vals:   []any{val},
		color:  rbRed,
		left:   t.nil_,
		right:  t.nil_,
		parent: y,
	}
	if y == t.nil_ {
		t.root = z
	} else if key < y.key {
		y.left = z
	} else {
		y.right = z
	}
	t.size++
	t.insertFixup(z)
}

func (t *RBTree) insertFixup(z *rbNode) {
	for z.parent.color == rbRed {
		if z.parent == z.parent.parent.left {
			y := z.parent.parent.right // uncle
			if y.color == rbRed {      // Case 1: recolor
				z.parent.color = rbBlack
				y.color = rbBlack
				z.parent.parent.color = rbRed
				z = z.parent.parent
			} else {
				if z == z.parent.right { // Case 2: inner → rotate
					z = z.parent
					t.leftRotate(z)
				}
				// Case 3: outer → rotate grandparent
				z.parent.color = rbBlack
				z.parent.parent.color = rbRed
				t.rightRotate(z.parent.parent)
			}
		} else { // mirror
			y := z.parent.parent.left
			if y.color == rbRed {
				z.parent.color = rbBlack
				y.color = rbBlack
				z.parent.parent.color = rbRed
				z = z.parent.parent
			} else {
				if z == z.parent.left {
					z = z.parent
					t.rightRotate(z)
				}
				z.parent.color = rbBlack
				z.parent.parent.color = rbRed
				t.leftRotate(z.parent.parent)
			}
		}
	}
	t.root.color = rbBlack
}

func (t *RBTree) leftRotate(x *rbNode) {
	y := x.right
	x.right = y.left
	if y.left != t.nil_ {
		y.left.parent = x
	}
	y.parent = x.parent
	if x.parent == t.nil_ {
		t.root = y
	} else if x == x.parent.left {
		x.parent.left = y
	} else {
		x.parent.right = y
	}
	y.left = x
	x.parent = y
}

func (t *RBTree) rightRotate(x *rbNode) {
	y := x.left
	x.left = y.right
	if y.right != t.nil_ {
		y.right.parent = x
	}
	y.parent = x.parent
	if x.parent == t.nil_ {
		t.root = y
	} else if x == x.parent.right {
		x.parent.right = y
	} else {
		x.parent.left = y
	}
	y.right = x
	x.parent = y
}

// Search returns the first value stored at key, or (nil, false).
func (t *RBTree) Search(key int64) (any, bool) {
	n := t.find(key)
	if n == nil {
		return nil, false
	}
	return n.vals[0], true
}

func (t *RBTree) find(key int64) *rbNode {
	x := t.root
	for x != t.nil_ {
		switch {
		case key == x.key:
			return x
		case key < x.key:
			x = x.left
		default:
			x = x.right
		}
	}
	return nil
}

// InOrder returns all values in ascending key order.
// Rows with equal keys are returned in insertion order.
func (t *RBTree) InOrder() []any {
	var res []any
	t.inorder(t.root, &res)
	return res
}

func (t *RBTree) inorder(n *rbNode, res *[]any) {
	if n == t.nil_ {
		return
	}
	t.inorder(n.left, res)
	*res = append(*res, n.vals...)
	t.inorder(n.right, res)
}

// InOrderDesc returns all values in descending key order.
func (t *RBTree) InOrderDesc() []any {
	var res []any
	t.inorderDesc(t.root, &res)
	return res
}

func (t *RBTree) inorderDesc(n *rbNode, res *[]any) {
	if n == t.nil_ {
		return
	}
	t.inorderDesc(n.right, res)
	*res = append(*res, n.vals...)
	t.inorderDesc(n.left, res)
}

// Delete removes the node with the given key.
func (t *RBTree) Delete(key int64) bool {
	z := t.find(key)
	if z == nil {
		return false
	}
	t.rbDelete(z)
	t.size--
	return true
}

func (t *RBTree) rbDelete(z *rbNode) {
	y := z
	yOrigColor := y.color
	var x *rbNode

	if z.left == t.nil_ {
		x = z.right
		t.transplant(z, z.right)
	} else if z.right == t.nil_ {
		x = z.left
		t.transplant(z, z.left)
	} else {
		y = t.minimum(z.right)
		yOrigColor = y.color
		x = y.right
		if y.parent == z {
			x.parent = y
		} else {
			t.transplant(y, y.right)
			y.right = z.right
			y.right.parent = y
		}
		t.transplant(z, y)
		y.left = z.left
		y.left.parent = y
		y.color = z.color
	}
	if yOrigColor == rbBlack {
		t.deleteFixup(x)
	}
}

func (t *RBTree) transplant(u, v *rbNode) {
	if u.parent == t.nil_ {
		t.root = v
	} else if u == u.parent.left {
		u.parent.left = v
	} else {
		u.parent.right = v
	}
	v.parent = u.parent
}

func (t *RBTree) minimum(x *rbNode) *rbNode {
	for x.left != t.nil_ {
		x = x.left
	}
	return x
}

func (t *RBTree) deleteFixup(x *rbNode) {
	for x != t.root && x.color == rbBlack {
		if x == x.parent.left {
			w := x.parent.right
			if w.color == rbRed {
				w.color = rbBlack
				x.parent.color = rbRed
				t.leftRotate(x.parent)
				w = x.parent.right
			}
			if w.left.color == rbBlack && w.right.color == rbBlack {
				w.color = rbRed
				x = x.parent
			} else {
				if w.right.color == rbBlack {
					w.left.color = rbBlack
					w.color = rbRed
					t.rightRotate(w)
					w = x.parent.right
				}
				w.color = x.parent.color
				x.parent.color = rbBlack
				w.right.color = rbBlack
				t.leftRotate(x.parent)
				x = t.root
			}
		} else {
			w := x.parent.left
			if w.color == rbRed {
				w.color = rbBlack
				x.parent.color = rbRed
				t.rightRotate(x.parent)
				w = x.parent.left
			}
			if w.right.color == rbBlack && w.left.color == rbBlack {
				w.color = rbRed
				x = x.parent
			} else {
				if w.left.color == rbBlack {
					w.right.color = rbBlack
					w.color = rbRed
					t.leftRotate(w)
					w = x.parent.left
				}
				w.color = x.parent.color
				x.parent.color = rbBlack
				w.left.color = rbBlack
				t.rightRotate(x.parent)
				x = t.root
			}
		}
	}
	x.color = rbBlack
}
