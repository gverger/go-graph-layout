package layout

import (
	"math"
	"sort"
)

// "Fast and Simple Horizontal Coordinate Assignment" by Ulrik Brandes and Boris Kopf, 2002
// Modified with "Erratum" from 2020 by Ulrik Brandes, Julian Walter and Johannes Zink.
// Computes horizontal coordinate in layered graph, given ordering within each layer.
// Produces result such that neighbors are close and long edges cross Layers are straight.
// Works on fully connected graphs.
// Assuming nodes do not have width.
type BrandesKopfLayersNodesHorizontalAssigner struct {
	Delta       int  // distance between nodes, including fake ones
	TopDownOnly bool // true if running the 2 top down strategies only (better for trees)
}

type Neighbors struct {
	Up   map[uint64][]uint64
	Down map[uint64][]uint64
}

func computeOrderedNeighbors(g LayeredGraph) Neighbors {
	n := Neighbors{
		Up:   make(map[uint64][]uint64),
		Down: make(map[uint64][]uint64),
	}

	for e := range g.Segments {
		n.Down[e[0]] = append(n.Down[e[0]], e[1])
		n.Up[e[1]] = append(n.Up[e[1]], e[0])
	}
	for _, d := range n.Down {
		sort.Slice(d, func(i, j int) bool { return g.NodePosition[d[i]].IsLeftOf(g.NodePosition[d[j]]) })
	}

	for _, d := range n.Up {
		sort.Slice(d, func(i, j int) bool { return g.NodePosition[d[i]].IsLeftOf(g.NodePosition[d[j]]) })
	}

	return n
}

type LayoutResult struct {
	x          map[uint64]int
	minX, maxX int
}

func (x LayoutResult) width() int {
	return x.maxX - x.minX
}

func (s BrandesKopfLayersNodesHorizontalAssigner) NodesHorizontalCoordinates(_ Graph, g LayeredGraph) map[uint64]int {
	neighbors := computeOrderedNeighbors(g)
	typeOneSegments := preprocessing(g, neighbors)

	resTL := runAlgo(TopLeft{}, g, typeOneSegments, neighbors, s)
	resTR := runAlgo(TopRight{}, g, typeOneSegments, neighbors, s)
	resBL := resTL
	resBR := resTR
	if !s.TopDownOnly {
		resBL = runAlgo(BottomLeft{}, g, typeOneSegments, neighbors, s)
		resBR = runAlgo(BottomRight{}, g, typeOneSegments, neighbors, s)
	}

	best := resTL
	if resTR.width() < best.width() {
		best = resTR
	}
	if resBL.width() < best.width() {
		best = resBL
	}
	if resBR.width() < best.width() {
		best = resBR
	}

	shiftTL := best.minX - resTL.minX
	shiftTR := best.maxX - resTR.maxX
	shiftBL := best.minX - resBL.minX
	shiftBR := best.maxX - resBR.maxX

	x := make(map[uint64]int, len(g.NodePosition))
	place := make([]int, 4)
	for n := range g.NodePosition {
		place[0] = resTL.x[n] + shiftTL
		place[1] = resTR.x[n] + shiftTR
		place[2] = resBL.x[n] + shiftBL
		place[3] = resBR.x[n] + shiftBR
		sort.Ints(place)
		x[n] = (place[1] + place[2]) / 2
	}

	return x
}

func runAlgo(dir singleDirAlgo, g LayeredGraph, typeOneSegments map[[2]uint64]bool, neighbors Neighbors, s BrandesKopfLayersNodesHorizontalAssigner) LayoutResult {
	root, align := dir.verticalAlignment(g, typeOneSegments, neighbors)
	x := dir.horizontalCompaction(g, root, align, s.Delta)

	res := LayoutResult{
		x:    x,
		minX: math.MaxInt,
		maxX: math.MinInt,
	}
	for _, v := range x {
		if v < res.minX {
			res.minX = v
		}
		if v > res.maxX {
			res.maxX = v
		}
	}

	return res
}

