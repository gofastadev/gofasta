package auditlog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofastadev/gofasta/pkg/auth"
	"github.com/gofastadev/gofasta/pkg/httpcontext"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// auditLogsDDL mirrors the Entry model on a driver that has neither jsonb nor
// gen_random_uuid(). The id default matters: Entry tags one, so GORM leaves the
// column out of every INSERT and expects the database to fill it. A table
// without a working default would give every row the same empty primary key.
const auditLogsDDL = `
CREATE TABLE audit_logs (
	id             TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	event_type     TEXT NOT NULL,
	subject_id     TEXT,
	service_name   TEXT NOT NULL,
	ip_address     TEXT,
	user_agent     TEXT,
	details        TEXT,
	resource_type  TEXT,
	resource_id    TEXT,
	created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	deleted_at     DATETIME,
	record_version INTEGER NOT NULL DEFAULT 1,
	is_active      BOOLEAN NOT NULL DEFAULT 1,
	is_deletable   BOOLEAN NOT NULL DEFAULT 1
)`

// newAuditDB returns an isolated in-memory database holding the audit_logs
// table. The shared cache plus a per-test name lets the pool hand out more than
// one connection to the same data without two tests seeing each other's rows.
func newAuditDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := newEmptyDB(t)
	if err := db.Exec(auditLogsDDL).Error; err != nil {
		t.Fatalf("creating audit_logs: %v", err)
	}
	return db
}

// newEmptyDB is newAuditDB without the table, for the paths that must report a
// database failure rather than swallow it.
func newEmptyDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("opening sqlite: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("unwrapping sqlite: %v", err)
	}
	// An in-memory SQLite database lives as long as a connection to it does.
	// Let the pool close the last one and the schema disappears mid-test.
	sqlDB.SetMaxIdleConns(1)
	// One connection, because the asynchronous log methods write from their own
	// goroutine while the test polls for the row. Two connections into a
	// shared-cache in-memory database take shared-cache table locks against
	// each other and the write fails with "database table is locked" — an
	// artifact of this fixture, not of the code under test. Postgres, which is
	// what these tables actually run on, has no such restriction.
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

// eventually polls until want holds. The asynchronous log methods hand the
// write to a goroutine and return, so there is no handle to wait on — polling
// is what a caller would have to do too.
func eventually(t *testing.T, what string, want func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if want() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// entriesIn returns every row, oldest first.
func entriesIn(t *testing.T, db *gorm.DB) []Entry {
	t.Helper()
	var entries []Entry
	if err := db.Order("created_at ASC").Find(&entries).Error; err != nil {
		t.Fatalf("reading audit_logs: %v", err)
	}
	return entries
}

func strPtr(s string) *string { return &s }

// ctxWithRequest builds the context httpcontext.Middleware would produce.
func ctxWithRequest(remoteAddr, userAgent, forwardedFor string) context.Context {
	r := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	r.RemoteAddr = remoteAddr
	if userAgent != "" {
		r.Header.Set("User-Agent", userAgent)
	}
	if forwardedFor != "" {
		r.Header.Set("X-Forwarded-For", forwardedFor)
	}
	return httpcontext.WithRequest(context.Background(), r)
}

func TestFromContext_CarriesClientAddressAndAgent(t *testing.T) {
	s := NewAuditService(nil, "test")

	_, ip, ua := s.FromContext(ctxWithRequest("203.0.113.7:5555", "probe/1.0", ""))

	if ip != "203.0.113.7:5555" {
		t.Errorf("ipAddress = %q, want the remote address", ip)
	}
	if ua != "probe/1.0" {
		t.Errorf("userAgent = %q", ua)
	}
}

func TestFromContext_ForwardedForWinsOverRemoteAddr(t *testing.T) {
	// Behind a proxy, RemoteAddr is the proxy. Recording it for every user
	// makes the whole column useless.
	s := NewAuditService(nil, "test")

	_, ip, _ := s.FromContext(ctxWithRequest("10.0.0.1:443", "", "198.51.100.9"))

	if ip != "198.51.100.9" {
		t.Errorf("ipAddress = %q, want the forwarded client address", ip)
	}
}

// This is the failure that shipped: a service installing its own
// request-in-context middleware with its own key wrote audit rows whose IP
// address and user agent were empty, and nothing reported it.
func TestFromContext_WithoutTheMiddlewareYieldsNothing(t *testing.T) {
	s := NewAuditService(nil, "test")

	subjectID, ip, ua := s.FromContext(context.Background())

	if subjectID != nil {
		t.Errorf("subjectID = %v, want nil", subjectID)
	}
	if ip != "" || ua != "" {
		t.Errorf("ip/ua = %q/%q, want empty — the caller must be able to detect this", ip, ua)
	}
}

func TestFromContext_DefaultSubjectReadsTheSubClaim(t *testing.T) {
	// The subject of an audit row is the token's registered `sub` claim, which
	// is what every OAuth 2.0 / OIDC issuer emits — including Solago, whose
	// tokens carry a UUID there.
	s := NewAuditService(nil, "test")

	ctx := context.WithValue(context.Background(), auth.ClaimsKey, &auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "11111111-2222-3333-4444-555555555555"},
	})
	subjectID, _, _ := s.FromContext(ctx)

	if subjectID == nil || *subjectID != "11111111-2222-3333-4444-555555555555" {
		t.Errorf("subjectID = %v, want the sub claim", subjectID)
	}
}

