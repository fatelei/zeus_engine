package core

import (
	"sync"
	"testing"
)

func TestVariablePool(t *testing.T) {
	pool := NewVariablePool()

	// Test Set and Get
	pool.Set("node1", "key1", "value1")
	val, ok := pool.Get("node1", "key1")
	if !ok {
		t.Error("Expected key1 to exist for node1")
	}
	if val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}

	// Test Get non-existent key
	_, ok = pool.Get("node1", "key2")
	if ok {
		t.Error("Expected key2 to not exist for node1")
	}

	// Test Get non-existent node
	_, ok = pool.Get("node2", "key1")
	if ok {
		t.Error("Expected node2 to not exist")
	}

	// Test GetNodeVars
	pool.Set("node1", "key2", "value2")
	vars := pool.GetNodeVars("node1")
	if len(vars) != 2 {
		t.Errorf("Expected 2 variables for node1, got %d", len(vars))
	}
	if vars["key1"] != "value1" || vars["key2"] != "value2" {
		t.Error("GetNodeVars returned incorrect values")
	}
}

func TestVariablePoolConcurrency(t *testing.T) {
	pool := NewVariablePool()
	var wg sync.WaitGroup

	// Concurrent Writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			pool.Set("node1", "key", val)
		}(i)
	}

	// Concurrent Reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pool.Get("node1", "key")
		}()
	}

	wg.Wait()
}
