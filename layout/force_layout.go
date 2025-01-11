package layout

import "math"

// Force computes forces for Nodes.
type Force interface {
	UpdateForce(g Graph, f map[uint64][2]float64)
}

// ForceGraphLayout will simulate node movement due to forces.
type ForceGraphLayout struct {
	Delta    float64 // how much move each step
	MaxSteps int     // limit of iterations
	Epsilon  float64 // minimal force
	Forces   []Force
}

func (l ForceGraphLayout) UpdateGraphLayout(g Graph) {
	for step := 0; step < l.MaxSteps; step++ {
		f := make(map[uint64][2]float64, len(g.Nodes))

		// accumulate all forces
		for i := range l.Forces {
			l.Forces[i].UpdateForce(g, f)
		}

		// delete tiny forces
		for i := range g.Nodes {
			if math.Hypot(f[i][0], f[i][1]) < l.Epsilon {
				delete(f, i)
			}
		}

		// early stop if no forces
		if len(f) == 0 {
			break
		}

		// move by delta
		for i := range g.Nodes {
			x := g.Nodes[i].X + int((f[i][0] * l.Delta))
			y := g.Nodes[i].Y + int((f[i][1] * l.Delta))
			g.Nodes[i] = Node{
				X: x,
				Y: y,
				W:  g.Nodes[i].W,
				H:  g.Nodes[i].H,
			}
		}
	}
}

// SpringForce is linear by distance.
type SpringForce struct {
	K         float64 // has to be positive
	L         float64 // distance at rest
	EdgesOnly bool    // true = only edges, false = all nodes
}

func (l SpringForce) UpdateForce(g Graph, f map[uint64][2]float64) {
	for i := range g.Nodes {
		var js []uint64

		if l.EdgesOnly {
			for e := range g.Edges {
				if e[0] == i {
					js = append(js, e[1])
				}
			}
		} else {
			for j := range g.Nodes {
				if i != j {
					js = append(js, j)
				}
			}
		}

		xi := float64(g.Nodes[i].X)
		yi := float64(g.Nodes[i].Y)

		for _, j := range js {
			xj := float64(g.Nodes[j].X)
			yj := float64(g.Nodes[j].Y)

			d := math.Hypot(xi-xj, yi-yj)

			if d > 1 {
				// if stretch, then attraction
				// if shrink, then repulsion
				af := (d - l.L) * l.K
				f[i] = [2]float64{
					f[i][0] + (af * (xj - xi) / d),
					f[i][1] + (af * (yj - yi) / d),
				}
			}
		}
	}
}

// GravityForce is gravity-like repulsive (or attractive) force.
type GravityForce struct {
	K         float64 // positive K for attraction
	EdgesOnly bool    // true = only edges, false = all nodes
}

func (l GravityForce) UpdateForce(g Graph, f map[uint64][2]float64) {
	for i := range g.Nodes {
		var js []uint64
		if l.EdgesOnly {
			for e := range g.Edges {
				if e[0] == i {
					js = append(js, e[1])
				}
			}
		} else {
			for j := range g.Nodes {
				if i != j {
					js = append(js, j)
				}
			}
		}

		xi := float64(g.Nodes[i].X)
		yi := float64(g.Nodes[i].Y)

		for _, j := range js {
			xj := float64(g.Nodes[j].X)
			yj := float64(g.Nodes[j].Y)

			d := math.Hypot(xi-xj, yi-yj)

			if d > 1 {
				af := l.K / d
				f[i] = [2]float64{
					f[i][0] + (af * (xj - xi) / d),
					f[i][1] + (af * (yj - yi) / d),
				}
			}
		}
	}
}