func TestFromContext_SubjectMayBeAClientNotAUser(t *testing.T) {
	// Under the client-credentials grant `sub` holds a client id. The column is
	// called subject_id rather than user_id precisely so this row is not read as
	// naming a person.
	s := NewAuditService(nil, "test")

	ctx := context.WithValue(context.Background(), auth.ClaimsKey, &auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "jedi_lms"},
	})
	subjectID, _, _ := s.FromContext(ctx)

	if subjectID == nil || *subjectID != "jedi_lms" {
		t.Errorf("subjectID = %v, want the client id", subjectID)
	}
}

func TestWithSubjectFunc_ReplacesIdentityExtraction(t *testing.T) {
	// The reason the hook exists: a project whose tokens carry the identity in
	// a different claim must not have to fork the package to say so.
	type projectClaims struct{ Sub string }
	type claimsKey struct{}

	s := NewAuditService(nil, "test", WithSubjectFunc(func(ctx context.Context) *string {
		c, ok := ctx.Value(claimsKey{}).(*projectClaims)
		if !ok || c.Sub == "" {
			return nil
		}
		return &c.Sub
	}))

	ctx := context.WithValue(context.Background(), claimsKey{}, &projectClaims{Sub: "solago-subject"})
	subjectID, _, _ := s.FromContext(ctx)

	if subjectID == nil || *subjectID != "solago-subject" {
		t.Errorf("subjectID = %v, want solago-subject", subjectID)
	}
}

func TestWithSubjectFunc_NilIsIgnored(t *testing.T) {
	// A nil hook would disable identity extraction entirely and silently.
	s := NewAuditService(nil, "test", WithSubjectFunc(nil))

	ctx := context.WithValue(context.Background(), auth.ClaimsKey, &auth.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "user-42"}})
	if subjectID, _, _ := s.FromContext(ctx); subjectID == nil {
		t.Error("a nil subject func displaced the default")
	}
}

func TestFromContext_EndToEndThroughTheMiddleware(t *testing.T) {
	// The wiring a service actually installs, asserted as one path.
	s := NewAuditService(nil, "test")

	var ip, ua string
	handler := httpcontext.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, ip, ua = s.FromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	req.RemoteAddr = "192.0.2.44:1234"
	req.Header.Set("User-Agent", "jedi/2.0")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if ip != "192.0.2.44:1234" || ua != "jedi/2.0" {
		t.Errorf("ip/ua = %q/%q through the real middleware", ip, ua)
	}
}