// Alg 1.
// Type 1 conflicts arise when a non-inner segment (normal edge) crosses an inner segment (edge between two fake nodes).
// The algorithm traverses Layers from left to right (index l) while maintaining the upper neighbors,
// v(i)_k0 and v(i)_k1, of the two closest inner Segments.
func preprocessing(g LayeredGraph, n Neighbors) (typeOneSegments map[[2]uint64]bool) {
	typeOneSegments = map[[2]uint64]bool{}

	layers := g.Layers()
	for i := range layers {
		if i == (len(layers) - 1) {
			continue
		}
		nextLayer := layers[i+1]

		k0 := 0
		l := 0

		for l1, v := range nextLayer {
			var upperNeighborInnerSegment uint64
			for _, u := range n.Up[v] {
				if g.IsInnerSegment([2]uint64{u, v}) {
					upperNeighborInnerSegment = u
					break
				}
			}

			if (l1 == (len(nextLayer) - 1)) || upperNeighborInnerSegment != 0 {
				k1 := len(layers[i]) - 1
				if upperNeighborInnerSegment != 0 {
					k1 = g.NodePosition[upperNeighborInnerSegment].Order
				}
				for l <= l1 {
					for k, u := range n.Up[nextLayer[l]] {
						if k < k0 || k > k1 {
							typeOneSegments[[2]uint64{u, v}] = true
						}
					}
					l += 1
				}
				k0 = k1
			}
		}
	}

	return typeOneSegments
}

type singleDirAlgo interface {
	verticalAlignment(g LayeredGraph, typeOneSegments map[[2]uint64]bool, n Neighbors) (root map[uint64]uint64, align map[uint64]uint64)

	horizontalCompaction(g LayeredGraph, root map[uint64]uint64, align map[uint64]uint64, delta int) (x map[uint64]int)
}

type TopLeft struct{}

// Alg 2.
// Obtain a leftmost alignment with upper neighbors.
// A maximal set of vertically aligned vertices is called a block, and we define the root of a block to be its topmost vertex.
// Blocks are stored as cyclicly linked lists, each node has reference to its lower aligned neighbor and lowest refers to topmost.
// Each node has additional reference to root of its block.
func (s TopLeft) verticalAlignment(g LayeredGraph, typeOneSegments map[[2]uint64]bool, n Neighbors) (root map[uint64]uint64, align map[uint64]uint64) {
	root = make(map[uint64]uint64, len(g.NodePosition))
	align = make(map[uint64]uint64, len(g.NodePosition))

	for v := range g.NodePosition {
		root[v] = v
		align[v] = v
	}

	layers := g.Layers()
	for i := range layers {
		r := -1
		for _, v := range layers[i] {
			upNeighbors := n.Up[v]
			if d := len(upNeighbors); d > 0 {
				for m := (d - 1) / 2; m <= (d+1)/2 && m < len(upNeighbors); m++ {
					if align[v] == v {
						u := upNeighbors[m]
						if !typeOneSegments[[2]uint64{u, v}] && r < g.NodePosition[u].Order {
							align[u] = v
							root[v] = root[u]
							align[v] = root[v]
							r = g.NodePosition[u].Order
						}
					}
				}
			}
		}
	}

	return root, align
}

// part of Alg 3.
func (s TopLeft) placeBlock(g LayeredGraph, x map[uint64]int, root map[uint64]uint64, align map[uint64]uint64, sink map[uint64]uint64, shift map[uint64]int, delta int, v uint64, layers [][]uint64) {
	if _, ok := x[v]; !ok {
		x[v] = 0
		flag := true
		w := v
		for ; flag; flag = v != w {
			if g.NodePosition[w].Order > 0 {
				u := root[layers[g.NodePosition[w].Layer][g.NodePosition[w].Order-1]]
				s.placeBlock(g, x, root, align, sink, shift, delta, u, layers)
				if sink[v] == v {
					sink[v] = sink[u]
				}
				if sink[v] != sink[u] {
					if s := x[v] - x[u] - delta; s < shift[sink[u]] {
						shift[sink[u]] = s
					}
				} else {
					if s := x[u] + delta; s > x[v] {
						x[v] = s
					}
				}
			}
			w = align[w]
		}
		for align[w] != v {
			w = align[w]
			x[w] = x[v]
			sink[w] = sink[v]
		}
	}
}

