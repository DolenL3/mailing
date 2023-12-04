package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratePgx "github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"

	// "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"mailing/internal/config"
	"mailing/internal/models"
)

// Postgres is a Storage implementation via pgx postgresql.
type Postgres struct {
	db     *pgxpool.Pool
	config *config.DBConfig
}

// New returns a Storage implementation via pgx postgresql.
func New(db *pgxpool.Pool, config *config.DBConfig) *Postgres {
	return &Postgres{
		db:     db,
		config: config,
	}
}

// MigrateUp performs a database migration to the last available version.
func (p *Postgres) MigrateUp(ctx context.Context) error {
	logger := zap.L()
	pg := &migratePgx.Postgres{}
	driver, err := pg.Open(fmt.Sprintf("postgresql://%s:%s@%s/%s", p.config.User, p.config.Password, p.config.Host, p.config.DBName))
	if err != nil {
		return errors.Wrap(err, "opening connection")
	}
	m, err := migrate.NewWithDatabaseInstance(
		p.config.MigrationURL,
		"pgx", driver)
	if err != nil {
		return errors.Wrap(err, "get migrate instance")
	}
	err = m.Up()
	if err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return errors.Wrap(err, "migrate up")
		}
		logger.Info("no change during migration")
		return nil
	}
	logger.Info("database migrated successfully")
	return nil
}

// GetClients returns all clients.
func (p *Postgres) GetClients(ctx context.Context) ([]*models.Client, error) {
	query := `
	SELECT *
	FROM client
	`
	rows, err := p.db.Query(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "select from client")
	}
	clients, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[models.Client])
	if err != nil {
		return nil, errors.Wrap(err, "collect rows")
	}
	return clients, nil
}

// GetClientByID returns client by id.
func (p *Postgres) GetClientByID(ctx context.Context, id int64) (*models.Client, error) {
	query := `
	SELECT *
	FROM client
	WHERE id = $1
	`
	row, err := p.db.Query(ctx, query, id)
	if err != nil {
		return nil, errors.Wrap(err, "select from client")
	}
	client, err := pgx.CollectOneRow(row, pgx.RowToAddrOfStructByName[models.Client])
	if err != nil {
		return nil, errors.Wrap(err, "collect row")
	}
	return client, nil
}

// SaveClient saves client with all his atributes in Storage.
func (p *Postgres) SaveClient(ctx context.Context, client *models.Client) error {
	l := zap.L()
	query := `
	INSERT INTO client(phone_number, phone_operator, tag, timezone)
	VALUES ($1, $2, $3, $4)
	`
	_, err := p.db.Exec(ctx, query, client.PhoneNumber, client.PhoneOperator, client.Tag, client.Timezone)
	if err != nil {
		return errors.Wrap(err, "insert into client")
	}
	l.Info(fmt.Sprintf("Saved client: %d", client.ID))
	return nil
}

// UpdateClient applies given update to client from storage by given id.
func (p *Postgres) UpdateClient(ctx context.Context, id int64, update *models.ClientUpdate) error {
	l := zap.L()
	query := `
	UPDATE client
	SET %s
	WHERE id = @id
	`
	updates := []string{}
	if update.PhoneNumber != 0 {
		updates = append(updates, "phone_number = @phoneNumber")
	}
	if update.PhoneOperator != 0 {
		updates = append(updates, "phone_operator = @phoneOperator")
	}
	if update.Tag != "" {
		updates = append(updates, "tag = @tag")
	}
	if update.Timezone != "" {
		updates = append(updates, "timezone = @timezone")
	}

	if len(updates) > 0 {
		query = fmt.Sprintf(query, strings.Join(updates, ", "))
	}
	args := pgx.NamedArgs{
		"phoneNumber":   update.PhoneNumber,
		"phoneOperator": update.PhoneOperator,
		"tag":           update.Tag,
		"timezone":      update.Timezone,
		"id":            id,
	}

	_, err := p.db.Exec(ctx, query, args)
	if err != nil {
		return errors.Wrap(err, "update client")
	}
	l.Info(fmt.Sprintf("Updated client: %d", id))
	return nil
}

