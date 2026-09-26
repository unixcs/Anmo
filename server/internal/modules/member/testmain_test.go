package member

import "path/filepath"

// absMigrations returns the migrations dir relative to this package
// (internal/modules/member → server/migrations).
func absMigrations() (string, error) {
	return filepath.Abs("../../../migrations")
}
