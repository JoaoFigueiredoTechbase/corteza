package Yeastar

import (
	"sync"
)

// ConfigManager handles configuration lifecycle
type ConfigManager struct {
	config *Config
	mu     sync.RWMutex
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{}
}

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

func (cm *ConfigManager) SetConfig(cfg *Config) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.config = cfg
}
