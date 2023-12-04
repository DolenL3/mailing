package models

import (
	"time"

	"github.com/google/uuid"
)

// Message is a message in mailing.
type Message struct {
	ID        int64      `db:"id"`
	TimeStamp time.Time  `db:"time_stamp"`
	MailingID uuid.UUID  `db:"mailing_id"`
	ClientID  int64      `db:"client_id"`
	Status    SendStatus `db:"status"`
}

type SendStatus int

const (
	// SendStatusPending is a message's status if message is in the process of sending.
	SendStatusPending SendStatus = 0
	// SendStatusSuccess is a message's status if message was sent successfully.
	SendStatusSuccess SendStatus = 1
	// SendStatusFailed is a message's status if a error occured while sending message.
	SendStatusFailed SendStatus = 2
)
