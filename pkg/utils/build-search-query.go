package utils

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// BuildQueryForAnyModel applies case-insensitive ILIKE filters for every
// non-nil *string value in filters, keyed by column name.
func BuildQueryForAnyModel(db *gorm.DB, filters map[string]interface{}) (*gorm.DB, error) {
	query := db
	for key, value := range filters {
		if strPtr, ok := value.(*string); ok && strPtr != nil {
			str := *strPtr
			query = query.Where(fmt.Sprintf("%s ILIKE ?", key), "%"+strings.ToLower(str)+"%")
		}
	}
	return query, query.Error
}
