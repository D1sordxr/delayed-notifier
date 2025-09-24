package params

import (
	"time"
	"wb-tech-l3/internal/domain/core/notification/vo"

	"github.com/google/uuid"
)

type UpdateNotificationStatusParams struct {
	ID       uuid.UUID  `json:"id"`
	Status   vo.Status  `json:"status"`
	Attempts int16      `json:"attempts"`
	SentAt   *time.Time `json:"sent_at"`
}
