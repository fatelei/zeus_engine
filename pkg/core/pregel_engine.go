package core

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// PregelEngine implements the Bulk Synchronous Parallel (BSP) model from the Pregel paper.
// Key concepts:
// 1. Supersteps: Synchronized computation phases
// 2. Message Passing: Nodes communicate via messages between supersteps
// 3. Vote to Halt: Nodes can deactivate when done
// 4. Combiners: Reduce message traffic by combining messages
type PregelEngine struct {
	graph      *Graph
	pool       *VariablePool
	
	// Message queues: double buffered (current superstep, next superstep)
	currentMessages map[string][]Message
	nextMessages    map[string][]Message
	messageMutex    sync.RWMutex
	
	// Active vertex tracking
	activeNodes     map[string]bool
	activeNodeMutex sync.RWMutex
	
	// Aggregators for global communication
	aggregators     map[string]*Aggregator
	aggregatorMutex sync.RWMutex
	
	// Superstep counter
	superstep int64
	
	// Worker pool
	workerCount int
	
	// Combiner for reducing messages
	combiner Combiner
}

// Message represents a message sent between nodes
type Message struct {
	TargetID string
	Value    interface{}
}

// Combiner reduces multiple messages to a single message
type Combiner interface {
	Combine(messages []Message) Message
}

// Aggregator performs global reduction operations
type Aggregator struct {
	name   string
	value  interface{}
	reduce func(a, b interface{}) interface{}
	mu     sync.Mutex
}

func NewPregelEngine(graph *Graph, pool *VariablePool, workers int) *PregelEngine {
	return &PregelEngine{
		graph:           graph,
		pool:            pool,
		currentMessages: make(map[string][]Message),
		nextMessages:    make(map[string][]Message),
		activeNodes:     make(map[string]bool),
		aggregators:     make(map[string]*Aggregator),
		workerCount:     workers,
		superstep:       0,
	}
}

func (e *PregelEngine) SetCombiner(c Combiner) {
	e.combiner = c
}

// Run executes the graph using the BSP model
func (e *PregelEngine) Run(ctx context.Context) error {
	fmt.Println("[PregelEngine] Starting BSP execution")
	
	// Initialize: all nodes start active
	for nodeID := range e.graph.Nodes {
		e.activeNodes[nodeID] = true
		if e.graph.InDegree[nodeID] == 0 {
			// Send initial "wakeup" message to root nodes
			e.SendMessage(Message{TargetID: nodeID, Value: nil})
		}
	}
	
	// Main BSP loop
	for {
		superstepNum := atomic.LoadInt64(&e.superstep)
		fmt.Printf("\n[PregelEngine] ===== Superstep %d =====\n", superstepNum)
		
		// Swap message buffers
		e.swapMessageBuffers()
		
	// Check termination: no active nodes and no messages
		if len(e.activeNodes) == 0 && len(e.currentMessages) == 0 {
			fmt.Println("[PregelEngine] All nodes halted. Terminating.")
			break
		}
		
		// Execute superstep
		// We must process active nodes AND nodes with messages.
		// getNodesWithMessages now returns union of active nodes + message targets
		if err := e.executeSuperstep(ctx); err != nil {
			return err
		}
		
		// Increment superstep counter
		atomic.AddInt64(&e.superstep, 1)
		
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	
	fmt.Println("[PregelEngine] Execution complete")
	return nil
}

func (e *PregelEngine) executeSuperstep(ctx context.Context) error {
	var wg sync.WaitGroup
	
	// Apply combiners if configured
	if e.combiner != nil {
		e.applyCombiner()
	}
	
	// Process nodes with messages (including reactivated nodes)
	nodesBatch := e.getNodesWithMessages()
	
	// Distribute work across workers
	workChan := make(chan string, len(nodesBatch))
	for nodeID := range nodesBatch {
		workChan <- nodeID
	}
	close(workChan)
	
	// Start workers
	for i := 0; i < e.workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for nodeID := range workChan {
				if err := e.computeNode(ctx, nodeID); err != nil {
					fmt.Printf("[Worker %d] Error processing node %s: %v\n", workerID, nodeID, err)
				}
			}
		}(i)
	}
	
	wg.Wait()
	return nil
}

