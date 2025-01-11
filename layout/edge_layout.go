package layout

// DirectEdge is straight line from center of one node to another.
func DirectEdge(from, to Node) Edge {
	return Edge{
		Path: [][2]int{
			{
				from.X + (from.W / 2),
				from.Y + (from.H / 2),
			},
			{
				to.X + (to.W / 2),
				to.Y + (to.H / 2),
			},
		},
	}
}

// DirectEdgesLayout are straight single line edges.
type DirectEdgesLayout struct{}

func (l DirectEdgesLayout) UpdateGraphLayout(g Graph) {
	for e := range g.Edges {
		g.Edges[e] = DirectEdge(g.Nodes[e[0]], g.Nodes[e[1]])
	}
}
