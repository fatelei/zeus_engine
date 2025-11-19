# Zeus Workflow Engine

Zeus is a lightweight, event-driven, concurrent workflow engine written in Go. It is designed to execute Directed Acyclic Graphs (DAGs) of tasks defined in a simple JSON DSL.

It demonstrates how to build a scalable orchestration engine using Go's concurrency patterns (Goroutines and Channels) to replace traditional thread-based approaches.

## Features

*   **Concurrent Execution**: Automatically runs independent branches of the graph in parallel.
*   **DAG Support**: Handles dependencies correctly, ensuring downstream nodes only run when all upstream dependencies are satisfied (Fan-In/Join).
*   **Event-Driven**: Uses an internal event loop to manage state transitions and scheduling.
*   **Extensible**: Interface-based design makes it easy to add new node types (e.g., LLM calls, API requests).
*   **JSON DSL**: Workflows are defined in a portable JSON format.

## Architecture

Zeus implements the **Pregel Bulk Synchronous Parallel (BSP)** model for distributed graph processing:

```
┌─────────────────────────────────────────────────────────────────┐
│                         PREGEL ENGINE                           │
│                                                                 │
│  ┌──────────────┐      ┌──────────────┐      ┌──────────────┐ │
│  │   Master     │      │   Worker 1   │      │   Worker N   │ │
│  │              │      │              │      │              │ │
│  │ - Coordinates│      │ - Executes   │      │ - Executes   │ │
│  │ - Tracks     │◄────►│   Nodes      │ ... │   Nodes      │ │
│  │   Supersteps │      │ - Sends Msgs │      │ - Sends Msgs │ │
│  └──────┬───────┘      └──────┬───────┘      └──────┬───────┘ │
│         │                     │                     │          │
│         ▼                     ▼                     ▼          │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │              MESSAGE BUFFERS (Double Buffered)           │ │
│  │   Current Superstep   │   Next Superstep                │ │
│  │   ┌──────────────┐    │   ┌──────────────┐             │ │
│  │   │ Node A: []msg│    │   │ Node B: []msg│             │ │
│  │   └──────────────┘    │   └──────────────┘             │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │                    VARIABLE POOL                         │ │
│  │    (Thread-Safe State Store with sync.RWMutex)          │ │
│  │    NodeID -> VariableName -> Value                       │ │
│  └──────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### BSP Execution Flow

```
Superstep 0        Superstep 1        Superstep 2        Superstep 3
───────────────    ───────────────    ───────────────    ───────────────
  ┌─────┐            ┌─────┐            ┌─────┐            ┌─────┐
  │Start│            │     │            │     │            │     │
  └──┬──┘            └─────┘            └─────┘            └─────┘
     │                                                           
     │ msg                                                       
     ▼                                                           
  ┌─────┐            ┌─────┐            ┌─────┐            ┌─────┐
  │Print│  ────msg──►│Print│            │     │            │     │
  │  1  │            │  1  │            │  1  │            │  1  │
  └─────┘            └──┬──┘            └─────┘            └─────┘
                        │                                         
                    ┌───┴───┐                                    
                    │       │                                    
                  msg1    msg2                                   
                    │       │                                    
                    ▼       ▼                                    
  ┌─────┐  ┌─────┐      ┌─────┐  ┌─────┐      ┌─────┐  ┌─────┐
  │Print│  │Print│      │Print│  │Print│      │Print│  │Print│
  │  2  │  │  3  │      │  2  │  │  3  │      │  2  │  │  3  │
  └─────┘  └─────┘      └──┬──┘  └──┬──┘      └─────┘  └─────┘
                            │        │                           
                            └───┬────┘                           
                                │                                
                              msg                                
                                ▼                                
  ┌─────┐            ┌─────┐            ┌─────┐            ┌─────┐
  │ End │            │ End │            │ End │            │ End │
  └─────┘            └─────┘            └─────┘            └──┬──┘
                                                               │
                                                             HALT
```

### Key Components

*   **PregelEngine**: Implements the BSP model with superstep synchronization
*   **Worker Pool**: Fixed number of Goroutines processing nodes in parallel
*   **Message Passing**: Asynchronous message queues with double buffering
*   **Active Set**: Tracks which nodes need computation in each superstep
*   **Combiners**: Optional message reduction to minimize network traffic
*   **Aggregators**: Global reduction operations across all nodes
*   **Variable Pool**: Thread-safe persistent state storage

## Getting Started

### Prerequisites

*   Go 1.25 or higher

### Installation

Clone the repository:

```bash
git clone https://github.com/fatelei/zeus.git
cd zeus
```

### Running the Demo

1.  **Define a Workflow**: Check `workflow.json` for an example graph.
2.  **Run the Engine**:

```bash
go run main.go
```

You should see output indicating parallel execution of branches and a final aggregation step.

## Defining Workflows

Workflows are defined in JSON. The following example demonstrates a graph with parallel execution branches:

```json
{
    "nodes": [
        {
            "id": "start-1",
            "type": "start",
            "data": {}
        },
        {
            "id": "print-1",
            "type": "print",
            "data": {
                "message": "Hello from Zeus! Processing step 1."
            }
        },
        {
            "id": "print-2",
            "type": "print",
            "data": {
                "message": "Parallel Step 2A"
            }
        },
        {
            "id": "print-3",
            "type": "print",
            "data": {
                "message": "Parallel Step 2B"
            }
        },
        {
            "id": "end-1",
            "type": "end",
            "data": {}
        }
    ],
    "edges": [
        {
            "source": "start-1",
            "target": "print-1"
        },
        {
            "source": "print-1",
            "target": "print-2"
        },
        {
            "source": "print-1",
            "target": "print-3"
        },
        {
            "source": "print-2",
            "target": "end-1"
        },
        {
            "source": "print-3",
            "target": "end-1"
        }
    ]
}
```

### Supported Node Types

*   `start`: The entry point of the workflow.
*   `end`: The terminal point.
*   `print`: A demo node that prints a message to stdout (simulating work).

## Extending Zeus

To add a new node type (e.g., `LLMNode`), implement the `Node` interface:

```go
// pkg/nodes/my_node.go

type MyNode struct {
    BaseNode
}

func NewMyNode(cfg core.NodeConfig) core.Node {
    return &MyNode{...}
}

func (n *MyNode) Run(ctx context.Context, pool *core.VariablePool) (*core.NodeOutput, error) {
    // Your logic here
    return &core.NodeOutput{Status: core.StatusSucceeded}, nil
}

func init() {
    Register("my-node-type", NewMyNode)
}
```
