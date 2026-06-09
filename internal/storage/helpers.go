// internal/storage/helpers.go
package storage

// boolInt converts a bool to 1 (true) or 0 (false) for SQLite INTEGER columns.
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// nullStr returns nil for an empty string so the driver stores SQL NULL,
// or the string value itself otherwise.
func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