// Alg 3.
// All node of a block are assigned the coordinate of the root.
// Partition each block in to classes.
// Class is defined by reachable sink which has the topmost root
// Within each class, we apply a longest path layering,
// i.e. the relative coordinate of a block with respect to the defining
// sink is recursively determined to be the maximum coordinate of
// the preceding blocks in the same class, plus minimum separation.
// For each class, from top to bottom, we then compute the absolute coordinates
// of its members by placing the class with minimum separation from previously placed classes.
func (s TopLeft) horizontalCompaction(g LayeredGraph, root map[uint64]uint64, align map[uint64]uint64, delta int) (x map[uint64]int) {
	sink := map[uint64]uint64{}
	shift := map[uint64]int{}
	x = map[uint64]int{}

	for v := range g.NodePosition {
		sink[v] = v
		shift[v] = math.MaxInt
	}

	layers := g.Layers()
	// root coordinates relative to sink
	for v := range g.NodePosition {
		if root[v] == v {
			s.placeBlock(g, x, root, align, sink, shift, delta, v, layers)
		}
	}

	// class offsets
	for i := 0; i < len(layers); i++ {
		layer := layers[i]
		vfirst := layer[0]
		if sink[vfirst] == vfirst {
			if shift[sink[vfirst]] == math.MaxInt {
				shift[sink[vfirst]] = 0
			}
			j := i
			k := 0
			for {
				v := layers[j][k]

				for align[v] != root[v] {
					v = align[v]
					j++
					if g.NodePosition[v].Order > 0 {
						u := layers[g.NodePosition[v].Layer][g.NodePosition[v].Order-1]
						shifted := shift[sink[v]] + x[v] - (x[u] + delta)
						if shifted < shift[sink[u]] {
							shift[sink[u]] = shifted
						}
					}
				}
				k = g.NodePosition[v].Order + 1

				if k > len(layers[j])-1 || sink[v] != sink[layers[j][k]] {
					break
				}
			}
		}
	}

	// absolute coordinates
	for v := range g.NodePosition {
		x[v] += shift[sink[v]]
	}

	return x
}

type TopRight struct{}

// Alg 2.
// Obtain a rightmost alignment with upper neighbors.
// A maximal set of vertically aligned vertices is called a block, and we define the root of a block to be its topmost vertex.
// Blocks are stored as cyclicly linked lists, each node has reference to its lower aligned neighbor and lowest refers to topmost.
// Each node has additional reference to root of its block.
func (s TopRight) verticalAlignment(g LayeredGraph, typeOneSegments map[[2]uint64]bool, n Neighbors) (root map[uint64]uint64, align map[uint64]uint64) {
	root = make(map[uint64]uint64, len(g.NodePosition))
	align = make(map[uint64]uint64, len(g.NodePosition))

	for v := range g.NodePosition {
		root[v] = v
		align[v] = v
	}

	layers := g.Layers()
	for i := range layers {
		r := math.MaxInt
		for j := len(layers[i]) - 1; j >= 0; j-- {
			v := layers[i][j]
			upNeighbors := n.Up[v]
			if d := len(upNeighbors); d > 0 {
				first := (d + 1) / 2
				if first >= d {
					first = d - 1
				}
				for m := first; m >= (d-1)/2; m-- {
					if align[v] == v {
						u := upNeighbors[m]
						if !typeOneSegments[[2]uint64{u, v}] && r > g.NodePosition[u].Order {
							align[u] = v
							root[v] = root[u]
							align[v] = root[v]
							r = g.NodePosition[u].Order
						}
					}
				}
			}
		}
	}

	return root, align
}