func TestFromContext_ClaimsWithNoIdentityYieldNoSubject(t *testing.T) {
	s := NewAuditService(nil, "test")

	ctx := context.WithValue(context.Background(), auth.ClaimsKey, &auth.Claims{})
	if subjectID, _, _ := s.FromContext(ctx); subjectID != nil {
		t.Errorf("subjectID = %v, want nil so the empty case stays detectable", subjectID)
	}
}

// ---------- the write path ----------

func TestLogEventSync_PersistsEveryColumn(t *testing.T) {
	db := newAuditDB(t)
	s := NewAuditService(db, "solago")

	err := s.LogEventSync(LoginOTPVerified, strPtr("user-1"), "203.0.113.7", "probe/1.0",
		map[string]interface{}{"method": "otp"})
	if err != nil {
		t.Fatalf("LogEventSync: %v", err)
	}

	entries := entriesIn(t, db)
	if len(entries) != 1 {
		t.Fatalf("got %d rows, want 1", len(entries))
	}
	got := entries[0]

	if got.ID == "" {
		t.Error("ID is empty — the database default did not fill the primary key")
	}
	if got.EventType != LoginOTPVerified {
		t.Errorf("EventType = %q", got.EventType)
	}
	if got.SubjectID == nil || *got.SubjectID != "user-1" {
		t.Errorf("SubjectID = %v", got.SubjectID)
	}
	if got.ServiceName != "solago" {
		t.Errorf("ServiceName = %q, want the service the Service was built for", got.ServiceName)
	}
	if got.IPAddress != "203.0.113.7" || got.UserAgent != "probe/1.0" {
		t.Errorf("ip/ua = %q/%q", got.IPAddress, got.UserAgent)
	}
	if string(got.Details) != `{"method":"otp"}` {
		t.Errorf("Details = %s, want the payload stored verbatim", got.Details)
	}
}

func TestLogEventSync_NilDetailsStoresNull(t *testing.T) {
	// A nil map must not become the four bytes "null" or an empty object —
	// "no details were recorded" and "the details were empty" are different
	// findings when someone is reconstructing an incident.
	db := newAuditDB(t)
	s := NewAuditService(db, "solago")

	if err := s.LogEventSync(Logout, nil, "", "", nil); err != nil {
		t.Fatalf("LogEventSync: %v", err)
	}

	entries := entriesIn(t, db)
	if len(entries) != 1 {
		t.Fatalf("got %d rows, want 1", len(entries))
	}
	if entries[0].Details != nil {
		t.Errorf("Details = %s, want NULL", entries[0].Details)
	}
	if entries[0].SubjectID != nil {
		t.Errorf("SubjectID = %v, want NULL for an unattributed action", entries[0].SubjectID)
	}
}

func TestLogEventSync_UnserializableDetailsAreReported(t *testing.T) {
	// The synchronous method exists for the events where the caller wants to
	// know. A marshal failure that returned nil would leave them believing a
	// row was written.
	db := newAuditDB(t)
	s := NewAuditService(db, "solago")

	err := s.LogEventSync("ODD", nil, "", "", map[string]interface{}{"ch": make(chan int)})

	if err == nil {
		t.Fatal("LogEventSync returned nil for details that cannot be marshaled")
	}
	if rows := entriesIn(t, db); len(rows) != 0 {
		t.Errorf("got %d rows, want none — nothing should be written after the failure", len(rows))
	}
}

func TestLogEventSync_DatabaseFailureIsReturned(t *testing.T) {
	// No audit_logs table: the write cannot succeed, and the caller must be
	// told rather than left with a silent gap in the trail.
	s := NewAuditService(newEmptyDB(t), "solago")

	if err := s.LogEventSync(Logout, nil, "", "", nil); err == nil {
		t.Fatal("LogEventSync returned nil against a database with no audit_logs table")
	}
}

