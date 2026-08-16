package auditlog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"gorm.io/gorm"

	"github.com/gofastadev/gofasta/pkg/httpcontext"
)

// The middleware reads the parsed operation, not the request body, so the tests
// drive gqlgen's real executor: the AST it inspects is the one gqlparser built
// and gqlgen validated, variable coercion included, dispatched through the same
// AroundOperations chain a server uses. A hand-built OperationContext would let
// a test pass on an AST the executor would never produce.
const auditTestSchema = `
type Query {
	ping: String!
}

input CourseInput { id: ID, title: String }
input UpperInput  { ID: ID }
input CountInput  { id: Int }
input TitleInput  { title: String }

enum Status { ACTIVE ARCHIVED }

type Mutation {
	createCourse(input: CourseInput!): String!
	registerDevice(input: UpperInput!): String!
	addEnrollment(input: CountInput!): String!
	assignBadge(input: TitleInput!): String!

	updateCourse(id: ID): String!
	deleteCourse(courseID: ID!): String!
	archiveLesson(Id: ID!): String!
	publishRelease(releaseId: ID!): String!

	setStatus(id: Status!): String!
	editPosition(id: Int!): String!
	toggleSync(syncID: Boolean!): String!

	renameCourse(title: String!): String!
	syncEverything: String!
}
`

// newAuditedExecutor returns a gqlgen executor with the audit middleware
// installed, and the database it logs into.
func newAuditedExecutor(t *testing.T) (*executor.Executor, *gorm.DB) {
	t.Helper()

	db := newAuditDB(t)
	service := NewAuditService(db, "solago")

	schema := gqlparser.MustLoadSchema(&ast.Source{Name: "audit_test", Input: auditTestSchema})

	exec := executor.New(&graphql.ExecutableSchemaMock{
		SchemaFunc: func() *ast.Schema { return schema },
		ComplexityFunc: func(_ context.Context, _, _ string, childComplexity int, _ map[string]any) (int, bool) {
			return childComplexity, true
		},
		ExecFunc: func(_ context.Context) graphql.ResponseHandler {
			return graphql.OneShot(&graphql.Response{Data: []byte(`{"ok":"done"}`)})
		},
	})
	exec.AroundOperations(GraphQLMiddleware(service))

	return exec, db
}

// run parses, validates and dispatches an operation, with the request in the
// context the way httpcontext.Middleware would put it — that is where the
// middleware finds the client address it records.
func run(t *testing.T, exec *executor.Executor, query string, variables map[string]any) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(query))
	req.RemoteAddr = "203.0.113.7:5555"
	req.Header.Set("User-Agent", "jedi/2.0")
	ctx := httpcontext.WithRequest(context.Background(), req)
	// What every gqlgen transport does before handing the request to the
	// executor; CreateOperationContext reads the start time it stamps.
	ctx = graphql.StartOperationTrace(ctx)

	opCtx, errs := exec.CreateOperationContext(ctx, &graphql.RawParams{
		Query:     query,
		Variables: variables,
	})
	if len(errs) > 0 {
		t.Fatalf("the operation did not parse or validate: %v", errs)
	}

	responses, ctx := exec.DispatchOperation(ctx, opCtx)
	resp := responses(ctx)
	if resp != nil && len(resp.Errors) > 0 {
		t.Fatalf("execution returned errors: %v", resp.Errors)
	}
}

// loggedEntries waits for the expected number of rows and returns them. The
// middleware logs after the operation resolves, on the asynchronous path.
func loggedEntries(t *testing.T, db *gorm.DB, want int) []Entry {
	t.Helper()
	eventually(t, "the mutation to be audited", func() bool {
		var n int64
		db.Model(&Entry{}).Count(&n)
		return n >= int64(want)
	})
	entries := entriesIn(t, db)
	if len(entries) != want {
		t.Fatalf("got %d audit rows, want %d", len(entries), want)
	}
	return entries
}

func TestGraphQLMiddleware_LogsAMutationWithItsResourceID(t *testing.T) {
	exec, db := newAuditedExecutor(t)

	run(t, exec, `mutation { updateCourse(id: "course-1") }`, nil)

	got := loggedEntries(t, db, 1)[0]

	if got.EventType != "COURSE_UPDATED" {
		t.Errorf("EventType = %q, want COURSE_UPDATED", got.EventType)
	}
	if got.ResourceType != "COURSE" || got.ResourceID != "course-1" {
		t.Errorf("resource = %q/%q, want COURSE/course-1", got.ResourceType, got.ResourceID)
	}
	if got.IPAddress != "203.0.113.7:5555" || got.UserAgent != "jedi/2.0" {
		t.Errorf("ip/ua = %q/%q — the request never reached the audit row",
			got.IPAddress, got.UserAgent)
	}

	var details map[string]any
	if err := json.Unmarshal(got.Details, &details); err != nil {
		t.Fatalf("details are not JSON: %v", err)
	}
	if details["mutation"] != "updateCourse" {
		t.Errorf("details.mutation = %v, want the field name", details["mutation"])
	}
	if details["resourceID"] != "course-1" {
		t.Errorf("details.resourceID = %v", details["resourceID"])
	}
}

