package rabbitmq

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// Message is what travels through the broker: only the id. The dispatcher
// reads the notification itself from the database, which stays the source
// of truth for its content and status (e.g. a cancellation after publishing).
type Message struct {
	ID uuid.UUID `json:"id"`
}

func EncodeMessage(id uuid.UUID) ([]byte, error) {
	return json.Marshal(Message{ID: id})
}

func DecodeMessage(body []byte) (Message, error) {
	var msg Message
	if err := json.Unmarshal(body, &msg); err != nil {
		return Message{}, fmt.Errorf("decode message: %w", err)
	}
	if msg.ID == uuid.Nil {
		return Message{}, errors.New("decode message: empty id")
	}

	return msg, nil
}
