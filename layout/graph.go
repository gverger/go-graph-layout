package layout

type NodeID = uint64

type Position struct {
	X int
	Y int
}

// Graph tells how to position nodes and paths for edges
type Graph struct {
	Edges map[[2]NodeID]Edge
	Nodes map[NodeID]Node
}

// Node is how to position node and its dimensions
type Node struct {
	Position
	W int
	H int
}

func (n Node) CenterXY() Position {
	x := n.X + n.W/2
	y := n.Y + n.H/2
	return Position{x, y}
}

// Edge is path of points that edge goes through
type Edge struct {
	Path []Position // [start: {x,y}, ... finish: {x,y}]
}

func (g Graph) Copy() Graph {
	ng := Graph{
		Nodes: make(map[NodeID]Node, len(g.Nodes)),
		Edges: make(map[[2]NodeID]Edge, len(g.Edges)),
	}
	for id, n := range g.Nodes {
		ng.Nodes[id] = n
	}
	for id, e := range g.Edges {
		ng.Edges[id] = Edge{Path: make([]Position, len(e.Path))}
		copy(ng.Edges[id].Path, e.Path)
	}
	return ng
}

func (g Graph) Roots() []NodeID {
	hasParent := make(map[NodeID]bool, len(g.Nodes))
	for e := range g.Edges {
		hasParent[e[1]] = true
	}

	var roots []NodeID
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
