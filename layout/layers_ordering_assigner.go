package layout

import (
	"log"
	"math/rand"
	"sort"
)

type LayerOrderingInitializer interface {
	Init(segments map[[2]uint64]bool, layers [][]uint64)
}

type LayerOrderingOptimizer interface {
	Optimize(segments map[[2]uint64]bool, layers [][]uint64, idx int, downUp bool)
}

type CompositeLayerOrderingOptimizer struct {
	Optimizers []LayerOrderingOptimizer
}

func (o CompositeLayerOrderingOptimizer) Optimize(segments map[[2]uint64]bool, layers [][]uint64, idx int, downUp bool) {
	for _, q := range o.Optimizers {
		q.Optimize(segments, layers, idx, downUp)
	}
}

// WarfieldOrderingOptimizer is heuristic based strategy for ordering optimization.
// Goes up and down number of iterations across all layers.
// Considers upper and lower fixed and permutes ordering in layer.
// Used in Graphviz/dot.
type WarfieldOrderingOptimizer struct {
	Epochs                   int
	LayerOrderingInitializer LayerOrderingInitializer
	LayerOrderingOptimizer   LayerOrderingOptimizer
}

func (o WarfieldOrderingOptimizer) Optimize(g Graph, lg LayeredGraph) {
	// layers is temporary layers
	layers := lg.Layers()
	o.LayerOrderingInitializer.Init(lg.Segments, layers)

	bestN := -1
	bestLayers := newLayersFrom(layers)

	for t := 0; t < o.Epochs; t++ {
		downUp := (t % 2) == 0
		for i := range layers {
			j := i
			if downUp {
				j = len(layers) - 1 - i
			}
			o.LayerOrderingOptimizer.Optimize(lg.Segments, layers, j, downUp)
		}

		N := numCrossings(lg.Segments, layers)
		if bestN < 0 || N < bestN {
			bestN = N
			copyLayers(bestLayers, layers)
		}
		log.Printf("warfield ordering optimizer:\t epoch(%d)\t best(%d)\t current(%d)\n", t, bestN, N)
		if N == 0 {
			break
		}
	}

	// store to graph
	for y, layer := range bestLayers {
		for x, node := range layer {
			lg.NodePosition[node] = LayerPosition{Layer: y, Order: x}
		}
	}
}

// BFSOrderingInitializer will set order in each layer by traversing BFS from roots.
type BFSOrderingInitializer struct{}

func (o BFSOrderingInitializer) Init(segments map[[2]uint64]bool, layers [][]uint64) {
	// accumulate where node can lead to
	fromNodeToNodes := map[uint64][]uint64{}
	for e := range segments {
		if _, ok := fromNodeToNodes[e[0]]; !ok {
			fromNodeToNodes[e[0]] = []uint64{}
		}
		fromNodeToNodes[e[0]] = append(fromNodeToNodes[e[0]], e[1])
	}

	// get roots
	hasParent := map[uint64]bool{}
	for e := range segments {
		hasParent[e[1]] = true
	}
	var roots []uint64
	for e := range segments {
		if _, ok := hasParent[e[1]]; !ok {
			roots = append(roots, e[1])
		}
	}

	// BFS starting with roots, keeping order when node is visited
	cnt := 0
	que := roots
	ord := map[uint64]int{}
	for len(que) > 0 {
		p := que[0]
		if len(que) > 1 {
			que = que[1:]
		} else {
			que = nil
		}

		if _, ok := ord[p]; ok {
			continue
		}

		ord[p] = cnt
		cnt++

		que = append(que, fromNodeToNodes[p]...)
	}

	for l := range layers {
		sort.Slice(layers[l], func(i, j int) bool { return ord[layers[l][i]] < ord[layers[l][j]] })
	}
}

// RandomLayerOrderingInitializer assigns random ordering in each layer
type RandomLayerOrderingInitializer struct{}

func (o RandomLayerOrderingInitializer) Init(_ map[[2]uint64]bool, layers [][]uint64) {
	for i := range layers {
		l := layers[i]
		rand.Shuffle(len(l), func(a, b int) { l[a], l[b] = l[b], l[a] })
	}
}

// WMedianOrderingOptimizer takes median of upper (or lower) level neighbors for each node in layer.
// Median has property of stable vertical edges which is especially useful for "long" edges (fake nodes).
// Eades and Wormald, 1994
// This is used in dot/Graphviz, Figure 3-2 in Graphviz dot paper TSE93.
type WMedianOrderingOptimizer struct{}

func (o WMedianOrderingOptimizer) Optimize(segments map[[2]uint64]bool, layers [][]uint64, y int, downUp bool) {
	w := map[uint64]float64{}

	for i, node := range layers[y] {
		var xs []int
		if downUp {
			xs = lowerNeighborsX(segments, layers, i, y)
		} else {
			xs = upperNeighborsX(segments, layers, i, y)
		}

		P := make([]float64, len(xs))
		for i, v := range xs {
			P[i] = float64(v)
		}
		w[node] = median(P)
	}

	sort.Slice(layers[y], func(i, j int) bool { return w[layers[y][i]] < w[layers[y][j]] })
}

