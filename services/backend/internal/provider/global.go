package provider

import "sync"

var (
	globalRegistry *Registry
	once           sync.Once
)

// InitGlobalRegistry initializes the global provider registry (should be called once at startup)
func InitGlobalRegistry() *Registry {
	once.Do(func() {
		globalRegistry = NewRegistry()
		globalRegistry.Register(NewPhoneProvider())
		globalRegistry.Register(NewGatewayHudhudProvider())
		globalRegistry.Register(NewGatewayPrivateProvider())
	})
	return globalRegistry
}

// GetGlobalRegistry returns the global provider registry instance
func GetGlobalRegistry() *Registry {
	if globalRegistry == nil {
		return InitGlobalRegistry()
	}
	return globalRegistry
}