// part of Alg 3.
func (s TopRight) placeBlock(g LayeredGraph, x map[uint64]int, root map[uint64]uint64, align map[uint64]uint64, sink map[uint64]uint64, shift map[uint64]int, delta int, v uint64, layers [][]uint64) {
	if _, ok := x[v]; !ok {
		x[v] = 0
		flag := true
		w := v
		for ; flag; flag = v != w {
			if g.NodePosition[w].Order < len(layers[g.NodePosition[w].Layer])-1 {
				u := root[layers[g.NodePosition[w].Layer][g.NodePosition[w].Order+1]]
				s.placeBlock(g, x, root, align, sink, shift, delta, u, layers)
				if sink[v] == v {
					sink[v] = sink[u]
				}
				if sink[v] != sink[u] {
					if s := x[v] + x[u] + delta; s > shift[sink[u]] {
						shift[sink[u]] = s
					}
				} else {
					if s := x[u] - delta; s < x[v] {
						x[v] = s
					}
				}
			}
			w = align[w]
		}
		for align[w] != v {
			w = align[w]
			x[w] = x[v]
			sink[w] = sink[v]
		}
	}
}

// Alg 3.
// All node of a block are assigned the coordinate of the root.
// Partition each block in to classes.
// Class is defined by reachable sink which has the topmost root
// Within each class, we apply a longest path layering,
// i.e. the relative coordinate of a block with respect to the defining
// sink is recursively determined to be the maximum coordinate of
// the preceding blocks in the same class, plus minimum separation.
// For each class, from top to bottom, we then compute the absolute coordinates
// of its members by placing the class with minimum separation from previously placed classes.
func (s TopRight) horizontalCompaction(g LayeredGraph, root map[uint64]uint64, align map[uint64]uint64, delta int) (x map[uint64]int) {
	sink := map[uint64]uint64{}
	shift := map[uint64]int{}
	x = map[uint64]int{}

	for v := range g.NodePosition {
		sink[v] = v
		shift[v] = math.MinInt
	}

	layers := g.Layers()
	// root coordinates relative to sink
	for v := range g.NodePosition {
		if root[v] == v {
			s.placeBlock(g, x, root, align, sink, shift, delta, v, layers)
		}
	}

	// class offsets
	for i := 0; i < len(layers); i++ {
		layer := layers[i]
		vfirst := layer[len(layer)-1]
		if sink[vfirst] == vfirst {
			if shift[sink[vfirst]] == math.MinInt {
				shift[sink[vfirst]] = 0
			}
			j := i
			k := len(layers[j]) - 1
			for {
				v := layers[j][k]

				for align[v] != root[v] {
					v = align[v]
					j++
					if g.NodePosition[v].Order < len(layers[j])-1 {
						u := layers[g.NodePosition[v].Layer][g.NodePosition[v].Order+1]
						shifted := shift[sink[v]] + x[v] - (x[u] - delta)
						if shifted > shift[sink[u]] {
							shift[sink[u]] = shifted
						}
					}
				}
				k = g.NodePosition[v].Order - 1

				if k < 0 || sink[v] != sink[layers[j][k]] {
					break
				}
			}
		}
	}

	// absolute coordinates
	for v := range g.NodePosition {
		x[v] += shift[sink[v]]
	}

	return x
}

type BottomLeft struct{}

// Alg 2.
// Obtain a leftmost alignment with lower neighbors.
// A maximal set of vertically aligned vertices is called a block, and we define the root of a block to be its topmost vertex.
// Blocks are stored as cyclicly linked lists, each node has reference to its lower aligned neighbor and lowest refers to topmost.
// Each node has additional reference to root of its block.
func (s BottomLeft) verticalAlignment(g LayeredGraph, typeOneSegments map[[2]uint64]bool, n Neighbors) (root map[uint64]uint64, align map[uint64]uint64) {
	root = make(map[uint64]uint64, len(g.NodePosition))
	align = make(map[uint64]uint64, len(g.NodePosition))

	for v := range g.NodePosition {
		root[v] = v
		align[v] = v
	}

	layers := g.Layers()
	for ri := range layers {
		i := len(layers) - ri - 1
		r := -1
		for _, v := range layers[i] {
			downNeighbors := n.Down[v]
			if d := len(downNeighbors); d > 0 {
				for m := (d - 1) / 2; m <= (d+1)/2 && m < len(downNeighbors); m++ {
					if align[v] == v {
						u := downNeighbors[m]
						if !typeOneSegments[[2]uint64{v, u}] && r < g.NodePosition[u].Order {
							align[u] = v
							root[v] = root[u]
							align[v] = root[v]
							r = g.NodePosition[u].Order
						}
					}
				}
			}
		}
	}

	return root, align
}

