package convert

import "github.com/jackc/pgx/v5/pgtype"

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

func TimeToPtr(v pgtype.Timestamptz) *string {
	if v.Valid {
		t := v.Time.Format("2006-01-02T15:04:05Z07:00")
		return &t
	}
	return nil
}
