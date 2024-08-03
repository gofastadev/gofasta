package utils

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

func BuildQueryForAnyModel(db *gorm.DB, filters map[string]string) (*gorm.DB, error) {
    query := db
    for key, value := range filters {
		query = query.Where(fmt.Sprintf("%s ILIKE ?", key), "%"+strings.ToLower(value)+"%")
    }
    return query, query.Error
}
