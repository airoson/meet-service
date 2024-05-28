package utils

import (
	"database/sql"
	"fmt"
)

func ClearUnusedDataStartup(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM room WHERE active=true OR starts_at IS NOT NULL AND starts_at < now();")
	if err != nil {
		return fmt.Errorf("can't perform startup cleaning: can't delete unused rooms: %v", err)
	}
	_, err = db.Exec("DELETE FROM user_at_call")
	if err != nil {
		return fmt.Errorf("can't perform startup cleaning: can't delete users: %v", err)
	}
	return nil
}
