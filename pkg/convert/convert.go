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

func StringToUUID(s string) (pgtype.UUID, error) {
	var pgId pgtype.UUID
	err := pgId.Scan(s)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgId, nil
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

func PtrToText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{
		String: *s,
		Valid:  true,
	}
}
