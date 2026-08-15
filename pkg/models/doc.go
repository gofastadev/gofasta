// Package models provides BaseModelImpl, a struct you embed in your domain
// models to get standard fields: ID (UUID), CreatedAt, UpdatedAt, DeletedAt,
// RecordVersion, IsActive, IsDeletable. It includes a GORM BeforeCreate hook
// that auto-generates UUIDs and timestamps.
package models
