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
			X: int(x * l.Scale),
			Y: int(y * l.Scale),
			W: g.Nodes[i].W,
			H: g.Nodes[i].H,
		}
	}

	// can not recompute edge layout as some paths are complex and not direct
	for e := range g.Edges {
		for p, xy := range g.Edges[e].Path {
			x := float64(xy[0])
			y := float64(xy[1])
			g.Edges[e].Path[p] = [2]int{int(x * l.Scale), int(y * l.Scale)}
		}

		// if edge was not previously set adding at least two nodes for start and end
		if len(g.Edges[e].Path) == 0 {
			g.Edges[e] = Edge{Path: make([][2]int, 2)}
		}

		// end and start should use center coordinates of nodes
		// note, this overrites ports for edges
		g.Edges[e].Path[0] = g.Nodes[e[0]].CenterXY()
		g.Edges[e].Path[len(g.Edges[e].Path)-1] = g.Nodes[e[1]].CenterXY()
	}
}
