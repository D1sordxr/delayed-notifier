package httputil

import (
	"time"

	"github.com/oapi-codegen/runtime/types"
)

func ParseGenDateOnly(reqTime types.Date) (time.Time, error) {
	return time.Parse(time.DateOnly, reqTime.String())
}

func ParseGenTime(reqTime types.Date) (time.Time, error) {
	return time.Parse(time.DateTime, reqTime.String())
}
