# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Common Commands

### Build
Build the project using the standard Go toolchain.
```bash
go build -o zeus main.go
```

### Run
Run the demo workflow.
```bash
go run main.go
```

### Test
Currently, there are no tests in the `pkg` directory. When adding tests, use:
```bash
go test ./...
```

### Lint
Use `golangci-lint` to lint the codebase.
```bash
golangci-lint run
```

## Code Architecture

Zeus is a lightweight, event-driven workflow engine based on the **Pregel Bulk Synchronous Parallel (BSP)** model.

### Directory Structure
- `main.go`: Entry point. Loads the JSON DSL (`workflow.json`), constructs the graph, and starts the engine.
- `pkg/core/`: Core engine logic.
    - `pregel_engine.go`: Implements the BSP execution model (Supersteps, Message Passing).
    - `engine.go`: Contains the older/simpler engine implementation (optional/deprecated).
    - `types.go`: Defines `Node`, `Graph`, `Message`, and other core interfaces.
    - `variable_pool.go`: Thread-safe state storage (`VariablePool`) using `sync.RWMutex`.
- `pkg/nodes/`: Node implementations.
    - `basic_nodes.go`: Contains basic node types like `start`, `end`, and `print`.

### Key Concepts

#### 1. Graph Execution (BSP)
The execution is divided into **Supersteps**.
- **Compute**: In each superstep, active nodes process input messages and their current state.
- **Communicate**: Nodes send messages to downstream neighbors for the next superstep.
- **Sync**: The engine waits for all workers to finish the current superstep before moving to the next.
- **Double Buffering**: Messages are stored in two buffers (Current and Next) to ensure isolation between supersteps.

#### 2. Nodes
Nodes implement the `core.Node` interface.
- They are registered in `nodes.Registry` via `init()` functions.
- New node types can be added by creating a struct that embeds `BaseNode` and implementing the `Run` method.

#### 3. Data Flow
- **Edges**: Define the flow of execution/messages.
- **Variable Pool**: Allows nodes to read/write global variables, shared across the workflow (thread-safe).