func TestLogEvent_WritesAsynchronously(t *testing.T) {
	db := newAuditDB(t)
	s := NewAuditService(db, "solago")

	s.LogEvent(TokenRefresh, strPtr("user-2"), "198.51.100.9", "cli/1.0",
		map[string]interface{}{"grant": "refresh_token"})

	eventually(t, "the asynchronous row", func() bool {
		var n int64
		db.Model(&Entry{}).Where("event_type = ?", TokenRefresh).Count(&n)
		return n == 1
	})
}

func TestLogEvent_FailureDoesNotReachTheCaller(t *testing.T) {
	// The whole point of the asynchronous path: an audit write must never be
	// the reason a user's action fails. There is no table here, so the write
	// fails — and LogEvent still returns, with nothing for the caller to check.
	s := NewAuditService(newEmptyDB(t), "solago")

	s.LogEvent(Logout, nil, "", "", nil)

	// Give the goroutine a chance to fail before the test tears the DB down.
	time.Sleep(50 * time.Millisecond)
}

func TestLogEventWithResource_RecordsWhatWasActedOn(t *testing.T) {
	db := newAuditDB(t)
	s := NewAuditService(db, "solago")

	s.LogEventWithResource(EventName("COURSE", ActionUpdated), strPtr("user-3"), "", "",
		map[string]interface{}{"field": "title"}, "COURSE", "course-42")

	eventually(t, "the resource row", func() bool {
		var n int64
		db.Model(&Entry{}).Where("resource_id = ?", "course-42").Count(&n)
		return n == 1
	})

	entries := entriesIn(t, db)
	if entries[0].ResourceType != "COURSE" || entries[0].ResourceID != "course-42" {
		t.Errorf("resource = %q/%q, want COURSE/course-42",
			entries[0].ResourceType, entries[0].ResourceID)
	}
}

func TestLogFromContext_CarriesTheRequestIdentity(t *testing.T) {
	db := newAuditDB(t)
	s := NewAuditService(db, "solago")

	ctx := context.WithValue(ctxWithRequest("10.0.0.1:443", "jedi/2.0", "198.51.100.9"),
		auth.ClaimsKey, &auth.Claims{
			RegisteredClaims: jwt.RegisteredClaims{Subject: "user-4"},
		})

	s.LogFromContext(ctx, LoginSocial, map[string]interface{}{"provider": "google"})

	eventually(t, "the context-derived row", func() bool {
		var n int64
		db.Model(&Entry{}).Where("subject_id = ?", "user-4").Count(&n)
		return n == 1
	})

	got := entriesIn(t, db)[0]
	if got.IPAddress != "198.51.100.9" {
		t.Errorf("IPAddress = %q, want the forwarded client address", got.IPAddress)
	}
	if got.UserAgent != "jedi/2.0" {
		t.Errorf("UserAgent = %q", got.UserAgent)
	}
}

func TestLogFromContextWithResource_CarriesBoth(t *testing.T) {
	db := newAuditDB(t)
	s := NewAuditService(db, "solago")

	ctx := context.WithValue(ctxWithRequest("203.0.113.7:5555", "probe/1.0", ""),
		auth.ClaimsKey, &auth.Claims{
			RegisteredClaims: jwt.RegisteredClaims{Subject: "user-5"},
		})

	s.LogFromContextWithResource(ctx, "COURSE_DELETED", "COURSE", "course-9",
		map[string]interface{}{"mutation": "deleteCourse"})

	eventually(t, "the context-derived resource row", func() bool {
		var n int64
		db.Model(&Entry{}).Where("resource_id = ?", "course-9").Count(&n)
		return n == 1
	})

	got := entriesIn(t, db)[0]
	if got.SubjectID == nil || *got.SubjectID != "user-5" {
		t.Errorf("SubjectID = %v, want the sub claim from the context", got.SubjectID)
	}
	if got.ResourceType != "COURSE" || got.IPAddress != "203.0.113.7:5555" {
		t.Errorf("resource/ip = %q/%q", got.ResourceType, got.IPAddress)
	}
}

