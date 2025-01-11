package layout

// BasicNodesVerticalCoordinatesAssigner will check maximum height in each layer.
// It will keep each node vertically in the middle within each layer.
type BasicNodesVerticalCoordinatesAssigner struct {
	MarginLayers   int // distance between layers
	FakeNodeHeight int
}

func layersMaxHeights(g Graph, layers [][]uint64) []int {
	hmax := make([]int, len(layers))
	for i, nodes := range layers {
		for _, node := range nodes {
			if hmax[i] < g.Nodes[node].H {
				hmax[i] = g.Nodes[node].H
			}
		}
	}
	return hmax
}

func (s BasicNodesVerticalCoordinatesAssigner) NodesVerticalCoordinates(g Graph, lg LayeredGraph) map[uint64]int {
	nodeY := make(map[uint64]int, len(lg.NodePosition))

	layers := lg.Layers()
	layersHMax := layersMaxHeights(g, layers)

	yOffset := 0
	for i, nodes := range layers {
		for _, node := range nodes {
			// put in the middle vertically
			nodeY[node] = yOffset + (layersHMax[i]) / 2
		}

		// move to next layer
		yOffset += layersHMax[i] + s.MarginLayers
	}

	return nodeY
}
