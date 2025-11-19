package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/fatelei/zeus/pkg/core"
	"github.com/fatelei/zeus/pkg/nodes"
)

func main() {
	fmt.Println("Starting Zeus Workflow Engine Demo...")

	// 1. Load DSL
	dslFile, err := os.Open("workflow.json")
	if err != nil {
		panic(err)
	}
	defer func() { _ = dslFile.Close() }()

	byteValue, _ := io.ReadAll(dslFile)

	var graphDef core.GraphDefinition
	if err := json.Unmarshal(byteValue, &graphDef); err != nil {
		panic(err)
	}

	// 2. Build Graph
	graph := &core.Graph{
		Nodes:     make(map[string]core.Node),
		Adjacency: make(map[string][]string),
		InDegree:  make(map[string]int),
	}

	for _, nodeCfg := range graphDef.Nodes {
		constructor, ok := nodes.Registry[nodeCfg.Type]
		if !ok {
			fmt.Printf("Unknown node type: %s\n", nodeCfg.Type)
			continue
		}
		node := constructor(nodeCfg)
		graph.Nodes[node.ID()] = node
		// Initialize degree
		graph.InDegree[node.ID()] = 0
	}

	for _, edge := range graphDef.Edges {
		graph.Adjacency[edge.Source] = append(graph.Adjacency[edge.Source], edge.Target)
		graph.InDegree[edge.Target]++
	}

	// 3. Initialize State
	pool := core.NewVariablePool()

	// 4. Run Engine
	// engine := core.NewEngine(graph, pool)
	// Use Pregel Engine instead
	engine := core.NewPregelEngine(graph, pool, 3)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	startTime := time.Now()
	if err := engine.Run(ctx); err != nil {
		fmt.Printf("Engine error: %v\n", err)
	}

	fmt.Printf("Workflow completed in %v\n", time.Since(startTime))
	
	// Print Final State (optional)
	// for id := range graph.Nodes {
	// 	vars := pool.GetNodeVars(id)
	// 	fmt.Printf("Node %s Vars: %v\n", id, vars)
	// }
}
