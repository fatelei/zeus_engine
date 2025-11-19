package nodes

import (
	"context"
	"testing"

	"github.com/fatelei/zeus/pkg/core"
)

func TestStartNode(t *testing.T) {
	cfg := core.NodeConfig{
		ID:     "start-1",
		Type:   "start",
		Config: map[string]interface{}{},
	}
	node := NewStartNode(cfg)

	if node.ID() != "start-1" {
		t.Errorf("Expected ID start-1, got %s", node.ID())
	}
	if node.Type() != "start" {
		t.Errorf("Expected Type start, got %s", node.Type())
	}

	// Test Run (First execution)
	ctx := context.Background()
	output, err := node.Run(ctx, map[string]interface{}{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if output.Status != core.StatusSucceeded {
		t.Errorf("Expected status succeeded, got %s", output.Status)
	}
	if output.Outputs["_executed"] != true {
		t.Error("Expected _executed output to be true")
	}

	// Test Run (Already executed)
	output, err = node.Run(ctx, map[string]interface{}{"_executed": true})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(output.Outputs) != 0 {
		t.Error("Expected no outputs for already executed node")
	}
}

func TestPrintNode(t *testing.T) {
	cfg := core.NodeConfig{
		ID:     "print-1",
		Type:   "print",
		Config: map[string]interface{}{"message": "test"},
	}
	node := NewPrintNode(cfg)

	// Test Run with no inputs (should halt)
	ctx := context.Background()
	output, err := node.Run(ctx, map[string]interface{}{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(output.Outputs) != 0 {
		t.Error("Expected no outputs when no inputs provided")
	}

	// Test Run with inputs
	output, err = node.Run(ctx, map[string]interface{}{"some_input": "data"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if output.Status != core.StatusSucceeded {
		t.Errorf("Expected status succeeded, got %s", output.Status)
	}
	if output.Outputs["printed"] != true {
		t.Error("Expected printed output to be true")
	}
}

func TestEndNode(t *testing.T) {
	cfg := core.NodeConfig{
		ID:     "end-1",
		Type:   "end",
		Config: map[string]interface{}{},
	}
	node := NewEndNode(cfg)

	// Test Run with inputs
	ctx := context.Background()
	output, err := node.Run(ctx, map[string]interface{}{"some_input": "data"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(output.Outputs) != 0 {
		t.Error("Expected no outputs for end node")
	}
}
