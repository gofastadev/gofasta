package utils

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

func BuildQueryForAnyModel(db *gorm.DB, filters map[string]string) (*gorm.DB, error) {
    query := db
    for key, value := range filters {
        if value != "" {
            query = query.Where(fmt.Sprintf("%s ILIKE ?", key), "%"+strings.ToLower(value)+"%")
        }
    }
    return query, nil
}