// DeleteClient deletes client from storage by given id.
func (p *Postgres) DeleteClient(ctx context.Context, id int64) error {
	query := `
	DELETE FROM client
	WHERE id = $1
	`
	_, err := p.db.Exec(ctx, query, id)
	if err != nil {
		return errors.Wrap(err, "delete from client")
	}
	return nil
}

// GetMailings return all mailings.
func (p *Postgres) GetMailings(ctx context.Context) ([]*models.Mailing, error) {
	query := `
	SELECT m.id, m.text, m.start_time, m.end_time, m.status, f.phone_operator, f.tag, f.timezone
	FROM mailing m , mailing_filter f
	WHERE m.id = f.mailing_id
	`
	rows, err := p.db.Query(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "select from mailing")
	}
	defer rows.Close()
	mailings := []*models.Mailing{}
	for rows.Next() {
		var id uuid.UUID
		var text, tag, timezone string
		var startTime, endTime time.Time
		var phoneOperator int
		var status models.MailingStatus
		err := rows.Scan(&id, &text, &startTime, &endTime, &status, &phoneOperator, &tag, &timezone)
		if err != nil {
			return nil, errors.Wrap(err, "scan values")
		}
		mailings = append(mailings, &models.Mailing{
			ID:        id,
			Text:      text,
			StartTime: startTime,
			EndTime:   endTime,
			Filter: &models.Filter{
				PhoneOperator: phoneOperator,
				Tag:           tag,
				Timezone:      timezone,
			},
			Status: status,
		})
	}
	return mailings, nil
}

// SaveMailing saves mailing with all it's atributes in Storage.
func (p *Postgres) SaveMailing(ctx context.Context, mailing *models.Mailing) error { // TODO complex sql query
	query := `
	INSERT INTO mailing
	VALUES ($1, $2, $3, $4, $5)
	`
	_, err := p.db.Exec(ctx, query, mailing.ID, mailing.Text, mailing.StartTime, mailing.EndTime, int(mailing.Status))
	if err != nil {
		return errors.Wrap(err, "insert into mailing")
	}
	query = `
	INSERT INTO mailing_filter
	VALUES ($1, $2, $3, $4)
	`
	_, err = p.db.Exec(ctx, query, mailing.ID, mailing.Filter.PhoneOperator, mailing.Filter.Tag, mailing.Filter.Timezone)
	if err != nil {
		return errors.Wrap(err, "insert into mailing_filter")
	}
	return nil
}

// UpdateMailing applies given update to mailing from storage by given id.
func (p *Postgres) UpdateMailing(ctx context.Context, id uuid.UUID, update *models.MailingUpdate) error { // TODO complex sql query
	query := `
	UPDATE mailing
	SET %s
	WHERE id = @id
	`
	updates := []string{}
	if !update.StartTime.IsZero() {
		updates = append(updates, "start_time = @startTime")
	}
	if !update.EndTime.IsZero() {
		updates = append(updates, "end_time = @endTime")
	}
	if update.Text != "" {
		updates = append(updates, "text = @text")
	}
	args := pgx.NamedArgs{
		"startTime": update.StartTime,
		"endTime":   update.EndTime,
		"text":      update.Text,
		"id":        id,
	}

	if len(updates) > 0 {
		query = fmt.Sprintf(query, strings.Join(updates, ", "))
		_, err := p.db.Exec(ctx, query, args)
		if err != nil {
			return errors.Wrap(err, "update mailing")
		}
	}
	query = `
	UPDATE mailing_filter
	SET %s
	WHERE mailing_id = @id
	`
	updatesFilter := []string{}
	if update.Filter != nil {
		if update.Filter.PhoneOperator != 0 {
			updatesFilter = append(updatesFilter, "phone_operator = @phoneOperator")
		}
		if update.Filter.Tag != "" {
			updatesFilter = append(updatesFilter, "tag = @tag")
		}
		if update.Filter.Timezone != "" {
			updatesFilter = append(updatesFilter, "timezone = @timezone")
		}
	}
	args = pgx.NamedArgs{
		"phoneOperator": update.Filter.PhoneOperator,
		"tag":           update.Filter.Tag,
		"timezone":      update.Filter.Timezone,
		"id":            id,
	}
	if len(updatesFilter) > 0 {
		query = fmt.Sprintf(query, strings.Join(updatesFilter, ", "))
		_, err := p.db.Exec(ctx, query, args)
		if err != nil {
			return errors.Wrap(err, "update mailing_filter")
		}
	}
	return nil
}

