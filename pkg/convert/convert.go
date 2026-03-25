package convert

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func UUIDToString(v pgtype.UUID) string {
	if v.Valid {
		return v.String()
	}
	return ""
}

func TextToString(v pgtype.Text) string {
	if v.Valid {
		return v.String
	}
	return ""
}

func BoolVal(v pgtype.Bool) bool {
	if v.Valid {
		return v.Bool
	}
	return false
}

func TimeToPtr(v pgtype.Timestamptz) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}
