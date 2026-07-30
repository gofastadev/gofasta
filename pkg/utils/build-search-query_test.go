package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func searchTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

type searchModel struct {
	ID    uint
	Name  string
	Email string
	Count int
}

// dryRunSQL renders the WHERE clause BuildQueryForAnyModel produced,
// without touching a real table beyond sqlite's dry-run planner.
func dryRunSQL(t *testing.T, db *gorm.DB) (sql string, vars []interface{}) {
	t.Helper()
	var rows []searchModel
	tx := db.Session(&gorm.Session{DryRun: true}).Find(&rows)
	return tx.Statement.SQL.String(), tx.Statement.Vars
}

func TestBuildQueryForAnyModel_StringValueFilters(t *testing.T) {
	db := searchTestDB(t).Model(&searchModel{})
	q, err := BuildQueryForAnyModel(db, map[string]interface{}{"name": "John"},
		[]string{"name", "email"})
	require.NoError(t, err)

	sql, vars := dryRunSQL(t, q)
	assert.Contains(t, sql, "name LIKE ?")
	require.Len(t, vars, 1)
	assert.Equal(t, "%John%", vars[0])
}

func TestBuildQueryForAnyModel_StringPointerFilters(t *testing.T) {
	db := searchTestDB(t).Model(&searchModel{})
	name := "Alice"
	q, err := BuildQueryForAnyModel(db, map[string]interface{}{"name": &name},
		[]string{"name"})
	require.NoError(t, err)

	sql, vars := dryRunSQL(t, q)
	assert.Contains(t, sql, "name LIKE ?")
	assert.Equal(t, []interface{}{"%Alice%"}, vars)
}

func TestBuildQueryForAnyModel_NonStringEquality(t *testing.T) {
	db := searchTestDB(t).Model(&searchModel{})
	count := 42
	active := true
	q, err := BuildQueryForAnyModel(db, map[string]interface{}{
		"count":  &count,
		"active": active,
	}, []string{"count", "active"})
	require.NoError(t, err)

	sql, vars := dryRunSQL(t, q)
	assert.Contains(t, sql, "count = ?")
	assert.Contains(t, sql, "active = ?")
	assert.Len(t, vars, 2)
}

func TestBuildQueryForAnyModel_NilPointerSkipped(t *testing.T) {
	db := searchTestDB(t).Model(&searchModel{})
	q, err := BuildQueryForAnyModel(db, map[string]interface{}{
		"name":  (*string)(nil),
		"count": (*int)(nil),
	}, []string{"name", "count"})
	require.NoError(t, err)

	sql, _ := dryRunSQL(t, q)
	assert.NotContains(t, sql, "WHERE")
}

func TestBuildQueryForAnyModel_ColumnNotAllowedSkipped(t *testing.T) {
	db := searchTestDB(t).Model(&searchModel{})
	q, err := BuildQueryForAnyModel(db, map[string]interface{}{
		"name":                    "x",
		"1=1; DROP TABLE users--": "y",
	}, []string{"name"})
	require.NoError(t, err)

	sql, _ := dryRunSQL(t, q)
	assert.Contains(t, sql, "name LIKE ?")
	assert.NotContains(t, sql, "DROP TABLE")
}

func TestBuildQueryForAnyModel_EmptyFilters(t *testing.T) {
	db := searchTestDB(t).Model(&searchModel{})
	q, err := BuildQueryForAnyModel(db, map[string]interface{}{}, []string{"name"})
	require.NoError(t, err)

	sql, _ := dryRunSQL(t, q)
	assert.NotContains(t, sql, "WHERE")
}

// TestBuildQueryForAnyModel_EndToEndSQLite proves the filter actually
// narrows result sets against a live (in-memory) database — the class
// of regression where every filter silently matched everything.
func TestBuildQueryForAnyModel_EndToEndSQLite(t *testing.T) {
	db := searchTestDB(t)
	require.NoError(t, db.AutoMigrate(&searchModel{}))
	require.NoError(t, db.Create(&[]searchModel{
		{Name: "Alice Smith", Email: "alice@example.com", Count: 1},
		{Name: "Bob Jones", Email: "bob@example.com", Count: 2},
		{Name: "alice cooper", Email: "cooper@example.com", Count: 3},
	}).Error)

	q, err := BuildQueryForAnyModel(db.Model(&searchModel{}),
		map[string]interface{}{"name": "alice"}, []string{"name"})
	require.NoError(t, err)

	var got []searchModel
	require.NoError(t, q.Find(&got).Error)
	require.Len(t, got, 2, "case-insensitive substring match must hit both Alices")

	q, err = BuildQueryForAnyModel(db.Model(&searchModel{}),
		map[string]interface{}{"count": 2}, []string{"count"})
	require.NoError(t, err)
	require.NoError(t, q.Find(&got).Error)
	require.Len(t, got, 1)
	assert.Equal(t, "Bob Jones", got[0].Name)
}

