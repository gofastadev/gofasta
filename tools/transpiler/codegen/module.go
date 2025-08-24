package codegen

import "fmt"

// generateModuleDeclaration generates Go code for a module
func (g *CodeGenerator) generateModuleDeclaration(module *ModuleDeclaration) error {
	// Generate module struct
	g.writeLine(fmt.Sprintf("type %s struct {", module.Name))
	g.indent()
	g.writeLine("core.BaseModule")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	// Generate Configure method
	g.generateModuleConfigureMethod(module)

	return nil
}

// generateModuleConfigureMethod generates the Configure method for a module
func (g *CodeGenerator) generateModuleConfigureMethod(module *ModuleDeclaration) {
	g.writeLine(fmt.Sprintf("func (m *%s) Configure(container *core.DIContainer) error {", module.Name))
	g.indent()

	// Get module configuration from decorators
	config := g.getModuleConfig(module)

	// Register controllers
	if len(config.Controllers) > 0 {
		g.writeLine("// Register controllers")
		for _, controller := range config.Controllers {
			g.writeLine(fmt.Sprintf("// TODO: Register %s controller", controller))
		}
		g.writeLine("")
	}

	// Register providers
	if len(config.Providers) > 0 {
		g.writeLine("// Register providers")
		for _, provider := range config.Providers {
			g.writeLine(fmt.Sprintf("// TODO: Register %s provider", provider))
		}
		g.writeLine("")
	}

	// Import other modules
	if len(config.Imports) > 0 {
		g.writeLine("// Import modules")
		for _, importModule := range config.Imports {
			g.writeLine(fmt.Sprintf("// TODO: Import %s module", importModule))
		}
		g.writeLine("")
	}

	g.writeLine("return nil")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// getModuleConfig extracts module configuration from decorators
func (g *CodeGenerator) getModuleConfig(module *ModuleDeclaration) ModuleConfig {
	config := ModuleConfig{
		Controllers: []string{},
		Providers:   []string{},
		Imports:     []string{},
		Exports:     []string{},
	}

	// Look for @Module decorator
	moduleDecorator := g.getDecorator(module.Decorators, "Module")
	if moduleDecorator == nil {
		return config
	}

	// Extract configuration from decorator arguments
	for _, arg := range moduleDecorator.Args {
		if objValue, ok := arg.Value.(map[string]interface{}); ok {
			// Extract controllers
			if controllers, exists := objValue["controllers"]; exists {
				if controllerList, ok := controllers.([]interface{}); ok {
					for _, ctrl := range controllerList {
						if ctrlStr, ok := ctrl.(string); ok {
							config.Controllers = append(config.Controllers, ctrlStr)
						}
					}
				}
			}

			// Extract providers
			if providers, exists := objValue["providers"]; exists {
				if providerList, ok := providers.([]interface{}); ok {
					for _, prov := range providerList {
						if provStr, ok := prov.(string); ok {
							config.Providers = append(config.Providers, provStr)
						}
					}
				}
			}

			// Extract imports
			if imports, exists := objValue["imports"]; exists {
				if importList, ok := imports.([]interface{}); ok {
					for _, imp := range importList {
						if impStr, ok := imp.(string); ok {
							config.Imports = append(config.Imports, impStr)
						}
					}
				}
			}

			// Extract exports
			if exports, exists := objValue["exports"]; exists {
				if exportList, ok := exports.([]interface{}); ok {
					for _, exp := range exportList {
						if expStr, ok := exp.(string); ok {
							config.Exports = append(config.Exports, expStr)
						}
					}
				}
			}
		}
	}

	return config
}