// ---------- the read path ----------

// seedEntries writes rows directly, bypassing the service, so a query test is
// not also a test of the write path.
func seedEntries(t *testing.T, db *gorm.DB, entries ...Entry) {
	t.Helper()
	for i := range entries {
		if entries[i].ID == "" {
			entries[i].ID = fmt.Sprintf("row-%d", i)
		}
		if err := db.Create(&entries[i]).Error; err != nil {
			t.Fatalf("seeding row %d: %v", i, err)
		}
	}
}

func TestQueryLogs_FiltersOnEveryField(t *testing.T) {
	db := newAuditDB(t)
	s := NewAuditService(db, "solago")

	old := time.Now().Add(-48 * time.Hour)
	recent := time.Now().Add(-1 * time.Hour)

	seedEntries(t, db,
		Entry{EventType: Logout, SubjectID: strPtr("user-1"), ServiceName: "solago",
			ResourceType: "SESSION", ResourceID: "s-1", CreatedAt: old},
		Entry{EventType: LoginSocial, SubjectID: strPtr("user-2"), ServiceName: "solago",
			ResourceType: "SESSION", ResourceID: "s-2", CreatedAt: recent},
		Entry{EventType: LoginSocial, SubjectID: strPtr("user-2"), ServiceName: "dinero",
			ResourceType: "INVOICE", ResourceID: "i-1", CreatedAt: recent},
	)

	cases := []struct {
		name   string
		filter Filter
		want   int64
	}{
		{"unfiltered", Filter{}, 3},
		{"event type", Filter{EventType: LoginSocial}, 2},
		{"subject", Filter{SubjectID: "user-2"}, 2},
		{"service", Filter{ServiceName: "dinero"}, 1},
		{"resource type", Filter{ResourceType: "SESSION"}, 2},
		{"resource id", Filter{ResourceID: "i-1"}, 1},
		{"start date", Filter{StartDate: &recent}, 2},
		{"end date", Filter{EndDate: &old}, 1},
		{"both dates", Filter{StartDate: &old, EndDate: &recent}, 3},
		{"combined", Filter{EventType: LoginSocial, ServiceName: "solago"}, 1},
		{"matches nothing", Filter{EventType: "NEVER_HAPPENED"}, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logs, total, err := s.QueryLogs(tc.filter)
			if err != nil {
				t.Fatalf("QueryLogs: %v", err)
			}
			if total != tc.want {
				t.Errorf("total = %d, want %d", total, tc.want)
			}
			if int64(len(logs)) != tc.want {
				t.Errorf("len(logs) = %d, want %d", len(logs), tc.want)
			}
		})
	}
}

func TestQueryLogs_NewestFirst(t *testing.T) {
	// An audit query answers "what just happened", so the most recent row has
	// to be the one on the first page.
	db := newAuditDB(t)
	s := NewAuditService(db, "solago")

	seedEntries(t, db,
		Entry{EventType: "OLDEST", ServiceName: "solago", CreatedAt: time.Now().Add(-3 * time.Hour)},
		Entry{EventType: "NEWEST", ServiceName: "solago", CreatedAt: time.Now().Add(-1 * time.Hour)},
		Entry{EventType: "MIDDLE", ServiceName: "solago", CreatedAt: time.Now().Add(-2 * time.Hour)},
	)

	logs, _, err := s.QueryLogs(Filter{})
	if err != nil {
		t.Fatalf("QueryLogs: %v", err)
	}
	if logs[0].EventType != "NEWEST" || logs[2].EventType != "OLDEST" {
		t.Errorf("order = %q, %q, %q; want NEWEST first",
			logs[0].EventType, logs[1].EventType, logs[2].EventType)
	}
}

