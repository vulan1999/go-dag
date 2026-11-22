package pipeline

import "encoding/json"

type Pipeline struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type Node struct {
	ID     string          `json:"id"`
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
}

type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}