// part of Alg 3.
func (s BottomLeft) placeBlock(g LayeredGraph, x map[uint64]int, root map[uint64]uint64, align map[uint64]uint64, sink map[uint64]uint64, shift map[uint64]int, delta int, v uint64, layers [][]uint64) {
	if _, ok := x[v]; !ok {
		x[v] = 0
		flag := true
		w := v
		for ; flag; flag = v != w {
			if g.NodePosition[w].Order > 0 {
				u := root[layers[g.NodePosition[w].Layer][g.NodePosition[w].Order-1]]
				s.placeBlock(g, x, root, align, sink, shift, delta, u, layers)
				if sink[v] == v {
					sink[v] = sink[u]
				}
				if sink[v] != sink[u] {
					if s := x[v] - x[u] - delta; s < shift[sink[u]] {
						shift[sink[u]] = s
					}
				} else {
					if s := x[u] + delta; s > x[v] {
						x[v] = s
					}
				}
			}
			w = align[w]
		}
		for align[w] != v {
			w = align[w]
			x[w] = x[v]
			sink[w] = sink[v]
		}
	}
}

// Alg 3.
// All node of a block are assigned the coordinate of the root.
// Partition each block in to classes.
// Class is defined by reachable sink which has the topmost root
// Within each class, we apply a longest path layering,
// i.e. the relative coordinate of a block with respect to the defining
// sink is recursively determined to be the maximum coordinate of
// the preceding blocks in the same class, plus minimum separation.
// For each class, from top to bottom, we then compute the absolute coordinates
// of its members by placing the class with minimum separation from previously placed classes.
func (s BottomLeft) horizontalCompaction(g LayeredGraph, root map[uint64]uint64, align map[uint64]uint64, delta int) (x map[uint64]int) {
	sink := map[uint64]uint64{}
	shift := map[uint64]int{}
	x = map[uint64]int{}

	for v := range g.NodePosition {
		sink[v] = v
		shift[v] = math.MaxInt
	}

	layers := g.Layers()
	// root coordinates relative to sink
	for v := range g.NodePosition {
		if root[v] == v {
			s.placeBlock(g, x, root, align, sink, shift, delta, v, layers)
		}
	}

	// class offsets
	for i := len(layers) - 1; i >= 0; i-- {
		layer := layers[i]
		vfirst := layer[0]
		if sink[vfirst] == vfirst {
			if shift[sink[vfirst]] == math.MaxInt {
				shift[sink[vfirst]] = 0
			}
			j := i
			k := 0
			for {
				v := layers[j][k]

				for align[v] != root[v] {
					v = align[v]
					j--
					if g.NodePosition[v].Order > 0 {
						u := layers[g.NodePosition[v].Layer][g.NodePosition[v].Order-1]
						shifted := shift[sink[v]] + x[v] - (x[u] + delta)
						if shifted < shift[sink[u]] {
							shift[sink[u]] = shifted
						}
					}
				}
				k = g.NodePosition[v].Order + 1

				if k > len(layers[j])-1 || sink[v] != sink[layers[j][k]] {
					break
				}
			}
		}
	}

	// absolute coordinates
	for v := range g.NodePosition {
		x[v] += shift[sink[v]]
	}

	return x
}

type BottomRight struct{} // RIGHT UP in paper

