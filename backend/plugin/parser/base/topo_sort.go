package base

import (
	"slices"

	"github.com/pkg/errors"
)

func NewGraph() *Graph {
	return &Graph{
		NodeMap: make(map[string]*Node),
	}
}

type Graph struct {
	NodeMap  map[string]*Node
	EdgeList []*Edge
}

type Node struct {
	ID string
}

type Edge struct {
	Start string
	End   string
}

func (g *Graph) AddNode(id string) {
	g.NodeMap[id] = &Node{ID: id}
}

func (g *Graph) AddEdge(start, end string) {
	if _, ok := g.NodeMap[start]; !ok {
		g.AddNode(start)
	}
	if _, ok := g.NodeMap[end]; !ok {
		g.AddNode(end)
	}
	g.EdgeList = append(g.EdgeList, &Edge{Start: start, End: end})
}

func (g *Graph) TopologicalSort() ([]string, error) {
	var result []string
	inDegree := make(map[string]int)
	outEdge := make(map[string][]*Edge)

	for _, edge := range g.EdgeList {
		inDegree[edge.End]++
		outEdge[edge.Start] = append(outEdge[edge.Start], edge)
	}

	var queue []string
	for id := range g.NodeMap {
		if inDegree[id] == 0 {
			queue = append(queue, id)
		}
	}
	slices.Sort(queue)

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		for _, edge := range outEdge[node] {
			inDegree[edge.End]--
			if inDegree[edge.End] == 0 {
				queue = append(queue, edge.End)
			}
		}
		result = append(result, node)
	}

	if len(result) != len(g.NodeMap) {
		return nil, errors.Errorf("graph has cycle")
	}

	return result, nil
}