// A query is not a change, and logging one per read would bury the writes
// anybody is actually looking for.
func TestGraphQLMiddleware_IgnoresQueries(t *testing.T) {
	exec, db := newAuditedExecutor(t)

	run(t, exec, `query { ping }`, nil)

	var n int64
	if err := db.Model(&Entry{}).Count(&n).Error; err != nil {
		t.Fatalf("counting: %v", err)
	}
	if n != 0 {
		t.Errorf("%d rows logged for a query, want 0", n)
	}
}

func TestGraphQLMiddleware_ResourceIDArgumentShapes(t *testing.T) {
	cases := []struct {
		name         string
		query        string
		variables    map[string]any
		wantEvent    string
		wantResource string
		wantID       string
	}{
		{
			name:         "direct id literal",
			query:        `mutation { deleteCourse(courseID: "c-9") }`,
			wantEvent:    "COURSE_DELETED",
			wantResource: "COURSE",
			wantID:       "c-9",
		},
		{
			name:         "capitalised Id argument",
			query:        `mutation { archiveLesson(Id: "l-3") }`,
			wantEvent:    "LESSON_DELETED",
			wantResource: "LESSON",
			wantID:       "l-3",
		},
		{
			name:         "argument ending in Id",
			query:        `mutation { publishRelease(releaseId: "r-7") }`,
			wantEvent:    "RELEASE_UPDATED",
			wantResource: "RELEASE",
			wantID:       "r-7",
		},
		{
			name:         "id passed as a variable",
			query:        `mutation Update($id: ID) { updateCourse(id: $id) }`,
			variables:    map[string]any{"id": "course-var"},
			wantEvent:    "COURSE_UPDATED",
			wantResource: "COURSE",
			wantID:       "course-var",
		},
		{
			name:         "inline input object",
			query:        `mutation { createCourse(input: {id: "c-inline", title: "Intro"}) }`,
			wantEvent:    "COURSE_CREATED",
			wantResource: "COURSE",
			wantID:       "c-inline",
		},
		{
			name:         "input object as a variable",
			query:        `mutation Create($input: CourseInput!) { createCourse(input: $input) }`,
			variables:    map[string]any{"input": map[string]any{"id": "c-var"}},
			wantEvent:    "COURSE_CREATED",
			wantResource: "COURSE",
			wantID:       "c-var",
		},
		{
			name:         "uppercase ID inside an input variable",
			query:        `mutation Register($input: UpperInput!) { registerDevice(input: $input) }`,
			variables:    map[string]any{"input": map[string]any{"ID": "d-var"}},
			wantEvent:    "DEVICE_CREATED",
			wantResource: "DEVICE",
			wantID:       "d-var",
		},
		{
			name:         "uppercase ID inline",
			query:        `mutation { registerDevice(input: {ID: "d-inline"}) }`,
			wantEvent:    "DEVICE_CREATED",
			wantResource: "DEVICE",
			wantID:       "d-inline",
		},
		{
			name:         "enum-valued id argument",
			query:        `mutation { setStatus(id: ACTIVE) }`,
			wantEvent:    "STATUS_UPDATED",
			wantResource: "STATUS",
			wantID:       "ACTIVE",
		},
		{
			name:         "integer-valued id argument",
			query:        `mutation { editPosition(id: 12) }`,
			wantEvent:    "POSITION_UPDATED",
			wantResource: "POSITION",
			wantID:       "12",
		},

		// Everything below has no id the middleware can find. The event is
		// still logged: knowing a mutation ran matters more than knowing what
		// it touched, and a missing id must not suppress the record.
		{
			name:         "no arguments at all",
			query:        `mutation { syncEverything }`,
			wantEvent:    "GRAPHQL_MUTATION_SYNC_EVERYTHING",
			wantResource: "SYNC_EVERYTHING",
			wantID:       "",
		},
		{
			name:         "no argument resembling an id",
			query:        `mutation { renameCourse(title: "New") }`,
			wantEvent:    "GRAPHQL_MUTATION_RENAME_COURSE",
			wantResource: "RENAME_COURSE",
			wantID:       "",
		},
		{
			name:         "input object carrying no id",
			query:        `mutation { assignBadge(input: {title: "gold"}) }`,
			wantEvent:    "BADGE_CREATED",
			wantResource: "BADGE",
			wantID:       "",
		},
		{
			name:         "id inside the input is not a string",
			query:        `mutation Add($input: CountInput!) { addEnrollment(input: $input) }`,
			variables:    map[string]any{"input": map[string]any{"id": 5}},
			wantEvent:    "ENROLLMENT_CREATED",
			wantResource: "ENROLLMENT",
			wantID:       "",
		},
		{
			name:         "id variable is not a string",
			query:        `mutation Toggle($flag: Boolean!) { toggleSync(syncID: $flag) }`,
			variables:    map[string]any{"flag": true},
			wantEvent:    "SYNC_UPDATED",
			wantResource: "SYNC",
			wantID:       "",
		},
		{
			name:         "a value kind that is not an id",
			query:        `mutation { toggleSync(syncID: true) }`,
			wantEvent:    "SYNC_UPDATED",
			wantResource: "SYNC",
			wantID:       "",
		},
		{
			name:         "omitted nullable id variable",
			query:        `mutation Update($id: ID) { updateCourse(id: $id) }`,
			variables:    map[string]any{},
			wantEvent:    "COURSE_UPDATED",
			wantResource: "COURSE",
			wantID:       "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			exec, db := newAuditedExecutor(t)

			run(t, exec, tc.query, tc.variables)

			got := loggedEntries(t, db, 1)[0]
			if got.EventType != tc.wantEvent {
				t.Errorf("EventType = %q, want %q", got.EventType, tc.wantEvent)
			}
			if got.ResourceType != tc.wantResource {
				t.Errorf("ResourceType = %q, want %q", got.ResourceType, tc.wantResource)
			}
			if got.ResourceID != tc.wantID {
				t.Errorf("ResourceID = %q, want %q", got.ResourceID, tc.wantID)
			}

			var details map[string]any
			if err := json.Unmarshal(got.Details, &details); err != nil {
				t.Fatalf("details are not JSON: %v", err)
			}
			if _, present := details["resourceID"]; present != (tc.wantID != "") {
				t.Errorf("details = %v; resourceID should be present only when one was found", details)
			}
		})
	}
}

