package service

import "path/filepath"

func filepathAbsMigrations() (string, error) {
	return filepath.Abs("../../../migrations")
}
