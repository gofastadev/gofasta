package utils

import (
	"strings"

	"gorm.io/gorm"
)

// BuildQueryForAnyModel applies the per-column filters to the query.
//
// allowedColumns is the caller's column allow-list — the same trust
// boundary SafeOrderBy establishes for ORDER BY: filter keys are
// interpolated into SQL as column names, so only names on the list are
// honored and anything else is silently skipped (a probe against a list
// endpoint should degrade to "unfiltered", not 500).
//
// Value handling by type:
//   - string / *string: case-insensitive substring match. The SQL form
//     is driver-aware (see likeClause): ILIKE on postgres/clickhouse,
//     LOWER(col) LIKE LOWER(?) on mysql/sqlserver, plain LIKE on sqlite
//     (case-insensitive for ASCII by default). LIKE metacharacters in
//     the needle are escaped, so a literal "%" or "_" matches itself.
//     Note: on sqlite and clickhouse, case folding is ASCII-only.
//   - every other non-nil value (int, float64, bool, uuid.UUID,
//     time.Time, and their pointer forms): exact equality.
//   - nil pointers: skipped ("don't filter on this column").
func BuildQueryForAnyModel(db *gorm.DB, filters map[string]interface{}, allowedColumns []string) (*gorm.DB, error) {
	query := db
	driver := db.Name()
	for column, raw := range filters {
		if !containsColumn(allowedColumns, column) {
			continue
		}
		value, ok := derefFilterValue(raw)
		if !ok {
			continue
		}
		if s, isString := value.(string); isString {
			query = query.Where(likeClause(driver, column), "%"+escapeLikePattern(driver, s)+"%")
			continue
		}
		query = query.Where(column+" = ?", value)
	}
	return query, query.Error
}

// derefFilterValue unwraps the pointer forms filter maps commonly carry
// (a nil pointer means "no filter on this column") and passes value
// types through untouched.
func derefFilterValue(v interface{}) (interface{}, bool) {
	switch t := v.(type) {
	case nil:
		return nil, false
	case *string:
		if t == nil {
			return nil, false
		}
		return *t, true
	case *int:
		if t == nil {
			return nil, false
		}
		return *t, true
	case *int64:
		if t == nil {
			return nil, false
		}
		return *t, true
	case *float64:
		if t == nil {
			return nil, false
		}
		return *t, true
	case *bool:
		if t == nil {
			return nil, false
		}
		return *t, true
	default:
		return v, true
	}
}

// likeClause returns the driver-appropriate case-insensitive LIKE
// expression for column. The single placeholder receives the escaped
// %needle% pattern.
func likeClause(driver, column string) string {
	switch driver {
	case "postgres", "clickhouse":
		return column + " ILIKE ?"
	case "mysql":
		// Backslash is MySQL's default LIKE escape character — no
		// ESCAPE clause needed.
		return "LOWER(" + column + ") LIKE LOWER(?)"
	case "sqlserver":
		return "LOWER(" + column + `) LIKE LOWER(?) ESCAPE '\'`
	default:
		// sqlite (LIKE is case-insensitive for ASCII) and anything
		// unrecognized. The explicit ESCAPE makes backslash escaping
		// effective — sqlite has no default escape character.
		return column + ` LIKE ? ESCAPE '\'`
	}
}

// escapeLikePattern escapes LIKE metacharacters in a user-supplied
// needle so they match literally: backslash, percent, underscore — and
// `[` on SQL Server, whose LIKE additionally treats it as a character
// class opener.
func escapeLikePattern(driver, s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	if driver == "sqlserver" {
		s = strings.ReplaceAll(s, `[`, `\[`)
	}
	return s
}
