package layout_test

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/nikolaydubina/jsonl-graph/graph"

	"github.com/nikolaydubina/go-graph-layout/layout"
	"github.com/nikolaydubina/go-graph-layout/svg"
)

func parseJSONLGraph(in string) (*graph.Graph, *layout.Graph, error) {
	gd, err := graph.NewGraphFromJSONL(strings.NewReader(in))
	if err != nil {
		return nil, nil, err
	}

	gl := layout.Graph{
		Nodes: make(map[uint64]layout.Node),
		Edges: make(map[[2]uint64]layout.Edge),
	}

	for id, node := range gd.Nodes {
		// compute w and h for nodes, since width and height of node depends on content
		rnodeData := node
		rnode := svg.Node{
			Title:    node.ID(),
			NodeData: rnodeData,
		}
		gl.Nodes[id] = layout.Node{W: rnode.Width(), H: rnode.Height()}
	}

	for e := range gd.Edges {
		gl.Edges[e] = layout.Edge{}
	}

	return &gd, &gl, nil
}

func writeSVG(gd graph.Graph, gl layout.Graph) string {
	graph := svg.Graph{
		ID:    "graph-root",
		Nodes: map[uint64]svg.Node{},
		Edges: map[[2]uint64]svg.Edge{},
	}

	for id, node := range gd.Nodes {
		graph.Nodes[id] = svg.Node{
			ID:       fmt.Sprintf("%d", id),
			X:        gl.Nodes[id].X,
			Y:        gl.Nodes[id].Y,
			Title:    node.ID(),
			NodeData: node,
		}
	}

	for e, edata := range gl.Edges {
		graph.Edges[e] = svg.Edge{
			Path: edata.Path,
		}
	}

	svgContainer := svg.SVG{
		ID:          "svg-root",
		Definitions: []svg.Renderable{},
		Body:        graph,
	}
	return svgContainer.Render()
}

//go:embed testdata/gin.jsonl
var ginJSONL string

//go:embed testdata/small.jsonl
var smallJSONL string

//go:embed testdata/brandeskopf.jsonl
var brandeskopfJSONL string

func TestE2E(t *testing.T) {
	inputJSONLGraphs := []struct {
		name            string
		inputJSONLGraph string
	}{
		{
			name:            "gin",
			inputJSONLGraph: ginJSONL,
		},
		{
			name:            "small",
			inputJSONLGraph: smallJSONL,
		},
		{
			name:            "brandeskopf",
			inputJSONLGraph: brandeskopfJSONL,
		},
	}

	layouts := []struct {
		name string
		l    layout.Layout
	}{
		{
			name: "forces",
			l: layout.SequenceLayout{
				Layouts: []layout.Layout{
					layout.ForceGraphLayout{
						Delta:    1,
						MaxSteps: 5000,
						Epsilon:  1.5,
						Forces: []layout.Force{
							layout.GravityForce{
								K:         -50,
								EdgesOnly: false,
							},
							layout.SpringForce{
								K:         0.2,
								L:         200,
								EdgesOnly: true,
							},
						},
					},
					layout.DirectEdgesLayout{},
				},
			},
		},
		{
			name: "eades",
			l: layout.SequenceLayout{
				Layouts: []layout.Layout{
					layout.EadesGonumLayout{
						Repulsion: 1,
						Rate:      0.05,
						Updates:   30,
						Theta:     0.2,
						ScaleX:    0.5,
						ScaleY:    0.5,
					},
					layout.DirectEdgesLayout{},
				},
			},
		},
		{
			name: "isomap",
			l: layout.SequenceLayout{
				Layouts: []layout.Layout{
					layout.IsomapR2GonumLayout{
						ScaleX: 0.5,
						ScaleY: 0.5,
					},
					layout.DirectEdgesLayout{},
				},
			},
		},
		{
			name: "layers",
			l: layout.SugiyamaLayersStrategyGraphLayout{
				CycleRemover:   layout.NewSimpleCycleRemover(),
				LevelsAssigner: layout.NewLayeredGraph,
				OrderingAssigner: layout.WarfieldOrderingOptimizer{
					Epochs:                   100,
					LayerOrderingInitializer: layout.BFSOrderingInitializer{},
					LayerOrderingOptimizer: layout.CompositeLayerOrderingOptimizer{
						Optimizers: []layout.LayerOrderingOptimizer{
							layout.WMedianOrderingOptimizer{},
							layout.SwitchAdjacentOrderingOptimizer{},
						},
					},
				}.Optimize,
				NodesHorizontalCoordinatesAssigner: layout.BrandesKopfLayersNodesHorizontalAssigner{
					Delta: 25,
				},
				NodesVerticalCoordinatesAssigner: layout.BasicNodesVerticalCoordinatesAssigner{
					MarginLayers:   25,
					FakeNodeHeight: 25,
				},
				EdgePathAssigner: layout.StraightEdgePathAssigner{}.UpdateGraphLayout,
			},
		},
	}
	for _, inputJSONLGraph := range inputJSONLGraphs {
		for _, l := range layouts {
			name := fmt.Sprintf("testdata/%s_%s.svg", inputJSONLGraph.name, l.name)
			t.Run(name, func(t *testing.T) {
				gd, gl, err := parseJSONLGraph(inputJSONLGraph.inputJSONLGraph)
				if err != nil {
					t.Error(err)
				}

				l.l.UpdateGraphLayout(*gl)
				svgResult := writeSVG(*gd, *gl)

				outputfile, err := os.Create(name)
				if err != nil {
					t.Error(err)
				}

				if _, err := outputfile.WriteString(svgResult); err != nil {
					t.Error(err)
				}
				if err := outputfile.Close(); err != nil {
					t.Error(err)
				}
			})
		}
	}
}
