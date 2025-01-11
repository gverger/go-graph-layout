package layout

import "fmt"

// Expects that graph g does not have cycles.
// This step creates fake nodes and splits long edges into segments.
func NewLayeredGraph(g Graph) LayeredGraph {
	positions := assignLevels(g)
	edges := makeEdges(g, positions)
	return LayeredGraph{
		NodePosition: positions,
		Segments:     makeSegments(edges),
		Dummy:        makeDummy(edges),
		Edges:        edges,
	}
}

func maxNodeID(g Graph) uint64 {
	var maxNodeID uint64
	for e := range g.Edges {
		if e[0] > maxNodeID {
			maxNodeID = e[0]
		}
		if e[1] > maxNodeID {
			maxNodeID = e[1]
		}
	}
	return maxNodeID
}

func assignLevels(g Graph) map[uint64]LayerPosition {
	nodeYX := make(map[uint64]LayerPosition, len(g.Nodes))
	neighbors := make(map[uint64][]uint64)
	for e := range g.Edges {
		neighbors[e[0]] = append(neighbors[e[0]], e[1])
	}
	for _, root := range g.Roots() {
		nodeYX[root] = LayerPosition{}
		for que := []uint64{root}; len(que) > 0; {
			// pop
			p := que[0]
			if len(que) > 1 {
				que = que[1:]
			} else {
				que = nil
			}

			// set max depth for each child
			for _, child := range neighbors[p] {
				if l := nodeYX[p].Layer + 1; l > nodeYX[child].Layer {
					nodeYX[child] = LayerPosition{Layer: l, Order: 0}
				}
				que = append(que, child)
			}
		}
	}
	return nodeYX
}

// for each long edge breaks it down to multiple segments, for short edge just adds it
func makeSegments(edges map[[2]uint64][]uint64) map[[2]uint64]bool {
	segments := map[[2]uint64]bool{}
	for e, nodes := range edges {
		switch {
		case len(nodes) == 2:
			segments[e] = true
		case len(nodes) > 2:
			for i := range nodes {
				if i == 0 {
					continue
				}
				segments[[2]uint64{nodes[i-1], nodes[i]}] = true
			}
		default:
			panic(fmt.Errorf("edge(%v) has only one node(%v) but at least 2 expected", e, nodes))
		}
	}
	return segments
}

// extracts all fake nodes for edges that are long into separate map
func makeDummy(edges map[[2]uint64][]uint64) map[uint64]bool {
	dummy := map[uint64]bool{}
	for _, nodes := range edges {
		if len(nodes) > 2 {
			for i, n := range nodes {
				if i == 0 || i == (len(nodes)-1) {
					continue
				}
				dummy[n] = true
			}
		}
	}
	return dummy
}

// makeEdges split long edges into segments and add fake nodes
// adds new fake nodes to nodeYX
func makeEdges(g Graph, nodeYX map[uint64]LayerPosition) map[[2]uint64][]uint64 {
	edges := make(map[[2]uint64][]uint64, len(g.Edges))

	nextFakeNodeID := maxNodeID(g) + 1
	for e := range g.Edges {
		fromLayer := nodeYX[e[0]].Layer
		toLayer := nodeYX[e[1]].Layer

		newEdge := []uint64{}
		newEdge = append(newEdge, e[0])

		if (toLayer - fromLayer) > 1 {
			for layer := fromLayer + 1; layer < toLayer; layer++ {
				nodeYX[nextFakeNodeID] = LayerPosition{Layer: layer, Order: 0}
				newEdge = append(newEdge, nextFakeNodeID)
				nextFakeNodeID++
			}
		}

		newEdge = append(newEdge, e[1])

		edges[e] = newEdge
	}

	return edges
}
