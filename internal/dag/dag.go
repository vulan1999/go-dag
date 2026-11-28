package dag

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

var (
	ErrInvalidNodeID = errors.New("invalid node id")
	ErrDuplicateNode = errors.New("duplicate node")
	ErrNodeNotFound  = errors.New("node not found")
	ErrDuplicateEdge = errors.New("duplicate edge")
	ErrCycleDetected = errors.New("cycle detected in DAG")
)

// DAG represents a directed acyclic graph of node IDs.
type DAG struct {
	nodes    map[string]struct{}
	children map[string][]string
	parents  map[string][]string
}

// New creates an empty DAG.
func New() *DAG {
	return &DAG{
		nodes:    make(map[string]struct{}),
		children: make(map[string][]string),
		parents:  make(map[string][]string),
	}
}

// AddNode registers a node ID in the graph.
func (d *DAG) AddNode(id string) error {
	if strings.TrimSpace(id) == "" {
		return ErrInvalidNodeID
	}
	if _, exists := d.nodes[id]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateNode, id)
	}
	d.nodes[id] = struct{}{}
	return nil
}

// AddEdge adds a directed edge from -> to.
func (d *DAG) AddEdge(from, to string) error {
	if _, ok := d.nodes[from]; !ok {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, from)
	}
	if _, ok := d.nodes[to]; !ok {
		return fmt.Errorf("%w: %s", ErrNodeNotFound, to)
	}
	if from == to {
		return fmt.Errorf("%w: self-loop on %s", ErrCycleDetected, from)
	}
	if slices.Contains(d.children[from], to) {
		return fmt.Errorf("%w: %s -> %s", ErrDuplicateEdge, from, to)
	}

	d.children[from] = append(d.children[from], to)
	d.parents[to] = append(d.parents[to], from)
	return nil
}

// Validate checks whether the graph is a valid acyclic DAG.
func (d *DAG) Validate() error {
	_, err := d.Levels()
	return err
}

// TopologicalSort returns a deterministic topological ordering of all nodes.
func (d *DAG) TopologicalSort() ([]string, error) {
	levels, err := d.Levels()
	if err != nil {
		return nil, err
	}

	order := make([]string, 0, len(d.nodes))
	for _, level := range levels {
		order = append(order, level...)
	}
	return order, nil
}

// Levels groups nodes into topological waves where nodes in the same wave have
// no dependencies on one another and can be executed concurrently.
func (d *DAG) Levels() ([][]string, error) {
	inDegree := make(map[string]int, len(d.nodes))
	for id := range d.nodes {
		inDegree[id] = len(d.parents[id])
	}

	var currentLevel []string
	for id, deg := range inDegree {
		if deg == 0 {
			currentLevel = append(currentLevel, id)
		}
	}
	slices.Sort(currentLevel)

	levels := make([][]string, 0)
	visited := 0

	for len(currentLevel) > 0 {
		levels = append(levels, currentLevel)
		visited += len(currentLevel)

		var nextLevel []string
		for _, u := range currentLevel {
			for _, v := range d.children[u] {
				inDegree[v]--
				if inDegree[v] == 0 {
					nextLevel = append(nextLevel, v)
				}
			}
		}
		slices.Sort(nextLevel)
		currentLevel = nextLevel
	}

	if visited != len(d.nodes) {
		return nil, ErrCycleDetected
	}

	return levels, nil
}

// Nodes returns a sorted slice of all node IDs in the graph.
func (d *DAG) Nodes() []string {
	res := make([]string, 0, len(d.nodes))
	for id := range d.nodes {
		res = append(res, id)
	}
	slices.Sort(res)
	return res
}

// Parents returns a sorted copy of the upstream node IDs for the given node.
func (d *DAG) Parents(id string) []string {
	res := slices.Clone(d.parents[id])
	if res == nil {
		return []string{}
	}
	slices.Sort(res)
	return res
}

// Children returns a sorted copy of the downstream node IDs for the given node.
func (d *DAG) Children(id string) []string {
	res := slices.Clone(d.children[id])
	if res == nil {
		return []string{}
	}
	slices.Sort(res)
	return res
}