// DeleteMailing deletes mailing from storage by given id.
func (p *Postgres) DeleteMailing(ctx context.Context, id uuid.UUID) error {
	query := `
	DELETE FROM mailing
	WHERE id = $1
	`
	_, err := p.db.Exec(ctx, query, id)
	if err != nil {
		return errors.Wrap(err, "delete from mailing")
	}
	return nil
}

// MarkMailing marks mailing status as given one.
func (p *Postgres) MarkMailing(ctx context.Context, mailing *models.Mailing, status models.MailingStatus) error {
	query := `
	UPDATE mailing
	SET status = $1
	WHERE id = $2
	`
	_, err := p.db.Exec(ctx, query, int(status), mailing.ID)
	if err != nil {
		return errors.Wrap(err, "update mailing")
	}
	return nil
}

// GetPendingMailings returns all mailings that were not yet started.
func (p *Postgres) GetPendingMailings(ctx context.Context) ([]*models.Mailing, error) {
	query := `
	SELECT m.id, m.text, m.start_time, m.end_time, f.phone_operator, f.tag, f.timezone
	FROM mailing m , mailing_filter f
	WHERE m.id = f.mailing_id AND m.status = $1
	`
	rows, err := p.db.Query(ctx, query, models.MailingStatusPending)
	if err != nil {
		return nil, errors.Wrap(err, "select from mailing")
	}
	defer rows.Close()
	mailings := []*models.Mailing{}
	for rows.Next() {
		var id uuid.UUID
		var text, tag, timezone string
		var startTime, endTime time.Time
		var phoneOperator int
		err := rows.Scan(&id, &text, &startTime, &endTime, &phoneOperator, &tag, &timezone)
		if err != nil {
			return nil, errors.Wrap(err, "scan values")
		}
		mailings = append(mailings, &models.Mailing{
			ID:        id,
			Text:      text,
			StartTime: startTime,
			EndTime:   endTime,
			Filter: &models.Filter{
				PhoneOperator: phoneOperator,
				Tag:           tag,
				Timezone:      timezone,
			},
		})
	}
	return mailings, nil
}

// SaveMessage saves message in Storage.
func (p *Postgres) SaveMessage(ctx context.Context, msg *models.Message) (int64, error) {
	query := `
	INSERT INTO message(time_stamp, mailing_id, client_id, status)
	VALUES ($1, $2, $3, $4)
	RETURNING id
	`
	var id int64
	err := p.db.QueryRow(ctx, query, msg.TimeStamp, msg.MailingID, msg.ClientID, int(msg.Status)).Scan(&id)
	if err != nil {
		return 0, errors.Wrap(err, "insert into message")
	}
	return id, nil
}

// MarkMessage marks message status as given one.
func (p *Postgres) MarkMessage(ctx context.Context, msg *models.Message, status models.SendStatus) error {
	query := `
	UPDATE message
	SET status = $1
	WHERE id = $2
	`
	_, err := p.db.Exec(ctx, query, int(status), msg.ID)
	if err != nil {
		return errors.Wrap(err, "update message")
	}
	return nil
}

// CommonStatistic returns common statistic for mailings.
func (p *Postgres) CommonStatistic(ctx context.Context) ([]*models.MailingStats, error) {
	query := `
	SELECT *
	FROM mailing_stats
	ORDER BY start_time DESC
	`
	rows, err := p.db.Query(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "select from mailing_stats")
	}
	mailingStats, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[models.MailingStats])
	if err != nil {
		return nil, errors.Wrap(err, "collect rows")
	}
	return mailingStats, nil
}

