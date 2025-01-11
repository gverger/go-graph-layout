package layout

// Graph tells how to position nodes and paths for edges
type Graph struct {
	Edges map[[2]uint64]Edge
	Nodes map[uint64]Node
}

// Node is how to position node and its dimensions
type Node struct {
	X  int
	Y  int
	W  int
	H  int
}

func (n Node) CenterXY() [2]int {
	x := n.X + n.W/2
	y := n.Y + n.H/2
	return [2]int{x, y}
}

// Edge is path of points that edge goes through
type Edge struct {
	Path [][2]int // [start: {x,y}, ... finish: {x,y}]
}

func (g Graph) Copy() Graph {
	ng := Graph{
		Nodes: make(map[uint64]Node, len(g.Nodes)),
		Edges: make(map[[2]uint64]Edge, len(g.Edges)),
	}
	for id, n := range g.Nodes {
		ng.Nodes[id] = n
	}
	for id, e := range g.Edges {
		ng.Edges[id] = Edge{Path: make([][2]int, len(e.Path))}
		copy(ng.Edges[id].Path, e.Path)
	}
	return ng
}

func (g Graph) Roots() []uint64 {
	hasParent := make(map[uint64]bool, len(g.Nodes))
	for e := range g.Edges {
		hasParent[e[1]] = true
	}

	var roots []uint64
	for n := range g.Nodes {
		if !hasParent[n] {
			roots = append(roots, n)
		}
	}
	return roots
}

func (g Graph) TotalNodesWidth() int {
	w := 0
	for _, node := range g.Nodes {
		w += node.W
	}
	return w
}

func (g Graph) TotalNodesHeight() int {
	h := 0
	for _, node := range g.Nodes {
		h += node.H
	}
	return h
}

// BoundingBox coordinates that should fit whole graph.
// Does not consider edges.
func (g Graph) BoundingBox() (minx, miny, maxx, maxy int) {
	for _, node := range g.Nodes {
		nx := node.X
		ny := node.Y

		if nx < minx {
			minx = nx
		}
		if x := nx + node.W; x > maxx {
			maxx = x
		}
		if ny < miny {
			miny = ny
		}
		if y := ny + node.H; y > maxy {
			maxy = y
		}
	}
	return minx, miny, maxx, maxy
}
