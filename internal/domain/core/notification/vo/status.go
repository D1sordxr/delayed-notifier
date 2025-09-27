package vo

import (
	"github.com/D1sordxr/delayed-notifier/internal/domain/core/notification/errorx"
)

type Status uint

const (
	Pending Status = iota
	Sent
	Failed
	Declined
)

func (s Status) String() string {
	switch s {
	case Pending:
		return "pending"
	case Sent:
		return "sent"
	case Failed:
		return "failed"
	case Declined:
		return "declined"
	default:
		return "unknown"
	}
}

func ParseStatus(str string) (Status, error) {
	switch str {
	case "pending":
		return Pending, nil
	case "sent":
		return Sent, nil
	case "failed":
		return Failed, nil
	case "declined":
		return Declined, nil
	default:
		return Status(0), errorx.ErrInvalidStatus
	}
}
