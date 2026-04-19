package config

import (
	"reflect"
	"sort"
	"strings"
	"time"
)

// JSONSchema returns a JSON Schema (Draft 7) document describing the
// AppConfig type. Emitted as a map so callers can marshal it with
// encoding/json without a dependency on a third-party JSON Schema
// library — the output is a plain data structure that any JSON
// encoder will serialize correctly.
//
// The schema is derived by reflecting over AppConfig at runtime, so it
// always matches the type definitions in this package — there is no
// second source of truth to keep in sync. Adding a new config field
// is immediately reflected the next time JSONSchema() runs.
//
// Intended consumers:
//
//   - AI coding agents that want to validate or generate config.yaml
//     without guessing the schema from stale training data.
//   - Editor extensions (VS Code YAML, JetBrains) that consume JSON
//     Schema for autocomplete, type errors, and enum suggestions.
//   - CI pipelines that validate config.yaml before deploy.
//
// Field-tag conventions honored:
//
//   - `koanf:"name"`  — sets the JSON property name (falls back to
//     lowercase field name if missing).
//   - `validate:"required"`   — the field is marked as required in the
//     parent object's "required" list.
//   - `validate:"oneof=a b c"` — the field's schema includes an enum
//     of those values.
//   - `desc:"..."` — the field's schema includes this text as its
//     "description". No struct currently uses this tag, but the
//     reflector honors it so future fields can add descriptions
//     without a schema function rewrite.
func JSONSchema() map[string]any {
	t := reflect.TypeOf(AppConfig{})
	schema := schemaForType(t)
	// Merge the top-level envelope fields that make this a complete
	// JSON Schema document rather than a bare subschema.
	schema["$schema"] = "http://json-schema.org/draft-07/schema#"
	schema["title"] = "AppConfig"
	schema["description"] = "Gofasta application configuration. Loaded from config.yaml and overridden by environment variables prefixed with the project name (e.g. MYAPP_DATABASE_HOST)."
	return schema
}

// schemaForType dispatches by kind, returning a JSON Schema fragment
// for the Go type t. Pointer types unwrap to their element type;
// time.Duration has a special-case mapping to "string" with format
// "duration" because it is idiomatically serialized as a human
// string like "30s" or "5m".
func schemaForType(t reflect.Type) map[string]any {
	// Unwrap pointers — a *Config means "optional Config", but the
	// JSON shape of the value itself is still the struct's shape.
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// time.Duration is technically an int64 but almost always written
	// as a human-readable duration string ("30s", "5m"). Emit a string
	// schema with format "duration" so editors suggest that shape.
	if t == reflect.TypeOf(time.Duration(0)) {
		return map[string]any{
			"type":        "string",
			"format":      "duration",
			"description": "A Go time.Duration expressed as a string (e.g. \"30s\", \"5m\", \"24h\").",
		}
	}

	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice, reflect.Array:
		return map[string]any{
			"type":  "array",
			"items": schemaForType(t.Elem()),
		}
	case reflect.Map:
		// Emit as an object whose additionalProperties are the value
		// type's schema. Covers map[string]int (queue concurrency) etc.
		return map[string]any{
			"type":                 "object",
			"additionalProperties": schemaForType(t.Elem()),
		}
	case reflect.Struct:
		return schemaForStruct(t)
	default:
		// Fall back to "any" for interfaces / channels / funcs — none
		// should appear in AppConfig, but the reflector stays total
		// so adding a future field never panics.
		return map[string]any{}
	}
}

// schemaForStruct walks a struct's fields and emits an object schema
// with properties + required list + additionalProperties: false (so
// editors flag unknown keys as typos, not silently-ignored fields).
func schemaForStruct(t reflect.Type) map[string]any {
	properties := map[string]any{}
	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// Skip unexported fields — they can't be populated from YAML
		// anyway and shouldn't appear in the schema.
		if !field.IsExported() {
			continue
		}

		name := fieldName(field)
		if name == "" || name == "-" {
			continue
		}

		fieldSchema := schemaForType(field.Type)
		applyValidateTag(fieldSchema, field.Tag.Get("validate"))
		if desc := field.Tag.Get("desc"); desc != "" {
			fieldSchema["description"] = desc
		}

		properties[name] = fieldSchema

		// A field is required when its struct tag marks it so. We do
		// NOT mark fields required just because they're non-pointer
		// — most config defaults come from LoadConfig() and should
		// be optional in YAML.
		if isRequired(field.Tag.Get("validate")) {
			required = append(required, name)
		}
	}

	// Deterministic ordering so the schema is reproducible across runs.
	// Map iteration in Go is randomized — without this, two runs would
	// produce byte-different JSON and confuse content-hash caches.
	sort.Strings(required)

	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// fieldName extracts the koanf tag's name, falling back to the lowercase
// field name. Matches how koanf loads values at runtime so the schema
// and the loader agree on key names.
func fieldName(field reflect.StructField) string {
	tag := field.Tag.Get("koanf")
	if tag == "" {
		return strings.ToLower(field.Name)
	}
	// koanf tags are simple — just "name" with no comma-separated
	// options. Use the first comma-delimited token to be safe.
	if i := strings.Index(tag, ","); i >= 0 {
		return tag[:i]
	}
	return tag
}

// applyValidateTag inspects a go-playground/validator tag and adds any
// constraints we can faithfully translate to JSON Schema — currently
// just "oneof=...", which maps to "enum".
func applyValidateTag(schema map[string]any, tag string) {
	if tag == "" {
		return
	}
	for _, part := range strings.Split(tag, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "oneof=") {
			values := strings.Fields(strings.TrimPrefix(part, "oneof="))
			anyValues := make([]any, len(values))
			for i, v := range values {
				anyValues[i] = v
			}
			schema["enum"] = anyValues
		}
	}
}

// isRequired reports whether the validator tag includes the "required"
// token as its own comma-separated segment. A literal substring match
// would be wrong — a tag like "oneof=required other" would falsely
// claim the field is required because the word "required" appears as
// an enum value, not as a validator token.
func isRequired(tag string) bool {
	for _, part := range strings.Split(tag, ",") {
		if strings.TrimSpace(part) == "required" {
			return true
		}
	}
	return false
}