func TestGraphQLMiddleware_LogsEveryFieldInOneOperation(t *testing.T) {
	// One request may carry several mutations, and each is a separate change.
	// Logging only the first would leave the rest of them unrecorded.
	exec, db := newAuditedExecutor(t)

	run(t, exec, `mutation {
		createCourse(input: {id: "c-1"})
		deleteCourse(courseID: "c-2")
	}`, nil)

	entries := loggedEntries(t, db, 2)

	byResourceID := map[string]Entry{}
	for _, e := range entries {
		byResourceID[e.ResourceID] = e
	}
	if got := byResourceID["c-1"].EventType; got != "COURSE_CREATED" {
		t.Errorf("c-1 logged as %q, want COURSE_CREATED", got)
	}
	if got := byResourceID["c-2"].EventType; got != "COURSE_DELETED" {
		t.Errorf("c-2 logged as %q, want COURSE_DELETED", got)
	}
}

func TestGraphQLMiddleware_SkipsSelectionsThatAreNotFields(t *testing.T) {
	// A fragment spread at the root is a selection with no field name to
	// classify. It has to be stepped over rather than dereferenced.
	exec, db := newAuditedExecutor(t)

	run(t, exec, `mutation { ...Everything }
		fragment Everything on Mutation { syncEverything }`, nil)

	var n int64
	if err := db.Model(&Entry{}).Count(&n).Error; err != nil {
		t.Fatalf("counting: %v", err)
	}
	if n != 0 {
		t.Errorf("%d rows logged, want 0 — a spread carries no field to name", n)
	}
}

// ---------- the defensive paths the parser cannot produce ----------
//
// gqlparser never hands the middleware a nil *ast.Value, so these guards are
// unreachable through the server above. They are still the difference between
// a nil map argument and a panic inside a request, which is why they are
// exercised directly.

func TestResolveValueString_NilValue(t *testing.T) {
	if got := resolveValueString(nil, nil); got != "" {
		t.Errorf("resolveValueString(nil) = %q, want empty", got)
	}
}

func TestExtractIDFromInputValue_NilValue(t *testing.T) {
	if got := extractIDFromInputValue(nil, nil); got != "" {
		t.Errorf("extractIDFromInputValue(nil) = %q, want empty", got)
	}
}

func TestExtractIDFromInputValue_VariableThatIsNotAnObject(t *testing.T) {
	// A variable declared as an input object but bound to a scalar: the type
	// assertion has to fail into "no id" rather than panic.
	value := &ast.Value{Kind: ast.Variable, Raw: "input"}

	if got := extractIDFromInputValue(value, map[string]any{"input": "not-an-object"}); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	if got := extractIDFromInputValue(value, map[string]any{}); got != "" {
		t.Errorf("missing variable gave %q, want empty", got)
	}
}

func TestExtractResourceID_NoArguments(t *testing.T) {
	if got := extractResourceID(nil, nil); got != "" {
		t.Errorf("extractResourceID(nil) = %q, want empty", got)
	}
}
