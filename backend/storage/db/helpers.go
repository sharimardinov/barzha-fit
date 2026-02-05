package db

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

// nullStr returns nil if s is empty, otherwise returns s.
// Used for inserting optional string fields into the database.
func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nullInt returns nil if v is 0, otherwise returns v.
// Used for inserting optional integer fields into the database.
func nullInt(v int) any {
	if v == 0 {
		return nil
	}
	return v
}

// nullFloat returns nil if v is 0, otherwise returns v.
// Used for inserting optional float fields into the database.
func nullFloat(v float64) any {
	if v == 0 {
		return nil
	}
	return v
}

// nullBoolPtr returns nil if v is nil, otherwise returns the dereferenced value.
// Used for inserting optional boolean pointer fields into the database.
func nullBoolPtr(v *bool) any {
	if v == nil {
		return nil
	}
	return *v
}

// rawJSON returns nil if b is empty, otherwise returns the string representation.
// Used for inserting JSONB fields into PostgreSQL.
func rawJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return string(b)
}

// IsNotFound returns true if the error is pgx.ErrNoRows.
// Use this for consistent error handling across repositories.
func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