// time: O(len(layer))
// space: O(len(layer))
func lowerNeighborsX(segments map[[2]uint64]bool, layers [][]uint64, x int, y int) []int {
	if y == (len(layers) - 1) {
		return nil
	}

	t := layers[y][x]

	var nx []int
	for i, n := range layers[y+1] {
		if segments[[2]uint64{t, n}] {
			nx = append(nx, i)
		}
	}

	return nx
}

// time: O(len(layer))
// space: O(len(layer))
func upperNeighborsX(segments map[[2]uint64]bool, layers [][]uint64, x int, y int) []int {
	if y == 0 {
		return nil
	}

	t := layers[y][x]

	var nx []int
	for i, n := range layers[y-1] {
		if segments[[2]uint64{n, t}] {
			nx = append(nx, i)
		}
	}

	return nx
}

func median(P []float64) float64 {
	m := len(P) / 2
	switch {
	case len(P) == 0:
		return -1
	case len(P)%2 == 1:
		return P[m]
	case len(P) == 2:
		return (P[0] + P[1]) / 2
	default:
		left := P[m-1] - P[0]
		right := P[len(P)-1] - P[m]
		return (P[m-1]*right + P[m]*left) / (left + right)
	}
}

// SwitchAdjacentOrderingOptimizer will try swapping two adjacent nodes in a layer will improve crossings.
// This is used in dot/Graphviz, Figure 3-3 in Graphviz dot paper TSE93 and called "transpose".
type SwitchAdjacentOrderingOptimizer struct{}

func (o SwitchAdjacentOrderingOptimizer) Optimize(segments map[[2]uint64]bool, layers [][]uint64, y int, downUp bool) {
	if len(layers[y]) < 2 {
		return
	}

	// does not have bellow
	if downUp && y == (len(layers)-1) {
		return
	}

	// does not have above
	if !downUp && y == 0 {
		return
	}

	for i := 0; i < (len(layers[y]) - 1); i++ {
		j := i + 1

		current := []uint64{layers[y][i], layers[y][j]}
		swapped := []uint64{layers[y][j], layers[y][i]}
		var currCrossings, swappedCrossings int
		if downUp {
			currCrossings = numCrossingsBetweenLayers(segments, current, layers[y+1])
			swappedCrossings = numCrossingsBetweenLayers(segments, swapped, layers[y+1])
		} else {
			currCrossings = numCrossingsBetweenLayers(segments, layers[y-1], current)
			swappedCrossings = numCrossingsBetweenLayers(segments, layers[y-1], swapped)
		}

		if swappedCrossings < currCrossings {
			layers[y][i], layers[y][j] = layers[y][j], layers[y][i]
		}
	}
}

// RandomLayerOrderingOptimizer picks best out of epochs random orderings.
type RandomLayerOrderingOptimizer struct {
	Epochs int
}

func (o RandomLayerOrderingOptimizer) Optimize(segments map[[2]uint64]bool, layers [][]uint64, idx int, downUp bool) {
	bestN := -1
	layer := make([]uint64, len(layers[idx]))
	copy(layer, layers[idx])

	for i := 0; i < o.Epochs; i++ {
		rand.Shuffle(len(layer), func(a, b int) { layer[a], layer[b] = layer[b], layer[a] })

		N := 0
		if idx > 0 {
			N += numCrossingsBetweenLayers(segments, layers[idx-1], layers[idx])
		}
		if (idx + 1) < len(layers) {
			N += numCrossingsBetweenLayers(segments, layers[idx], layers[idx+1])
		}

		if bestN < 0 || N < bestN {
			bestN = N
			copy(layers[idx], layer)
		}
	}
}

// time: O(ntop * nbot * log(ntop))
// memory: O(ntop)
func numCrossingsBetweenLayers(segments map[[2]uint64]bool, ltop, lbottom []uint64) int {
	sum := 0
	bit := NewFenwickTree(len(ltop))
	for i := len(lbottom) - 1; i >= 0; i-- {
		node := lbottom[i]
		for j := len(ltop) - 1; j >= 0; j-- {
			neighbor := ltop[j]
			if segments[[2]uint64{neighbor, node}] {
				bit.Update(j+1, 1)
				sum += bit.Query(j)
			}
		}
	}
	return sum
}

type FenwickTree []int

func NewFenwickTree(nbElements int) FenwickTree {
	return make(FenwickTree, nbElements+1)
}

func (bit FenwickTree) Update(idx int, value int) {
	for ; idx < len(bit); idx += idx & (-idx) {
		bit[idx] += value
	}
}

func (bit FenwickTree) Query(idx int) int {
	sum := 0
	for ; idx > 0; idx -= idx & (-idx) {
		sum += bit[idx]
	}
	return sum
}

// time: O(?)
// memory: O(1)
func numCrossings(segments map[[2]uint64]bool, layers [][]uint64) int {
	count := 0
	for i := range layers {
		if i == 0 {
			continue
		}
		count += numCrossingsBetweenLayers(segments, layers[i-1], layers[i])
	}
	return count
}
