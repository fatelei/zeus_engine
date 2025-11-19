package core

import (
	"sync"
)

// VariablePool manages the state of the workflow
type VariablePool struct {
	mu   sync.RWMutex
	data map[string]map[string]interface{}
}

func NewVariablePool() *VariablePool {
	return &VariablePool{
		data: make(map[string]map[string]interface{}),
	}
}

// Set stores a value for a specific node and variable name
func (p *VariablePool) Set(nodeID, key string, value interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.data[nodeID]; !exists {
		p.data[nodeID] = make(map[string]interface{})
	}
	p.data[nodeID][key] = value
}

// Get retrieves a value
func (p *VariablePool) Get(nodeID, key string) (interface{}, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if nodeVars, ok := p.data[nodeID]; ok {
		val, ok := nodeVars[key]
		return val, ok
	}
	return nil, false
}

// GetAll returns all variables for a node
func (p *VariablePool) GetNodeVars(nodeID string) map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]interface{})
	if nodeVars, ok := p.data[nodeID]; ok {
		for k, v := range nodeVars {
			result[k] = v
		}
	}
	return result
}
