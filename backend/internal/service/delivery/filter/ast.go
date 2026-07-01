package filter

type node interface {
	eval(lowerTitle string) bool
}

type orNode struct{ left, right node }

func (n orNode) eval(t string) bool { return n.left.eval(t) || n.right.eval(t) }

type andNode struct{ left, right node }

func (n andNode) eval(t string) bool { return n.left.eval(t) && n.right.eval(t) }

type notNode struct{ operand node }

func (n notNode) eval(t string) bool { return !n.operand.eval(t) }

type keywordNode struct{ lower string }

func (n keywordNode) eval(t string) bool { return containsSubstr(t, n.lower) }

type alwaysTrueNode struct{}

func (alwaysTrueNode) eval(string) bool { return true }
