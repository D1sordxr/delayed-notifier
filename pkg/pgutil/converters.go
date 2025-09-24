package pgutil

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// ToText конвертирует string в pgtype.Text
func ToText(val string) pgtype.Text {
	if val == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: val, Valid: true}
}

// ToTextPtrPg конвертирует *string в pgtype.Text
func ToTextPtrPg(val *string) pgtype.Text {
	if val == nil || *val == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *val, Valid: true}
}

// ToInt4 конвертирует int32 в pgtype.Int4
func ToInt4(val int32) pgtype.Int4 {
	if val == 0 {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: val, Valid: true}
}

// ToInt4Ptr конвертирует *int32 в pgtype.Int4
func ToInt4Ptr(val *int32) pgtype.Int4 {
	if val == nil || *val == 0 {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: *val, Valid: true}
}

// ToInt8 конвертирует int64 в pgtype.Int8
func ToInt8(val int64) pgtype.Int8 {
	if val == 0 {
		return pgtype.Int8{Valid: false}
	}
	return pgtype.Int8{Int64: val, Valid: true}
}

// ToInt8PtrPg конвертирует *int64 в pgtype.Int8
func ToInt8PtrPg(val *int64) pgtype.Int8 {
	if val == nil || *val == 0 {
		return pgtype.Int8{Valid: false}
	}
	return pgtype.Int8{Int64: *val, Valid: true}
}

// ToUUID конвертирует uuid.UUID в pgtype.UUID
func ToUUID(val uuid.UUID) pgtype.UUID {
	if val == uuid.Nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: val, Valid: true}
}

// ToUUIDPtr конвертирует *uuid.UUID в pgtype.UUID
func ToUUIDPtr(val *uuid.UUID) pgtype.UUID {
	if val == nil || *val == uuid.Nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *val, Valid: true}
}

// ToTimestamp конвертирует time.Time в pgtype.Timestamp
func ToTimestamp(t time.Time) pgtype.Timestamp {
	if t.IsZero() {
		return pgtype.Timestamp{Valid: false}
	}
	return pgtype.Timestamp{Time: t, Valid: true}
}

// ToTimestampPtr конвертирует *time.Time в pgtype.Timestamp
func ToTimestampPtr(t *time.Time) pgtype.Timestamp {
	if t == nil || t.IsZero() {
		return pgtype.Timestamp{Valid: false}
	}
	return pgtype.Timestamp{Time: *t, Valid: true}
}

// ToTimestamptz конвертирует time.Time в pgtype.Timestamptz
func ToTimestamptz(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// ToTimestamptzPtr конвертирует *time.Time в pgtype.Timestamptz
func ToTimestamptzPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil || t.IsZero() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// ToBool конвертирует bool в pgtype.Bool
func ToBool(val bool) pgtype.Bool {
	return pgtype.Bool{Bool: val, Valid: true}
}

// ToBoolPtr конвертирует *bool в pgtype.Bool
func ToBoolPtr(val *bool) pgtype.Bool {
	if val == nil {
		return pgtype.Bool{Valid: false}
	}
	return pgtype.Bool{Bool: *val, Valid: true}
}

// ToFloat4 конвертирует float32 в pgtype.Float4
func ToFloat4(val float32) pgtype.Float4 {
	if val == 0 {
		return pgtype.Float4{Valid: false}
	}
	return pgtype.Float4{Float32: val, Valid: true}
}

// ToFloat4Ptr конвертирует *float32 в pgtype.Float4
func ToFloat4Ptr(val *float32) pgtype.Float4 {
	if val == nil || *val == 0 {
		return pgtype.Float4{Valid: false}
	}
	return pgtype.Float4{Float32: *val, Valid: true}
}

// ToFloat8 конвертирует float64 в pgtype.Float8
func ToFloat8(val float64) pgtype.Float8 {
	if val == 0 {
		return pgtype.Float8{Valid: false}
	}
	return pgtype.Float8{Float64: val, Valid: true}
}

// ToFloat8Ptr конвертирует *float64 в pgtype.Float8
func ToFloat8Ptr(val *float64) pgtype.Float8 {
	if val == nil || *val == 0 {
		return pgtype.Float8{Valid: false}
	}
	return pgtype.Float8{Float64: *val, Valid: true}
}

// ToInt2 конвертирует int16 в pgtype.Int2
func ToInt2(val int16) pgtype.Int2 {
	if val == 0 {
		return pgtype.Int2{Valid: false}
	}
	return pgtype.Int2{Int16: val, Valid: true}
}

// ToInt2Ptr конвертирует *int16 в pgtype.Int2
func ToInt2Ptr(val *int16) pgtype.Int2 {
	if val == nil || *val == 0 {
		return pgtype.Int2{Valid: false}
	}
	return pgtype.Int2{Int16: *val, Valid: true}
}

// FromUUID конвертирует pgtype.UUID в uuid.UUID
func FromUUID(u pgtype.UUID) uuid.UUID {
	if !u.Valid {
		return uuid.Nil
	}
	return u.Bytes
}

// FromText конвертирует pgtype.Text в *string
func FromText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

// FromInt8 конвертирует pgtype.Int8 в *int64
func FromInt8(i pgtype.Int8) *int64 {
	if !i.Valid {
		return nil
	}
	return &i.Int64
}

// FromTimestamptz конвертирует pgtype.Timestamptz в *time.Time
func FromTimestamptz(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// ToNullString конвертирует *string в sql.NullString
func ToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

// ToNullInt64 конвертирует *int64 в sql.NullInt64
func ToNullInt64(i *int64) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: *i, Valid: true}
}

// ToNullTime конвертирует *time.Time в sql.NullTime
func ToNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func ToTextPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ToInt8Ptr(i int64) *int64 {
	return &i
}