// TestBuildQueryForAnyModel_LikeMetacharactersLiteral — a needle
// containing % or _ matches those characters literally.
func TestBuildQueryForAnyModel_LikeMetacharactersLiteral(t *testing.T) {
	db := searchTestDB(t)
	require.NoError(t, db.AutoMigrate(&searchModel{}))
	require.NoError(t, db.Create(&[]searchModel{
		{Name: "100% cotton"},
		{Name: "100x cotton"},
		{Name: "snake_case"},
		{Name: "snakeXcase"},
	}).Error)

	var got []searchModel
	q, err := BuildQueryForAnyModel(db.Model(&searchModel{}),
		map[string]interface{}{"name": "100%"}, []string{"name"})
	require.NoError(t, err)
	require.NoError(t, q.Find(&got).Error)
	require.Len(t, got, 1)
	assert.Equal(t, "100% cotton", got[0].Name)

	q, err = BuildQueryForAnyModel(db.Model(&searchModel{}),
		map[string]interface{}{"name": "snake_"}, []string{"name"})
	require.NoError(t, err)
	require.NoError(t, q.Find(&got).Error)
	require.Len(t, got, 1)
	assert.Equal(t, "snake_case", got[0].Name)
}

func TestLikeClause_PerDriver(t *testing.T) {
	cases := map[string]string{
		"postgres":   "name ILIKE ?",
		"clickhouse": "name ILIKE ?",
		"mysql":      "LOWER(name) LIKE LOWER(?)",
		"sqlserver":  `LOWER(name) LIKE LOWER(?) ESCAPE '\'`,
		"sqlite":     `name LIKE ? ESCAPE '\'`,
		"unknown":    `name LIKE ? ESCAPE '\'`,
	}
	for driver, want := range cases {
		assert.Equal(t, want, likeClause(driver, "name"), "driver=%s", driver)
	}
}

func TestEscapeLikePattern(t *testing.T) {
	assert.Equal(t, `100\%`, escapeLikePattern("postgres", "100%"))
	assert.Equal(t, `a\_b`, escapeLikePattern("mysql", "a_b"))
	assert.Equal(t, `back\\slash`, escapeLikePattern("sqlite", `back\slash`))
	// sqlserver additionally escapes the character-class opener.
	assert.Equal(t, `\[tag]`, escapeLikePattern("sqlserver", "[tag]"))
	assert.Equal(t, "[tag]", escapeLikePattern("postgres", "[tag]"))
}

func TestDerefFilterValue(t *testing.T) {
	s := "v"
	i := 7
	f := 1.5
	b := true
	var i64 int64 = 9

	got, ok := derefFilterValue(&s)
	assert.True(t, ok)
	assert.Equal(t, "v", got)
	got, ok = derefFilterValue(&i)
	assert.True(t, ok)
	assert.Equal(t, 7, got)
	got, ok = derefFilterValue(&i64)
	assert.True(t, ok)
	assert.Equal(t, int64(9), got)
	got, ok = derefFilterValue(&f)
	assert.True(t, ok)
	assert.Equal(t, 1.5, got)
	got, ok = derefFilterValue(&b)
	assert.True(t, ok)
	assert.Equal(t, true, got)

	_, ok = derefFilterValue(nil)
	assert.False(t, ok)
	_, ok = derefFilterValue((*string)(nil))
	assert.False(t, ok)
	_, ok = derefFilterValue((*bool)(nil))
	assert.False(t, ok)
}

// TestDerefFilterValue_AllNilPointerForms — every pointer arm's nil
// branch must skip the filter (not dereference).
func TestDerefFilterValue_AllNilPointerForms(t *testing.T) {
	for _, v := range []interface{}{
		(*string)(nil),
		(*int)(nil),
		(*int64)(nil),
		(*float64)(nil),
		(*bool)(nil),
	} {
		_, ok := derefFilterValue(v)
		assert.False(t, ok, "%T nil pointer must be skipped", v)
	}
}
