package mailing

import (
	"context"

	"mailing/internal/models"

	"github.com/google/uuid"
)

// Storage is an interface to interact with storage.
type Storage interface {
	// MigrateUp performs a database migration to the last available version.
	MigrateUp(ctx context.Context) error
	// GetClients returns all clients.
	GetClients(ctx context.Context) ([]*models.Client, error)
	// GetClientByID returns client by id.
	GetClientByID(ctx context.Context, id int64) (*models.Client, error)
	// SaveClient saves client with all his atributes in Storage.
	SaveClient(ctx context.Context, client *models.Client) error
	// UpdateClient applies given update to client from storage by given id.
	UpdateClient(ctx context.Context, id int64, update *models.ClientUpdate) error
	// DeleteClient deletes client from storage by given id.
	DeleteClient(ctx context.Context, id int64) error
	// GetMailings return all mailings.
	GetMailings(ctx context.Context) ([]*models.Mailing, error)
	// SaveMailing saves mailing with all it's atributes in Storage.
	SaveMailing(ctx context.Context, mailing *models.Mailing) error
	// UpdateMailing applies given update to mailing from storage by given id.
	UpdateMailing(ctx context.Context, id uuid.UUID, update *models.MailingUpdate) error
	// DeleteMailing deletes mailing from storage by given id.
	DeleteMailing(ctx context.Context, id uuid.UUID) error
	// MarkMailing marks mailing status as given one.
	MarkMailing(ctx context.Context, mailing *models.Mailing, status models.MailingStatus) error
	// GetPendingMailings returns all mailings that were not yet started.
	GetPendingMailings(ctx context.Context) ([]*models.Mailing, error)
	// SaveMessage saves message in Storage and returns it's id.
	SaveMessage(ctx context.Context, msg *models.Message) (int64, error)
	// MarkMessage marks message status as given one.
	MarkMessage(ctx context.Context, msg *models.Message, status models.SendStatus) error
	// CommonStatistic returns common statistic for mailings.
	CommonStatistic(ctx context.Context) ([]*models.MailingStats, error)
	// DetailedStatistic returns detailed statistic for given mailing.
	DetailedStatistic(ctx context.Context, mailingID uuid.UUID) (*models.DetailedMailingStats, error)
	// GetClientsForMailing gets clients which satisfy mailing's filter.
	GetClientsForMailing(ctx context.Context, mailing *models.Mailing) ([]*models.Client, error)
	// SaveStats saves stats of executed mailing.
	SaveStats(ctx context.Context, mailingStats *models.MailingStats) error
}

// MessageSender is an interface to send messages.
type MessageSender interface {
	// Send sends message via MessageSender.
	Send(ctx context.Context, msgID int64, clientPhone int64, text string) error
}
