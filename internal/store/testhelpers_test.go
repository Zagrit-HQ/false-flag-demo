package store_test

import "github.com/google/uuid"

func nullUUID(s string) uuid.NullUUID {
	u, err := uuid.Parse(s)
	if err != nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: u, Valid: true}
}
