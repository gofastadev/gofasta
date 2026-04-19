package config

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONSchema_TopLevelEnvelope(t *testing.T) {
	schema := JSONSchema()

	assert.Equal(t, "http://json-schema.org/draft-07/schema#", schema["$schema"])
	assert.Equal(t, "AppConfig", schema["title"])
	assert.NotEmpty(t, schema["description"])
	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, false, schema["additionalProperties"])
}

func TestJSONSchema_RoundTripsThroughJSON(t *testing.T) {
	// The emitted schema must be serializable — no non-JSON types
	// (func, chan, reflect.Type) should accidentally sneak in.
	schema := JSONSchema()
	data, err := json.Marshal(schema)
	require.NoError(t, err, "schema must marshal to JSON without error")

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(data, &decoded),
		"schema must round-trip through JSON cleanly")
	assert.Equal(t, "AppConfig", decoded["title"])
}

func TestJSONSchema_EveryTopLevelSectionPresent(t *testing.T) {
	schema := JSONSchema()
	props, ok := schema["properties"].(map[string]any)
	require.True(t, ok, "properties must be a map")

	expected := []string{
		"server", "database", "graphql", "log", "email", "jobs", "auth",
		"rate_limit", "cache", "security", "storage", "queue", "websocket",
		"i18n", "feature_flag", "encryption", "session", "observability",
	}
	for _, name := range expected {
		assert.Contains(t, props, name, "top-level section %q missing from schema", name)
	}
}

func TestJSONSchema_NestedStructsHaveProperties(t *testing.T) {
	schema := JSONSchema()
	props := schema["properties"].(map[string]any)

	// database should be a nested object with its own properties.
	db, ok := props["database"].(map[string]any)
	require.True(t, ok, "database must be a nested object")
	assert.Equal(t, "object", db["type"])

	dbProps, ok := db["properties"].(map[string]any)
	require.True(t, ok, "database must have properties")
	assert.Contains(t, dbProps, "driver")
	assert.Contains(t, dbProps, "host")
	assert.Contains(t, dbProps, "port")
}

