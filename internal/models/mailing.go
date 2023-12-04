package models

import (
	"time"

	"github.com/google/uuid"
)

// Mailing is a struct that represents mailing object.
type Mailing struct {
	ID        uuid.UUID     `db:"id"`
	Text      string        `db:"text"`
	Filter    *Filter       `db:"-"`
	StartTime time.Time     `db:"start_time"`
	EndTime   time.Time     `db:"end_time"`
	Status    MailingStatus `db:"status"`
}

type MailingStatus int

const (
	// MailingStatusPending is mailing's status if mailing did not start yet.
	MailingStatusPending MailingStatus = 0
	// MailingStatusExecuting is mailing's status if mailing is started and running.
	MailingStatusExecuting MailingStatus = 1
	// MailingStatusDone is mailing's status if mailing is done.
	MailingStatusDone MailingStatus = 2
	// MailingStatusCanceled is mailing's status if mailing was canceled before completion.
	MailingStatusCanceled MailingStatus = 3
	// MailingStatusFailed is mailing's status if mailing failed.
	MailingStatusFailed MailingStatus = 4
	// MailingStatusInvalid is mailing's status if mailing's atributes were considered invalid.
	MailingStatusInvalid MailingStatus = 5
	// MailingStatusNotAMailing means that object is not a mailing, but a dependency for sending standalone messages.
	MailingStatusNotAMailing MailingStatus = 6
)

// MailingUpdate is a struct with updates which should be applied to Mailing.
type MailingUpdate struct {
	Text      string
	Filter    *Filter
	StartTime time.Time
	EndTime   time.Time
}

// MailingStats is a struct with common mailing statistic.
type MailingStats struct {
	// ID is mailing's id
	ID uuid.UUID `db:"mailing_id"`
	// Matches is the amount of clients, that were matched to filter during mailing execution.
	Matches int `db:"matches"`
	// Sent is the amount of messages, that tried to send, no matter the success.
	Sent int `db:"sent"`
	// Fails is the amount of messages, that tried to send, but failed.
	Fails int `db:"fails"`
	// StartTime is the mailing start time.
	StartTime time.Time `db:"start_time"`
	// TimeExecuting is the duration of executing the mailing.
	TimeExecuting time.Duration `db:"time_executing"`
}

// DetailedStats is a struct with detailed mailing statistic.
type DetailedMailingStats struct {
	CommonStats MailingStats
	Attributes  Mailing
	Messages    []Message
}
