package Yeastar

import (
	"context"
	"sync"
)

// ConfigManager handles configuration lifecycle
type ConfigManager struct {
	config    *Config
	mu        sync.RWMutex
	readyChan chan struct{}
	onceReady sync.Once
	isReady   bool
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		readyChan: make(chan struct{}),
	}
}

// SetConfig sets the configuration and marks it as ready
func (cm *ConfigManager) SetConfig(cfg *Config) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.config = cfg
	cm.onceReady.Do(func() {
		cm.isReady = true
		close(cm.readyChan)
	})
}

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

// WaitForConfig waits for configuration to be available
func (cm *ConfigManager) WaitForConfig(ctx context.Context) (*Config, error) {
	select {
	case <-cm.readyChan:
		return cm.GetConfig(), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// IsReady returns true if configuration is ready
func (cm *ConfigManager) IsReady() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.isReady
}
