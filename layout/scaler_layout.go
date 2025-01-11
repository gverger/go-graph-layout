package layout

// ScalerLayout will scale existing layout by constant factor.
type ScalerLayout struct {
	Scale float64
}

func (l *ScalerLayout) UpdateGraphLayout(g Graph) {
	for i := range g.Nodes {
		x := float64(g.Nodes[i].X)
		y := float64(g.Nodes[i].Y)

		g.Nodes[i] = Node{
			Position: Position{
				X: int(x * l.Scale),
				Y: int(y * l.Scale),
			},
			W: g.Nodes[i].W,
			H: g.Nodes[i].H,
		}
	}

	// can not recompute edge layout as some paths are complex and not direct
	for e := range g.Edges {
		for p, pos := range g.Edges[e].Path {
			g.Edges[e].Path[p] = Position{
				X: int(float64(pos.X) * l.Scale),
				Y: int(float64(pos.Y) * l.Scale),
			}
		}

		// if edge was not previously set adding at least two nodes for start and end
		if len(g.Edges[e].Path) == 0 {
			g.Edges[e] = Edge{Path: make([]Position, 2)}
		}

		// end and start should use center coordinates of nodes
		// note, this overrites ports for edges
		g.Edges[e].Path[0] = g.Nodes[e[0]].CenterXY()
		g.Edges[e].Path[len(g.Edges[e].Path)-1] = g.Nodes[e[1]].CenterXY()
	}
}