// DetailedStatistic returns detailed statistic for given mailing.
func (p *Postgres) DetailedStatistic(ctx context.Context, mailingID uuid.UUID) (*models.DetailedMailingStats, error) {
	query := `
	SELECT *
	FROM mailing_stats
	WHERE mailing_id = $1
	`
	rows, err := p.db.Query(ctx, query, mailingID)
	if err != nil {
		return nil, errors.Wrap(err, "select from mailing_stats")
	}
	mailingStats, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.MailingStats])
	if err != nil {
		return nil, errors.Wrap(err, "collect row mailing stats")
	}
	query = `
	SELECT m.id, m.text, m.start_time, m.end_time, m.status, f.phone_operator, f.tag, f.timezone
	FROM mailing m , mailing_filter f
	WHERE m.id = f.mailing_id AND m.id = $1
	`
	var id uuid.UUID
	var text, tag, timezone string
	var startTime, endTime time.Time
	var phoneOperator int
	var status models.MailingStatus
	err = p.db.QueryRow(ctx, query, mailingID).Scan(&id, &text, &startTime, &endTime, &status, &phoneOperator, &tag, &timezone)
	if err != nil {
		return nil, errors.Wrap(err, "select from mailing")
	}
	attributes := models.Mailing{
		ID:        id,
		Text:      text,
		StartTime: startTime,
		EndTime:   endTime,
		Filter: &models.Filter{
			PhoneOperator: phoneOperator,
			Tag:           tag,
			Timezone:      timezone,
		},
		Status: status,
	}
	query = `
	SELECT *
	FROM message
	WHERE mailing_id = $1
	ORDER BY status DESC
	`
	rows, err = p.db.Query(ctx, query, mailingID)
	if err != nil {
		return nil, errors.Wrap(err, "select from message")
	}
	messages, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Message])
	if err != nil {
		return nil, errors.Wrap(err, "collect rows messages")
	}
	return &models.DetailedMailingStats{
		CommonStats: mailingStats,
		Attributes:  attributes,
		Messages:    messages,
	}, nil
}

// GetClientsForMailing gets clients which satisfy mailing's filter.
func (p *Postgres) GetClientsForMailing(ctx context.Context, mailing *models.Mailing) ([]*models.Client, error) {
	query := `
	SELECT *
	FROM client

	`
	filters := []string{}
	if mailing.Filter != nil {
		if mailing.Filter.PhoneOperator != 0 {
			filters = append(filters, "phone_operator = @phoneOperator")
		}
		if mailing.Filter.Tag != "" {
			filters = append(filters, "tag = @tag")
		}
		if mailing.Filter.Timezone != "" {
			filters = append(filters, "timezone = @timezone")
		}

		if len(filters) > 0 {
			query += `WHERE `
			query += strings.Join(filters, ` AND `)
		}
	}

	args := pgx.NamedArgs{
		"phoneOperator": mailing.Filter.PhoneOperator,
		"tag":           mailing.Filter.Tag,
		"timezone":      mailing.Filter.Timezone,
	}
	rows, err := p.db.Query(ctx, query, args)
	if err != nil {
		return nil, errors.Wrap(err, "select from client")
	}
	clients, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[models.Client])
	if err != nil {
		return nil, errors.Wrap(err, "collect rows")
	}
	return clients, nil
}

// SaveStats saves stats of executed mailing.
func (p *Postgres) SaveStats(ctx context.Context, mailingStats *models.MailingStats) error {
	query := `
	INSERT INTO mailing_stats
	VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := p.db.Exec(ctx, query, mailingStats.ID, mailingStats.Matches, mailingStats.Sent,
		mailingStats.Fails, mailingStats.StartTime, mailingStats.TimeExecuting)
	if err != nil {
		return errors.Wrap(err, "insert into mailing_stats")
	}
	return nil
}
