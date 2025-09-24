package vo

import (
	"wb-tech-l3/internal/domain/core/notification/errorx"
)

type Channel uint

const (
	Email Channel = iota
	Telegram
	SMS
)

func (c Channel) String() string {
	switch c {
	case Email:
		return "email"
	case Telegram:
		return "telegram"
	case SMS:
		return "sms"
	default:
		return "unknown"
	}
}

func ParseChannel(str string) (Channel, error) {
	switch str {
	case "email":
		return Email, nil
	case "telegram":
		return Telegram, nil
	case "sms":
		return SMS, nil
	default:
		return Channel(0), errorx.ErrInvalidChannel
	}
}
