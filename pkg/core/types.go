package core

import "context"

type ExecutionStatus string

const (
	StatusSucceeded ExecutionStatus = "succeeded"
	StatusFailed    ExecutionStatus = "failed"
	StatusRunning   ExecutionStatus = "running"
	StatusSkipped   ExecutionStatus = "skipped"
)

// NodeOutput represents the result of a node execution
type NodeOutput struct {
	NodeID  string
	Status  ExecutionStatus
	Outputs map[string]interface{}
	Error   error
}

// NodeConfig represents the static configuration from DSL
type NodeConfig struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"data"` // Changed 'data' to 'config' concept
}

// GraphDefinition represents the JSON DSL
type GraphDefinition struct {
	Nodes []NodeConfig `json:"nodes"`
	Edges []Edge       `json:"edges"`
}

type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// Node interface for all workflow nodes
// Based on Pregel's vertex-centric model:
// Vertices receive messages (inputs), perform computation, and send messages (outputs).
type Node interface {
	ID() string
	Type() string
	// Run executes the node logic.
	// inputs: map of input names to values received from upstream nodes.
	Run(ctx context.Context, inputs map[string]interface{}) (*NodeOutput, error)
}
