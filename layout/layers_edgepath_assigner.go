package layout

import (
	"fmt"
)

// StraightEdgePathAssigner will check node locations for each fake/real node in path and set edge path to go through middle of it.
type StraightEdgePathAssigner struct{}

func (l StraightEdgePathAssigner) UpdateGraphLayout(g Graph, lg LayeredGraph, allNodesXY map[uint64]Position) {
	numAssignedEdges := 0
	for e, nodes := range lg.Edges {
		if _, ok := g.Edges[e]; !ok {
			panic(fmt.Errorf("layered graph edge(%v) is not found in the original graph", e))
		}

		path := make([]Position, len(nodes))
		for i, n := range nodes {
			path[i] = allNodesXY[n]
		}

		g.Edges[e] = Edge{Path: path}
		numAssignedEdges++
	}

	if numAssignedEdges != len(g.Edges) {
		panic(fmt.Errorf("layered graph has wrong number of edges(%d) vs graph num edges (%d)", numAssignedEdges, len(g.Edges)))
	}
}
