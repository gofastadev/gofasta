package validation

import (
	"github.com/healtronlabs/gofasta/packages/core"
)

// ValidationModule provides validation capabilities
type ValidationModule struct {
	*core.BaseModule
}

// NewValidationModule creates a new validation module
func NewValidationModule() *ValidationModule {
	module := &ValidationModule{
		BaseModule: core.NewBaseModule(),
	}
	
	// Add validation pipe as a provider
	module.AddProvider(NewValidationPipe())
	
	return module
}

// Configure configures the validation module
func (m *ValidationModule) Configure(container *core.DIContainer) error {
	// Register validation pipe
	validationPipe := NewValidationPipe()
	return container.RegisterProvider(validationPipe)
}