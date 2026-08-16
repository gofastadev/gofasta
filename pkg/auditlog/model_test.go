package auditlog

import (
	"encoding/json"
	"testing"
	"time"
)

// The Go type was renamed when this package moved into the framework; the table
// was not, because rows already live there. A default-derived name
// ("entries") would read and write an empty table on every existing
// deployment, and nothing about that failure looks like a failure.
func TestEntry_TableNameStaysAuditLogs(t *testing.T) {
	if got := (Entry{}).TableName(); got != "audit_logs" {
		t.Errorf("TableName() = %q, want audit_logs", got)
	}
}

// SubjectID is a *string because an unattributed action is a real thing: a
// scheduled job, a system event, a request that failed before authentication.
// A non-pointer would record every one of them as the zero-value subject and
// make "nobody" indistinguishable from "the empty user".
func TestEntry_SubjectIDIsNullable(t *testing.T) {
	var entry Entry
	if entry.SubjectID != nil {
		t.Fatalf("zero Entry.SubjectID = %v, want nil", entry.SubjectID)
	}

	subject := "11111111-2222-3333-4444-555555555555"
	entry.SubjectID = &subject
	if *entry.SubjectID != subject {
		t.Errorf("SubjectID = %q, want %q", *entry.SubjectID, subject)
	}
}

// Details is json.RawMessage rather than a map so the stored bytes survive a
// round trip unchanged — an audit record that re-serializes its own payload has
// already stopped being a record of what happened.
func TestEntry_DetailsRoundTripsVerbatim(t *testing.T) {
	raw := json.RawMessage(`{"mutation":"createCourse","resourceID":"c-1"}`)
	entry := Entry{Details: raw}

	if string(entry.Details) != string(raw) {
		t.Errorf("Details = %s, want %s", entry.Details, raw)
	}
}

func TestFilter_ZeroValueSelectsNothing(t *testing.T) {
	// Every field of the zero Filter must be inert: QueryLogs adds a WHERE
	// clause per non-empty field, so a zero value that carried a bound would
	// silently hide rows from an unfiltered query.
	var f Filter

	if f.EventType != "" || f.SubjectID != "" || f.ServiceName != "" ||
		f.ResourceType != "" || f.ResourceID != "" {
		t.Errorf("zero Filter has a non-empty string field: %+v", f)
	}
	if f.StartDate != nil || f.EndDate != nil {
		t.Errorf("zero Filter has a date bound: %v..%v", f.StartDate, f.EndDate)
	}
	if f.Limit != 0 || f.Offset != 0 {
		t.Errorf("zero Filter has a page: limit=%d offset=%d", f.Limit, f.Offset)
	}

	// And the date fields are pointers precisely so "no bound" is expressible;
	// a bare time.Time would default to the zero instant and read as one.
	now := time.Now()
	f.StartDate = &now
	if f.StartDate == nil {
		t.Error("StartDate must be settable to an explicit bound")
	}
}
