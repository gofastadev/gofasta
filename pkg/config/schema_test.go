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