// Alg 2.
// Obtain a rightmost alignment with lower neighbors.
// A maximal set of vertically aligned vertices is called a block, and we define the root of a block to be its topmost vertex.
// Blocks are stored as cyclicly linked lists, each node has reference to its lower aligned neighbor and lowest refers to topmost.
// Each node has additional reference to root of its block.
func (s BottomRight) verticalAlignment(g LayeredGraph, typeOneSegments map[[2]uint64]bool, n Neighbors) (root map[uint64]uint64, align map[uint64]uint64) {
	root = make(map[uint64]uint64, len(g.NodePosition))
	align = make(map[uint64]uint64, len(g.NodePosition))

	for v := range g.NodePosition {
		root[v] = v
		align[v] = v
	}

	layers := g.Layers()
	for i := len(layers) - 1; i >= 0; i-- {
		r := math.MaxInt
		for j := len(layers[i]) - 1; j >= 0; j-- {
			v := layers[i][j]
			downNeighbors := n.Down[v]
			if d := len(downNeighbors); d > 0 {
				first := (d + 1) / 2
				if first >= d {
					first = d - 1
				}
				for m := first; m >= (d-1)/2; m-- {
					if align[v] == v {
						u := downNeighbors[m]
						if !typeOneSegments[[2]uint64{v, u}] && r > g.NodePosition[u].Order {
							align[u] = v
							root[v] = root[u]
							align[v] = root[v]
							r = g.NodePosition[u].Order
						}
					}
				}
			}
		}
	}

	return root, align
}

// part of Alg 3.
func (s BottomRight) placeBlock(g LayeredGraph, x map[uint64]int, root map[uint64]uint64, align map[uint64]uint64, sink map[uint64]uint64, shift map[uint64]int, delta int, v uint64, layers [][]uint64) {
	if _, ok := x[v]; !ok {
		x[v] = 0
		flag := true
		w := v
		for ; flag; flag = v != w {
			if g.NodePosition[w].Order < len(layers[g.NodePosition[w].Layer])-1 {
				u := root[layers[g.NodePosition[w].Layer][g.NodePosition[w].Order+1]]
				s.placeBlock(g, x, root, align, sink, shift, delta, u, layers)
				if sink[v] == v {
					sink[v] = sink[u]
				}
				if sink[v] != sink[u] {
					if s := x[v] + x[u] + delta; s > shift[sink[u]] {
						shift[sink[u]] = s
					}
				} else {
					if s := x[u] - delta; s < x[v] {
						x[v] = s
					}
				}
			}
			w = align[w]
		}
		for align[w] != v {
			w = align[w]
			x[w] = x[v]
			sink[w] = sink[v]
		}
	}
}

// Alg 3.
// All node of a block are assigned the coordinate of the root.
// Partition each block in to classes.
// Class is defined by reachable sink which has the topmost root
// Within each class, we apply a longest path layering,
// i.e. the relative coordinate of a block with respect to the defining
// sink is recursively determined to be the maximum coordinate of
// the preceding blocks in the same class, plus minimum separation.
// For each class, from top to bottom, we then compute the absolute coordinates
// of its members by placing the class with minimum separation from previously placed classes.
func (s BottomRight) horizontalCompaction(g LayeredGraph, root map[uint64]uint64, align map[uint64]uint64, delta int) (x map[uint64]int) {
	sink := map[uint64]uint64{}
	shift := map[uint64]int{}
	x = map[uint64]int{}

	for v := range g.NodePosition {
		sink[v] = v
		shift[v] = math.MinInt
	}

	layers := g.Layers()
	// root coordinates relative to sink
	for v := range g.NodePosition {
		if root[v] == v {
			s.placeBlock(g, x, root, align, sink, shift, delta, v, layers)
		}
	}

	// class offsets
	for i := len(layers) - 1; i >= 0; i-- {
		layer := layers[i]
		vfirst := layer[len(layer)-1]
		if sink[vfirst] == vfirst {
			if shift[sink[vfirst]] == math.MinInt {
				shift[sink[vfirst]] = 0
			}
			j := i
			k := len(layers[j]) - 1
			for {
				v := layers[j][k]

				for align[v] != root[v] {
					v = align[v]
					j--
					if g.NodePosition[v].Order < len(layers[j])-1 {
						u := layers[g.NodePosition[v].Layer][g.NodePosition[v].Order+1]
						shifted := shift[sink[v]] + x[v] - (x[u] - delta)
						if shifted > shift[sink[u]] {
							shift[sink[u]] = shifted
						}
					}
				}
				k = g.NodePosition[v].Order - 1

				if k < 0 || sink[v] != sink[layers[j][k]] {
					break
				}
			}
		}
	}

	// absolute coordinates
	for v := range g.NodePosition {
		x[v] += shift[sink[v]]
	}

	return x
}
