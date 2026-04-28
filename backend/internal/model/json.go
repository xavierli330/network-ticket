package model

import (
	"database/sql/driver"
	"fmt"
)

// JSON is a shared type for MySQL JSON columns.
type JSON []byte

// Scan implements sql.Scanner for JSON.
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan JSON: %v", value)
	}
	*j = append((*j)[0:0], bytes...)
	return nil
}

// Value implements driver.Valuer for JSON.
func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}
