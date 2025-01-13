package layout

import (
	"fmt"
	"sort"
	"strings"
)

type LayerPosition struct {
	Layer int // Layer index: is the root
	Order int // Order in the layer
}

func (p LayerPosition) IsLeftOf(other LayerPosition) bool {
	if p.Layer != other.Layer {
		panic(fmt.Sprintf("positions not on same layer: %+v < %+v", p, other))
	}

	return p.Order < other.Order
}

// LayeredGraph is graph with dummy nodes such that there is no long edges.
// Short edge is between nodes in Layers next to each other.
// Long edge is between nodes in 1+ Layers between each other.
// Segment is either a short edge or a long edge.
// Top layer has lowest layer number.
type LayeredGraph struct {
	Segments     map[[2]uint64]bool       // segment is an edge in layered graph, can be real edge or piece of fake edge
	Dummy        map[uint64]bool          // fake nodes
	NodePosition map[uint64]LayerPosition // node -> {layer, ordering in layer}
	Edges        map[[2]uint64][]uint64   // real long/short edge -> {real, fake, fake, fake, real} nodes
}

func (g LayeredGraph) Layers() [][]uint64 {
	maxLayer := 0
	for _, position := range g.NodePosition {
		if position.Layer > maxLayer {
			maxLayer = position.Layer
		}
	}

	layers := make([][]uint64, maxLayer+1)
	for node, position := range g.NodePosition {
		layers[position.Layer] = append(layers[position.Layer], node)
	}

	for layerIdx := 0; layerIdx < len(layers); layerIdx++ {
		sort.Slice(layers[layerIdx], func(i, j int) bool {
			return g.NodePosition[layers[layerIdx][i]].IsLeftOf(g.NodePosition[layers[layerIdx][j]])
		})
	}

	return layers
}

func (g LayeredGraph) Validate() error {
	for e := range g.Segments {
		from := g.NodePosition[e[0]].Layer
		to := g.NodePosition[e[1]].Layer
		if from >= to {
			return fmt.Errorf("edge(%v) is wrong direction, got from level(%d) to level(%d)", e, from, to)
		}
	}
	return nil
}

func (g LayeredGraph) String() string {
	out := ""

	out += fmt.Sprintf("fake nodes: %+v\n", g.Dummy)

	segments := []string{}
	for e := range g.Segments {
		segments = append(segments, fmt.Sprintf("%d->%d", e[0], e[1]))
	}
	sort.Strings(segments)
	out += fmt.Sprintf("segments: %s\n", strings.Join(segments, " "))

	layers := g.Layers()
	for l, nodes := range layers {
		vs := ""
		for _, node := range nodes {
			vs += fmt.Sprintf(" %d", node)
		}
		out += fmt.Sprintf("%d: %s\n", l, vs)
	}
	return out
}

// IsInnerSegment tells when edge is between two Dummy nodes.
func (g LayeredGraph) IsInnerSegment(segment [2]uint64) bool {
	return g.Dummy[segment[0]] && g.Dummy[segment[1]]
}

// UpperNeighbors are nodes in upper layer that are connected to given node.
func (g LayeredGraph) UpperNeighbors(node uint64) []uint64 {
	var nodes []uint64
	for e := range g.Segments {
		if e[1] == node {
			if g.NodePosition[e[0]].Layer == (g.NodePosition[e[1]].Layer - 1) {
				nodes = append(nodes, e[0])
			}
		}
	}
	return nodes
}

// LowerNeighbors are nodes in lower layer that are connected to given node.
func (g LayeredGraph) LowerNeighbors(node uint64) []uint64 {
	var nodes []uint64
	for e := range g.Segments {
		if e[0] == node {
			if g.NodePosition[e[0]].Layer == (g.NodePosition[e[1]].Layer - 1) {
				nodes = append(nodes, e[0])
			}
		}
	}
	return nodes
}

// newLayersFrom makes new layers with content identical to source.
func newLayersFrom(src [][]uint64) (dst [][]uint64) {
	dst = make([][]uint64, len(src))
	for i, l := range src {
		dst[i] = make([]uint64, len(l))
		copy(dst[i], l)
	}
	return dst
}

// copyLayers copies from src to destination
func copyLayers(dst, src [][]uint64) {
	for i := range src {
		copy(dst[i], src[i])
	}
}
