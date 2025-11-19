package core

import (
	"context"
	"fmt"
	"sync"
)

type Graph struct {
	Nodes     map[string]Node
	Adjacency map[string][]string // NodeID -> []NextNodeIDs
	InDegree  map[string]int      // NodeID -> count of incoming edges
}

type Engine struct {
	graph      *Graph
	pool       *VariablePool
	readyQueue chan Node
	eventQueue chan NodeOutput
	wg         sync.WaitGroup
}

func NewEngine(graph *Graph, pool *VariablePool) *Engine {
	return &Engine{
		graph:      graph,
		pool:       pool,
		readyQueue: make(chan Node, 100),
		eventQueue: make(chan NodeOutput, 100),
	}
}

func (e *Engine) Run(ctx context.Context) error {
	// Find start nodes (InDegree 0)
	for id, node := range e.graph.Nodes {
		if e.graph.InDegree[id] == 0 {
			e.readyQueue <- node
		}
	}

	// Start workers
	workerCount := 3
	for i := 0; i < workerCount; i++ {
		e.wg.Add(1)
		go e.worker(ctx)
	}

	// Start dispatcher to handle events and scheduling
	// We use a separate goroutine for dispatching to keep the main Run blocking or manageable
	// We can wait for the dispatcher to signal completion or context done
	dispatcherDone := make(chan struct{})
	go func() {
		e.dispatcher(ctx)
		close(dispatcherDone)
	}()
	
	// Wait for context done or manual stop
	select {
	case <-ctx.Done():
	case <-dispatcherDone:
	}

	e.wg.Wait()
	return nil
}

func (e *Engine) worker(ctx context.Context) {
	defer e.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case node, ok := <-e.readyQueue:
			if !ok {
				return
			}
			// Build inputs from pool (for legacy engine)
			inputs := make(map[string]interface{})
			output, err := node.Run(ctx, inputs)
			if err != nil {
				output = &NodeOutput{
					NodeID: node.ID(),
					Status: StatusFailed,
					Error:  err,
				}
			} else {
				output.Status = StatusSucceeded
				output.NodeID = node.ID()
			}
			e.eventQueue <- *output
		}
	}
}

func (e *Engine) dispatcher(ctx context.Context) {
	// Track incoming edge completions for DAG execution
	// Map: NodeID -> Number of completed incoming edges
	completedInputs := make(map[string]int)
	
	// Track total active nodes to detect idle state (optional, but good for termination)
	// For now we just rely on flow.

	for {
		select {
		case <-ctx.Done():
			return
		case output := <-e.eventQueue:
			fmt.Printf("[Engine] Node '%s' finished with status %s\n", output.NodeID, output.Status)
			
			if output.Status == StatusSucceeded {
				// Write outputs to pool
				for k, v := range output.Outputs {
					e.pool.Set(output.NodeID, k, v)
				}

				// Find next nodes
				nextIDs := e.graph.Adjacency[output.NodeID]
				leafNode := true
				
				for _, nextID := range nextIDs {
					leafNode = false
					if nextNode, exists := e.graph.Nodes[nextID]; exists {
						// Check if ALL dependencies are met.
						completedInputs[nextID]++
						required := e.graph.InDegree[nextID]
						
						if completedInputs[nextID] >= required {
							fmt.Printf("[Engine] All dependencies met for %s (needs %d). Scheduling.\n", nextID, required)
							// Reset counter if we want to support loops? No, DAG only for now.
							e.readyQueue <- nextNode
						} else {
							fmt.Printf("[Engine] Node %s waiting for dependencies (%d/%d)\n", nextID, completedInputs[nextID], required)
						}
					}
				}
				
				if leafNode {
					fmt.Printf("[Engine] Node '%s' is a leaf. Path complete.\n", output.NodeID)
				}
			}
		}
	}
}
