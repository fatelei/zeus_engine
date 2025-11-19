package nodes

import (
	"context"
	"fmt"
	"time"

	"github.com/fatelei/zeus/pkg/core"
)

// Registry holds node constructors
var Registry = map[string]func(core.NodeConfig) core.Node{}

func Register(nodeType string, constructor func(core.NodeConfig) core.Node) {
	Registry[nodeType] = constructor
}

// --- Base Node ---

type BaseNode struct {
	id     string
	typ    string
	config map[string]interface{}
}

func (n *BaseNode) ID() string {
	return n.id
}

func (n *BaseNode) Type() string {
	return n.typ
}

// --- Concrete Nodes ---

// StartNode
type StartNode struct {
	BaseNode
}

func NewStartNode(cfg core.NodeConfig) core.Node {
	return &StartNode{
		BaseNode: BaseNode{id: cfg.ID, typ: cfg.Type, config: cfg.Config},
	}
}

func (n *StartNode) Run(ctx context.Context, inputs map[string]interface{}) (*core.NodeOutput, error) {
	// In Pregel model: check if this node has already been processed
	if alreadyRun, ok := inputs["_executed"]; ok && alreadyRun == true {
		// Already executed - vote to halt without producing new outputs
		fmt.Printf("  -> [%s] Already executed, halting\n", n.ID())
		return &core.NodeOutput{
			Status:  core.StatusSucceeded,
			Outputs: map[string]interface{}{},
		}, nil
	}
	
	fmt.Printf("  -> [%s] Start execution\n", n.ID())
	return &core.NodeOutput{
		Status: core.StatusSucceeded,
		Outputs: map[string]interface{}{
			"start_time": time.Now().String(),
			"_executed":  true, // Mark as executed for downstream
		},
	}, nil
}

// EndNode
type EndNode struct {
	BaseNode
}

func NewEndNode(cfg core.NodeConfig) core.Node {
	return &EndNode{
		BaseNode: BaseNode{id: cfg.ID, typ: cfg.Type, config: cfg.Config},
	}
}

func (n *EndNode) Run(ctx context.Context, inputs map[string]interface{}) (*core.NodeOutput, error) {
	// If no inputs, we can't do anything (unless we are a source node).
	// Vote to halt and wait for data.
	if len(inputs) == 0 {
		return &core.NodeOutput{
			Status:  core.StatusSucceeded,
			Outputs: map[string]interface{}{},
		}, nil
	}

	fmt.Printf("  -> [%s] End execution. Inputs: %v\n", n.ID(), inputs)
	
	// End node doesn't need to emit downstream, effectively halting the path
	return &core.NodeOutput{
		Status:  core.StatusSucceeded,
		Outputs: map[string]interface{}{},
	}, nil
}

// PrintNode (Demo Node)
type PrintNode struct {
	BaseNode
}

func NewPrintNode(cfg core.NodeConfig) core.Node {
	return &PrintNode{
		BaseNode: BaseNode{id: cfg.ID, typ: cfg.Type, config: cfg.Config},
	}
}

func (n *PrintNode) Run(ctx context.Context, inputs map[string]interface{}) (*core.NodeOutput, error) {
	// If no inputs, halt.
	if len(inputs) == 0 {
		return &core.NodeOutput{
			Status:  core.StatusSucceeded,
			Outputs: map[string]interface{}{},
		}, nil
	}

	msg, _ := n.config["message"].(string)
	fmt.Printf("  -> [%s] PRINT: %s. Received %d inputs.\n", n.ID(), msg, len(inputs))
	
	// Simulate work
	time.Sleep(100 * time.Millisecond)
	
	return &core.NodeOutput{
		Status: core.StatusSucceeded,
		Outputs: map[string]interface{}{
			"printed":   true,
			"_executed": true,
		},
	}, nil
}

func init() {
	Register("start", NewStartNode)
	Register("end", NewEndNode)
	Register("print", NewPrintNode)
}
