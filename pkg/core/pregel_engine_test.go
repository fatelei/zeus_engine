package core

import (
	"context"
	"testing"
)

// MockNode is a simple node for testing
type MockNode struct {
	id     string
	runFn  func(ctx context.Context, inputs map[string]interface{}) (*NodeOutput, error)
}

func (n *MockNode) ID() string { return n.id }
func (n *MockNode) Type() string { return "mock" }
func (n *MockNode) Run(ctx context.Context, inputs map[string]interface{}) (*NodeOutput, error) {
	if n.runFn != nil {
		return n.runFn(ctx, inputs)
	}
	return &NodeOutput{Status: StatusSucceeded, Outputs: map[string]interface{}{"out": "val"}}, nil
}

func TestPregelEngine_Run_Simple(t *testing.T) {
	// Create a simple graph: A -> B
	nodeA := &MockNode{id: "A"}
	nodeB := &MockNode{id: "B"}
	
	// Node A logic: Emit output
	nodeA.runFn = func(ctx context.Context, inputs map[string]interface{}) (*NodeOutput, error) {
		// Run only if not already run (simulated by inputs check, though in this test we rely on engine logic)
		// For simple test, we just emit.
		// But wait, engine will re-run if we emit? 
		// The engine re-runs nodes if they have messages.
		// Initial step: A has in-degree 0, so it runs.
		
		// To prevent infinite loop in this simple mock without internal state:
		// We check if we received inputs. Node A (root) receives nil input initially.
		// But if we want it to stop, we should return empty outputs eventually?
		// Actually, in the engine:
		// "If node has no messages and was already inactive, skip"
		// "If output.Outputs is empty, no downstream propagation"
		// So, A should emit once.
		
		// We can rely on the fact that A only receives the initial wake-up message.
		if val, ok := inputs["processed"]; ok && val.(bool) {
             return &NodeOutput{Status: StatusSucceeded, Outputs: nil}, nil
        }

		return &NodeOutput{Status: StatusSucceeded, Outputs: map[string]interface{}{"val": "fromA", "processed": true}}, nil
	}
	
	// Node B logic: Receive input, emit nothing (halt)
	nodeB.runFn = func(ctx context.Context, inputs map[string]interface{}) (*NodeOutput, error) {
		if val, ok := inputs["val"]; ok && val == "fromA" {
			return &NodeOutput{Status: StatusSucceeded, Outputs: map[string]interface{}{"val": "fromB"}}, nil
		}
		return &NodeOutput{Status: StatusSucceeded, Outputs: nil}, nil
	}

	graph := &Graph{
		Nodes: map[string]Node{
			"A": nodeA,
			"B": nodeB,
		},
		Adjacency: map[string][]string{
			"A": {"B"},
		},
		InDegree: map[string]int{
			"A": 0,
			"B": 1,
		},
	}

	pool := NewVariablePool()
	engine := NewPregelEngine(graph, pool, 1)

	if err := engine.Run(context.Background()); err != nil {
		t.Errorf("Engine execution failed: %v", err)
	}

	// Verify B ran and produced output in pool
	val, ok := pool.Get("B", "val")
	if !ok {
		t.Error("Expected B to produce output in pool")
	}
	if val != "fromB" {
		t.Errorf("Expected fromB, got %v", val)
	}
}

func TestPregelEngine_Combiner(t *testing.T) {
	// Test Combiner Logic
	engine := &PregelEngine{
		currentMessages: make(map[string][]Message),
	}
	
	// Add messages for Node A
	engine.currentMessages["A"] = []Message{
		{Value: 1},
		{Value: 2},
		{Value: 3},
	}
	
	mockCombiner := &MockCombiner{}
	engine.SetCombiner(mockCombiner)
	
	engine.applyCombiner()
	
	msgs := engine.currentMessages["A"]
	if len(msgs) != 1 {
		t.Errorf("Expected 1 message after combine, got %d", len(msgs))
	}
	if msgs[0].Value != 6 {
		t.Errorf("Expected combined value 6, got %v", msgs[0].Value)
	}
}

type MockCombiner struct{}

func (c *MockCombiner) Combine(messages []Message) Message {
	sum := 0
	for _, msg := range messages {
		if val, ok := msg.Value.(int); ok {
			sum += val
		}
	}
	return Message{Value: sum}
}