func TestJSONSchema_TimeDurationMapsToString(t *testing.T) {
	schema := JSONSchema()
	props := schema["properties"].(map[string]any)
	server := props["server"].(map[string]any)
	serverProps := server["properties"].(map[string]any)

	shutdown, ok := serverProps["shutdown_timeout"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", shutdown["type"])
	assert.Equal(t, "duration", shutdown["format"])
}

func TestJSONSchema_SliceMapsToArray(t *testing.T) {
	schema := JSONSchema()
	props := schema["properties"].(map[string]any)
	server := props["server"].(map[string]any)
	serverProps := server["properties"].(map[string]any)

	origins, ok := serverProps["allowed_origins"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "array", origins["type"])

	items := origins["items"].(map[string]any)
	assert.Equal(t, "string", items["type"])
}

func TestJSONSchema_SliceOfStructsHasItemsObject(t *testing.T) {
	schema := JSONSchema()
	props := schema["properties"].(map[string]any)

	// Jobs is []JobConfig — a slice whose items are objects.
	jobs, ok := props["jobs"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "array", jobs["type"])

	items := jobs["items"].(map[string]any)
	assert.Equal(t, "object", items["type"])

	// Each JobConfig has name / schedule / enabled.
	itemProps := items["properties"].(map[string]any)
	assert.Contains(t, itemProps, "name")
	assert.Contains(t, itemProps, "schedule")
	assert.Contains(t, itemProps, "enabled")
}

func TestJSONSchema_MapMapsToObjectWithAdditionalProperties(t *testing.T) {
	schema := JSONSchema()
	props := schema["properties"].(map[string]any)
	queue := props["queue"].(map[string]any)
	queueProps := queue["properties"].(map[string]any)

	queues, ok := queueProps["queues"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "object", queues["type"])
	// additionalProperties should describe the value type (int).
	addl, ok := queues["additionalProperties"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "integer", addl["type"])
}

func TestJSONSchema_PrimitiveTypes(t *testing.T) {
	schema := JSONSchema()
	props := schema["properties"].(map[string]any)
	auth := props["auth"].(map[string]any)
	authProps := auth["properties"].(map[string]any)

	// JWT secret — string
	secret := authProps["jwt_secret"].(map[string]any)
	assert.Equal(t, "string", secret["type"])

	// Database has integers
	db := props["database"].(map[string]any)
	dbProps := db["properties"].(map[string]any)
	maxIdle := dbProps["max_idle"].(map[string]any)
	assert.Equal(t, "integer", maxIdle["type"])

	// RateLimit has a bool
	rl := props["rate_limit"].(map[string]any)
	rlProps := rl["properties"].(map[string]any)
	enabled := rlProps["enabled"].(map[string]any)
	assert.Equal(t, "boolean", enabled["type"])
}

// --- helper tests -----------------------------------------------------------

func TestFieldName_FallsBackToLowercase(t *testing.T) {
	type X struct {
		FirstName string
	}
	got := fieldName(reflect.TypeOf(X{}).Field(0))
	assert.Equal(t, "firstname", got)
}

func TestFieldName_UsesKoanfTag(t *testing.T) {
	type X struct {
		FirstName string `koanf:"first_name"`
	}
	got := fieldName(reflect.TypeOf(X{}).Field(0))
	assert.Equal(t, "first_name", got)
}

func TestApplyValidateTag_Oneof(t *testing.T) {
	m := map[string]any{"type": "string"}
	applyValidateTag(m, "oneof=a b c")
	enum, ok := m["enum"].([]any)
	require.True(t, ok)
	assert.Equal(t, []any{"a", "b", "c"}, enum)
}

func TestApplyValidateTag_OneofWithOtherTokens(t *testing.T) {
	m := map[string]any{"type": "string"}
	applyValidateTag(m, "required,oneof=x y")
	enum, ok := m["enum"].([]any)
	require.True(t, ok)
	assert.Equal(t, []any{"x", "y"}, enum)
}

func TestIsRequired(t *testing.T) {
	assert.True(t, isRequired("required"))
	assert.True(t, isRequired("required,min=3"))
	assert.True(t, isRequired("min=3,required,max=10"))
	assert.False(t, isRequired(""))
	assert.False(t, isRequired("oneof=required other"),
		"must not match token inside oneof values")
}

// TestSchemaForType_SpecialCases — pointer unwrap + unknown kind fallback.
func TestSchemaForType_PointerUnwrap(t *testing.T) {
	ptr := reflect.TypeOf((*string)(nil))
	schema := schemaForType(ptr)
	assert.Equal(t, "string", schema["type"])
}

// TestJSONSchema_Deterministic — two successive calls produce the same
// JSON output. Guards against non-deterministic map iteration ordering
// in the required-fields list.
func TestJSONSchema_Deterministic(t *testing.T) {
	a, err := json.Marshal(JSONSchema())
	require.NoError(t, err)
	b, err := json.Marshal(JSONSchema())
	require.NoError(t, err)
	// We're comparing JSON bytes directly — Go's encoding/json sorts
	// map keys alphabetically, so as long as required[] is sorted
	// (which it is), output is stable.
	assert.Equal(t, string(a), string(b),
		"two calls to JSONSchema() must produce identical JSON")
}

// TestJSONSchema_HonorsTimeDurationTypeAssertion — prevents regression
// where time.Duration gets treated as int64 (its underlying type).
func TestJSONSchema_DurationIsNotInt(t *testing.T) {
	// AuthConfig.AccessTokenExpiry is a time.Duration — must not be
	// serialized as an integer.
	schema := JSONSchema()
	props := schema["properties"].(map[string]any)
	auth := props["auth"].(map[string]any)
	authProps := auth["properties"].(map[string]any)
	expiry := authProps["access_token_expiry"].(map[string]any)
	assert.Equal(t, "string", expiry["type"], "time.Duration must map to string, not integer")
	assert.Equal(t, "duration", expiry["format"])
	// Sanity: it really is a time.Duration in the source.
	f, ok := reflect.TypeOf(AppConfig{}).FieldByName("Auth")
	require.True(t, ok)
	assert.Equal(t, reflect.TypeOf(time.Duration(0)),
		f.Type.Field(1).Type,
		"test guards AuthConfig.AccessTokenExpiry remains a time.Duration")
}

// --- branch-coverage tests -------------------------------------------------
//
// AppConfig's real fields don't exercise every path in the reflection
// helpers (no floats, no interfaces, no unexported fields, no koanf:"-"
// skips, no desc tags, no validate:"required"). The tests below craft
// small local structs that specifically hit each untouched branch so
// schema.go reaches 100% coverage.

// TestSchemaForType_Float — the float32 / float64 branch maps to
// JSON Schema "number". No AppConfig field is currently float-typed.
func TestSchemaForType_Float(t *testing.T) {
	type Money struct {
		Price32 float32 `koanf:"price32"`
		Price64 float64 `koanf:"price64"`
	}
	schema := schemaForStruct(reflect.TypeOf(Money{}))
	props := schema["properties"].(map[string]any)
	assert.Equal(t, "number", props["price32"].(map[string]any)["type"])
	assert.Equal(t, "number", props["price64"].(map[string]any)["type"])
}

// TestSchemaForType_Fallback — kinds that don't correspond to any JSON
// type (channels, functions, raw interfaces) fall through the dispatch
// switch and return an empty schema map. Guards against accidental
// panics when a future AppConfig field adds such a type.
func TestSchemaForType_Fallback(t *testing.T) {
	// reflect.Type of a chan yields a reflect.Kind not handled in the
	// switch, exercising the default: branch.
	schema := schemaForType(reflect.TypeOf(make(chan int)))
	assert.Empty(t, schema)

	// func type hits the same default branch.
	schema = schemaForType(reflect.TypeOf(func() {}))
	assert.Empty(t, schema)

	// interface{} (any) also falls through.
	var i any
	schema = schemaForType(reflect.TypeOf(&i).Elem())
	assert.Empty(t, schema)
}

// TestSchemaForStruct_SkipsUnexportedFields — reflection on a struct
// with an unexported field should not include that field in the
// generated schema properties.
func TestSchemaForStruct_SkipsUnexportedFields(t *testing.T) {
	type WithUnexported struct {
		Exported   string `koanf:"exported"`
		unexported string //nolint:unused // field intentionally unexported for this test
	}
	schema := schemaForStruct(reflect.TypeOf(WithUnexported{}))
	props := schema["properties"].(map[string]any)
	assert.Contains(t, props, "exported")
	assert.NotContains(t, props, "unexported")
}

// TestSchemaForStruct_SkipsDashKoanfTag — a koanf:"-" tag marks the
// field as excluded from loading; the schema must honor the same
// convention and omit it from the properties.
func TestSchemaForStruct_SkipsDashKoanfTag(t *testing.T) {
	type WithDash struct {
		Visible string `koanf:"visible"`
		Hidden  string `koanf:"-"`
	}
	schema := schemaForStruct(reflect.TypeOf(WithDash{}))
	props := schema["properties"].(map[string]any)
	assert.Contains(t, props, "visible")
	assert.NotContains(t, props, "-")
	assert.NotContains(t, props, "hidden")
}

// TestSchemaForStruct_HonorsDescTag — a `desc:"..."` tag is copied
// verbatim into the field's "description" property.
func TestSchemaForStruct_HonorsDescTag(t *testing.T) {
	type WithDesc struct {
		Name string `koanf:"name" desc:"Human-readable service name shown in the logs."`
	}
	schema := schemaForStruct(reflect.TypeOf(WithDesc{}))
	props := schema["properties"].(map[string]any)
	name := props["name"].(map[string]any)
	assert.Equal(t,
		"Human-readable service name shown in the logs.",
		name["description"],
	)
}

// TestSchemaForStruct_MarksRequiredFields — a field tagged with
// validate:"required" lands in the parent object's "required" list.
// No AppConfig field is currently tagged required, so this is
// specifically the path that the live schema doesn't hit.
func TestSchemaForStruct_MarksRequiredFields(t *testing.T) {
	type WithRequired struct {
		Key   string `koanf:"key"    validate:"required"`
		Value string `koanf:"value"`
		Other string `koanf:"other"  validate:"required,min=1"`
	}
	schema := schemaForStruct(reflect.TypeOf(WithRequired{}))
	req, ok := schema["required"].([]string)
	require.True(t, ok, "required list must be present")
	// Sorted deterministically.
	assert.Equal(t, []string{"key", "other"}, req)
}

// TestFieldName_HandlesCommaSeparatedKoanfTag — koanf tags can carry
// options after a comma (e.g. `koanf:"name,omitempty"`). fieldName
// returns only the first token.
func TestFieldName_HandlesCommaSeparatedKoanfTag(t *testing.T) {
	type X struct {
		FirstName string `koanf:"first_name,omitempty"`
	}
	got := fieldName(reflect.TypeOf(X{}).Field(0))
	assert.Equal(t, "first_name", got)
}