func TestQueryLogs_PagesAndDefaultsTheLimit(t *testing.T) {
	// An unset limit must not mean "every row ever recorded" — that table is
	// the one that grows without bound.
	db := newAuditDB(t)
	s := NewAuditService(db, "solago")

	for i := 0; i < 5; i++ {
		seedEntries(t, db, Entry{
			ID:          fmt.Sprintf("page-%d", i),
			EventType:   fmt.Sprintf("E%d", i),
			ServiceName: "solago",
			CreatedAt:   time.Now().Add(-time.Duration(i) * time.Hour),
		})
	}

	all, total, err := s.QueryLogs(Filter{})
	if err != nil {
		t.Fatalf("QueryLogs: %v", err)
	}
	if total != 5 || len(all) != 5 {
		t.Fatalf("unpaged: total=%d len=%d, want 5/5", total, len(all))
	}

	page, total, err := s.QueryLogs(Filter{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatalf("QueryLogs: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want the unpaged count", total)
	}
	if len(page) != 2 {
		t.Fatalf("len(page) = %d, want 2", len(page))
	}
	if page[0].EventType != all[2].EventType {
		t.Errorf("offset 2 started at %q, want %q", page[0].EventType, all[2].EventType)
	}
}

func TestQueryLogs_DatabaseFailureIsReturned(t *testing.T) {
	s := NewAuditService(newEmptyDB(t), "solago")

	logs, total, err := s.QueryLogs(Filter{})

	if err == nil {
		t.Fatal("QueryLogs returned nil against a database with no audit_logs table")
	}
	if logs != nil || total != 0 {
		t.Errorf("logs/total = %v/%d, want nil/0 alongside the error", logs, total)
	}
}

func TestLogEventSync_DetailsSurviveAsJSON(t *testing.T) {
	// Details is json.RawMessage so what is read back is what was written,
	// nested structure included.
	db := newAuditDB(t)
	s := NewAuditService(db, "solago")

	if err := s.LogEventSync("NESTED", nil, "", "", map[string]interface{}{
		"changes": map[string]interface{}{"title": "new"},
	}); err != nil {
		t.Fatalf("LogEventSync: %v", err)
	}

	var decoded struct {
		Changes struct {
			Title string `json:"title"`
		} `json:"changes"`
	}
	if err := json.Unmarshal(entriesIn(t, db)[0].Details, &decoded); err != nil {
		t.Fatalf("stored details are not valid JSON: %v", err)
	}
	if decoded.Changes.Title != "new" {
		t.Errorf("changes.title = %q, want new", decoded.Changes.Title)
	}
}

func TestQueryLogs_ScanFailureIsReturned(t *testing.T) {
	// The count and the read are two statements against the same table, and
	// only the second one touches column values. A schema that has drifted
	// from the model — created_at holding text a date cannot be read from —
	// therefore counts fine and fails on the way out. Returning the rows read
	// so far would present a truncated audit trail as a complete one.
	db := newEmptyDB(t)
	if err := db.Exec(`
		CREATE TABLE audit_logs (
			id           TEXT PRIMARY KEY,
			event_type   TEXT NOT NULL,
			service_name TEXT NOT NULL,
			created_at   TEXT NOT NULL,
			deleted_at   DATETIME
		)`).Error; err != nil {
		t.Fatalf("creating drifted audit_logs: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO audit_logs (id, event_type, service_name, created_at) VALUES (?, ?, ?, ?)`,
		"drift-1", Logout, "solago", "whenever",
	).Error; err != nil {
		t.Fatalf("seeding drifted row: %v", err)
	}

	s := NewAuditService(db, "solago")

	logs, total, err := s.QueryLogs(Filter{})

	if err == nil {
		t.Fatal("QueryLogs returned nil for a row it could not scan")
	}
	if logs != nil || total != 0 {
		t.Errorf("logs/total = %v/%d, want nil/0 alongside the error", logs, total)
	}
}
