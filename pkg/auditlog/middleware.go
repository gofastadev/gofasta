package auditlog

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
)

// GraphQLMiddleware returns a gqlgen AroundOperations middleware that
// automatically logs all GraphQL mutations. It parses the operation's
// selection set to log one entry per mutation field with resource type,
// action, and resource ID extracted from arguments.
func GraphQLMiddleware(auditService *Service) graphql.OperationMiddleware {
	return func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		oc := graphql.GetOperationContext(ctx)

		// Only log mutations
		if oc.Operation == nil || oc.Operation.Operation != ast.Mutation {
			return next(ctx)
		}

		// Extract mutation fields from the AST before execution
		type mutationInfo struct {
			fieldName  string
			resource   string
			action     string
			resourceID string
		}

		var mutations []mutationInfo
		for _, sel := range oc.Operation.SelectionSet {
			field, ok := sel.(*ast.Field)
			if !ok {
				continue
			}
			resource, action := ParseMutationName(field.Name)
			resourceID := extractResourceID(field.Arguments, oc.Variables)

			mutations = append(mutations, mutationInfo{
				fieldName:  field.Name,
				resource:   resource,
				action:     action,
				resourceID: resourceID,
			})
		}

		resp := next(ctx)

		// Log one entry per mutation field
		for _, m := range mutations {
			eventType := EventName(m.resource, m.action)
			if m.action == "" {
				eventType = "GRAPHQL_MUTATION_" + m.resource
			}

			details := map[string]interface{}{
				"mutation": m.fieldName,
			}
			if m.resourceID != "" {
				details["resourceID"] = m.resourceID
			}

			auditService.LogFromContextWithResource(ctx, eventType, m.resource, m.resourceID, details)
		}

		return resp
	}
}

// extractResourceID tries to find a resource ID from the mutation field's arguments.
// It checks for common patterns: "id", "ID", and "input" object with an "id" field.
func extractResourceID(args ast.ArgumentList, variables map[string]interface{}) string {
	// Direct id argument
	for _, arg := range args {
		name := arg.Name
		if name == "id" || name == "ID" || name == "Id" {
			return resolveValueString(arg.Value, variables)
		}
	}

	// Check for arguments ending in "ID" or "Id" (e.g., courseID, userId)
	for _, arg := range args {
		name := arg.Name
		if len(name) > 2 && (name[len(name)-2:] == "ID" || name[len(name)-2:] == "Id") {
			return resolveValueString(arg.Value, variables)
		}
	}

	// Check "input" argument for an "id" field
	for _, arg := range args {
		if arg.Name == "input" {
			return extractIDFromInputValue(arg.Value, variables)
		}
	}

	return ""
}

// resolveValueString resolves an AST value to a string, handling variable references.
func resolveValueString(v *ast.Value, variables map[string]interface{}) string {
	if v == nil {
		return ""
	}

	// If it's a variable reference, look it up
	if v.Kind == ast.Variable {
		val, ok := variables[v.Raw]
		if !ok {
			return ""
		}
		if s, ok := val.(string); ok {
			return s
		}
		return ""
	}

	// Direct string/enum/int value
	if v.Kind == ast.StringValue || v.Kind == ast.EnumValue || v.Kind == ast.IntValue {
		return v.Raw
	}

	return ""
}

// extractIDFromInputValue looks for an "id" field inside an input object value.
func extractIDFromInputValue(v *ast.Value, variables map[string]interface{}) string {
	if v == nil {
		return ""
	}

	// If the input is a variable, resolve it from the variables map
	if v.Kind == ast.Variable {
		val, ok := variables[v.Raw]
		if !ok {
			return ""
		}
		if m, ok := val.(map[string]interface{}); ok {
			if id, ok := m["id"].(string); ok {
				return id
			}
			if id, ok := m["ID"].(string); ok {
				return id
			}
		}
		return ""
	}

	// Inline object value
	if v.Kind == ast.ObjectValue {
		for _, child := range v.Children {
			if child.Name == "id" || child.Name == "ID" {
				return resolveValueString(child.Value, variables)
			}
		}
	}

	return ""
}