func (e *PregelEngine) computeNode(ctx context.Context, nodeID string) error {
	node, exists := e.graph.Nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}
	
	// Gather messages for this node
	e.messageMutex.RLock()
	messages := e.currentMessages[nodeID]
	e.messageMutex.RUnlock()
	
	// If node has no messages and was already inactive, skip
	if len(messages) == 0 && !e.activeNodes[nodeID] {
		return nil
	}

	// Build inputs map from messages. 
	// If messages contain output maps, merge them.
	inputs := make(map[string]interface{})
	for _, msg := range messages {
		if valMap, ok := msg.Value.(map[string]interface{}); ok {
			for k, v := range valMap {
				inputs[k] = v
			}
		} else {
			// Fallback for non-map messages (e.g. initial nil message)
			// We don't need to store nil values
		}
	}
	
	// Execute node computation
	// In Pregel, Compute() is called every superstep for active nodes
	output, err := node.Run(ctx, inputs)
	if err != nil {
		return err
	}
	
	fmt.Printf("  [Compute] Node %s executed (status: %s)\n", nodeID, output.Status)
	
	// Handle node status
	if output.Status == StatusSucceeded {
		// Write outputs to variable pool
		for k, v := range output.Outputs {
			e.pool.Set(nodeID, k, v)
		}
		
		// Propagate to downstream
		e.propagateToDownstream(nodeID, output.Outputs)
		
		// VOTE TO HALT: In this simple DAG model, a node halts after successful execution.
		// It will only wake up if it receives a new message in a future superstep.
		e.activeNodeMutex.Lock()
		delete(e.activeNodes, nodeID)
		e.activeNodeMutex.Unlock()
	}
	
	return nil
}

func (e *PregelEngine) propagateToDownstream(nodeID string, outputs map[string]interface{}) {
	neighbors := e.graph.Adjacency[nodeID]
	if len(neighbors) == 0 {
		// Leaf node - no propagation
		return
	}
	
	// CRITICAL: Only send messages if there are actual outputs
	// Empty outputs means node has voted to halt and no downstream propagation
	if len(outputs) == 0 {
		fmt.Printf("  [Propagate] Node %s has no outputs. No messages sent.\n", nodeID)
		return
	}
	
	for _, neighborID := range neighbors {
		// Send message to each neighbor
		e.SendMessage(Message{
			TargetID: neighborID,
			Value:    outputs,
		})
		
		// Reactivate neighbor (it will run in next superstep)
		// NOTE: In true Pregel, reactivation happens automatically when a message arrives
		e.activeNodeMutex.Lock()
		e.activeNodes[neighborID] = true
		e.activeNodeMutex.Unlock()
	}
}

// SendMessage queues a message for delivery in the next superstep
func (e *PregelEngine) SendMessage(msg Message) {
	e.messageMutex.Lock()
	defer e.messageMutex.Unlock()
	e.nextMessages[msg.TargetID] = append(e.nextMessages[msg.TargetID], msg)
}

func (e *PregelEngine) swapMessageBuffers() {
	e.messageMutex.Lock()
	defer e.messageMutex.Unlock()
	
	// Move next to current
	e.currentMessages = e.nextMessages
	e.nextMessages = make(map[string][]Message)
}

func (e *PregelEngine) getNodesWithMessages() map[string]bool {
	e.activeNodeMutex.RLock()
	// e.messageMutex is already held or not needed for keys if we are careful, 
	// but let's lock to be safe as we access currentMessages keys
	e.messageMutex.RLock()
	defer e.messageMutex.RUnlock()
	defer e.activeNodeMutex.RUnlock()
	
	nodes := make(map[string]bool)
	
	// active nodes
	for nodeID := range e.activeNodes {
		nodes[nodeID] = true
	}
	
	// nodes with messages (they become active)
	for nodeID := range e.currentMessages {
		nodes[nodeID] = true
	}
	
	return nodes
}

func (e *PregelEngine) applyCombiner() {
	if e.combiner == nil {
		return
	}
	
	e.messageMutex.Lock()
	defer e.messageMutex.Unlock()
	
	for targetID, messages := range e.currentMessages {
		if len(messages) > 1 {
			combined := e.combiner.Combine(messages)
			e.currentMessages[targetID] = []Message{combined}
			fmt.Printf("  [Combiner] Reduced %d messages to 1 for node %s\n", len(messages), targetID)
		}
	}
}

// RegisterAggregator adds a global aggregator
func (e *PregelEngine) RegisterAggregator(name string, initial interface{}, reduceFn func(a, b interface{}) interface{}) {
	e.aggregatorMutex.Lock()
	defer e.aggregatorMutex.Unlock()
	
	e.aggregators[name] = &Aggregator{
		name:   name,
		value:  initial,
		reduce: reduceFn,
	}
}

// GetAggregator retrieves the current value of an aggregator
func (e *PregelEngine) GetAggregator(name string) (interface{}, bool) {
	e.aggregatorMutex.RLock()
	defer e.aggregatorMutex.RUnlock()
	
	agg, exists := e.aggregators[name]
	if !exists {
		return nil, false
	}
	return agg.value, true
}
