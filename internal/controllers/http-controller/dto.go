package httpcontroller

import (
	"mailing/internal/models"
	"time"

	"github.com/google/uuid"
)

// Data transfer objects. For reference see models package.

type clientDTO struct {
	ID            int64  `json:"id"`
	PhoneNumber   int64  `json:"phoneNumber"`
	PhoneOperator int    `json:"phoneOperator"`
	Tag           string `json:"tag"`
	Timezone      string `json:"timezone"`
}

func clientToDTO(c *models.Client) *clientDTO {
	return &clientDTO{
		ID:            c.ID,
		PhoneNumber:   c.PhoneNumber,
		PhoneOperator: c.PhoneOperator,
		Tag:           c.Tag,
		Timezone:      c.Timezone,
	}
}

type clientUpdateDTO struct {
	PhoneNumber   int64  `json:"phoneNumber"`
	PhoneOperator int    `json:"phoneOperator"`
	Tag           string `json:"tag"`
	Timezone      string `json:"timezone"`
}

type mailingUpdateDTO struct {
	Text      string    `json:"text"`
	Filter    filterDTO `json:"filter"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
}

type filterDTO struct {
	PhoneOperator int    `json:"phoneOperator"`
	Tag           string `json:"tag"`
	Timezone      string `json:"timezone"`
}

var mailingStatus map[models.MailingStatus]string = map[models.MailingStatus]string{
	0: "pending", 1: "executing", 2: "done", 3: "canceled", 4: "failed", 5: "invalid", 6: "not a mailing",
}

type mailingDTO struct {
	ID        uuid.UUID  `json:"id"`
	Text      string     `json:"text"`
	Filter    *filterDTO `json:"filter"`
	StartTime string     `json:"startTime"`
	EndTime   string     `json:"endTime"`
	Status    string     `json:"status"`
}

func mailingToDTO(m *models.Mailing) *mailingDTO {
	return &mailingDTO{
		ID:   m.ID,
		Text: m.Text,
		Filter: &filterDTO{
			PhoneOperator: m.Filter.PhoneOperator,
			Tag:           m.Filter.Tag,
			Timezone:      m.Filter.Timezone,
		},
		StartTime: m.StartTime.String(),
		EndTime:   m.EndTime.String(),
		Status:    mailingStatus[m.Status],
	}
}

type mailingStatisticDTO struct {
	ID            uuid.UUID `json:"id"`
	Matches       int       `json:"matches"`
	Sent          int       `json:"sent"`
	Fails         int       `json:"fails"`
	StartTime     time.Time `json:"startTime"`
	TimeExecuting string    `json:"timeExecuting"`
}

func mailingStatisticToDTO(s *models.MailingStats) *mailingStatisticDTO {
	return &mailingStatisticDTO{
		ID:            s.ID,
		Matches:       s.Matches,
		Sent:          s.Sent,
		Fails:         s.Fails,
		StartTime:     s.StartTime,
		TimeExecuting: s.TimeExecuting.String(),
	}
}

var messageStatus map[models.SendStatus]string = map[models.SendStatus]string{0: "pending", 1: "success", 2: "failed"}

type messageDTO struct {
	ID        int64     `json:"id"`
	TimeStamp time.Time `json:"timeStamp"`
	MailingID uuid.UUID `json:"mailingID"`
	ClientID  int64     `json:"clientID"`
	Status    string    `json:"status"`
}

func messageToDTO(m *models.Message) *messageDTO {
	return &messageDTO{
		ID:        m.ID,
		TimeStamp: m.TimeStamp,
		MailingID: m.MailingID,
		ClientID:  m.ClientID,
		Status:    messageStatus[m.Status],
	}
}

type detailedMailingStatsDTO struct {
	CommonStats *mailingStatisticDTO `json:"commonStatistic"`
	Attributes  *mailingDTO          `json:"attributes"`
	Messages    []*messageDTO        `json:"messages"`
}

func detailedStatisticToDTO(d *models.DetailedMailingStats) *detailedMailingStatsDTO {
	messages := []*messageDTO{}
	for _, msg := range d.Messages {
		messages = append(messages, messageToDTO(&msg))
	}
	return &detailedMailingStatsDTO{
		CommonStats: mailingStatisticToDTO(&d.CommonStats),
		Attributes:  mailingToDTO(&d.Attributes),
		Messages:    messages,
	}
}
