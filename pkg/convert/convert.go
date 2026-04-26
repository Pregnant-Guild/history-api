package convert

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func GenerateQueryKey(prefix string, params any) string {
	b, _ := json.Marshal(params)
	hash := fmt.Sprintf("%x", md5.Sum(b))
	return fmt.Sprintf("%s:query:%s", prefix, hash)
}

func UUIDToString(v pgtype.UUID) string {
	if v.Valid {
		return v.String()
	}
	return ""
}

func UUIDToStringPtr(v pgtype.UUID) *string {
	if v.Valid {
		str := v.String()
		return &str
	}
	return nil
}

func ListUUIDToString(v []pgtype.UUID) []string {
	if len(v) == 0 {
		return []string{}
	}
	res := make([]string, 0, len(v))
	for _, u := range v {
		if u.Valid {
			res = append(res, UUIDToString(u))
		}
	}
	return res
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
	if s == nil || *s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{
		String: *s,
		Valid:  true,
	}
}

func StringToText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{
		String: s,
		Valid:  true,
	}
}

func TextToPtr(v pgtype.Text) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func Int4ToPtr(v pgtype.Int4) *int32 {
	if !v.Valid {
		return nil
	}
	return &v.Int32
}

func Int4ToInt32(v pgtype.Int4) int32 {
	if v.Valid {
		return v.Int32
	}
	return 0
}
