package mailing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"mailing/internal/models"
)

// MailingService is a service with business logic.
type MailingService struct {
	mu            sync.Mutex
	Storage       Storage
	MessageSender MessageSender
}

// New creates new MailingService.
func New(storage Storage, messageSender MessageSender) *MailingService {
	return &MailingService{
		Storage:       storage,
		MessageSender: messageSender,
	}
}

// StartFetchingMailings starts fetching mailings, that need to be started.
func (m *MailingService) StartFetchingMailings(ctx context.Context) error {
	l := zap.L()
	l.Info("Started fetching mailings")
	wg := &sync.WaitGroup{}
	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()
		case <-ticker.C:
			mailings, err := m.Storage.GetPendingMailings(ctx)
			if err != nil {
				l.Error(fmt.Sprintf("FAIL: get pending mailings\nError: %v", err))
				continue
			}
			for _, mailing := range mailings {
				if mailing.StartTime.Compare(time.Now().UTC()) <= 0 {
					wg.Add(1)
					l.Info(fmt.Sprintf("mailing %v fetched, starting...", mailing.ID))
					go func(mailing *models.Mailing) {
						defer wg.Done()
						if time.Now().After(mailing.EndTime) {
							l.Warn(fmt.Sprintf("End time of mailing exceeds current time, omitting mailing...\nMailing: %v", mailing.ID))
							err := m.Storage.MarkMailing(ctx, mailing, models.MailingStatusInvalid)
							if err != nil {
								l.Error(fmt.Sprintf("FAIL: mark mailing status as invalid\nMailing: %v; Error: %v", mailing.ID, err))
							}
							return
						}
						ctx, cancel := context.WithDeadline(ctx, mailing.EndTime)
						defer cancel()
						l.Info(fmt.Sprintf("Started mailing on: %v", mailing.ID))
						err := m.mail(ctx, mailing)
						if err != nil {
							l.Error(fmt.Sprintf("FAIL: mail on %v\nError: %v", mailing.ID, err))
							return
						}
						l.Info(fmt.Sprintf("Finished mailing on: %v", mailing.ID))
					}(mailing)
				}
			}
		}
	}
}

// Mail starts given mailing.
func (m *MailingService) mail(ctx context.Context, mailing *models.Mailing) error {
	l := zap.L()
	err := m.Storage.MarkMailing(ctx, mailing, models.MailingStatusExecuting)
	if err != nil {
		return errors.Wrap(err, "mark mailing status as executing")
	}
	clients, err := m.Storage.GetClientsForMailing(ctx, mailing)
	if err != nil {
		nestedErr := m.Storage.MarkMailing(ctx, mailing, models.MailingStatusFailed)
		if nestedErr != nil {
			l.Error(fmt.Sprintf("FAIL: mark mailing status as failed\nMailing: %v; Error: %v", mailing.ID, nestedErr))
		}
		return errors.Wrap(err, "get clients for mailing")
	}
	fails := 0
	wg := &sync.WaitGroup{}
	start := time.Now()
	for count, client := range clients {
		select {
		// ctx.Done is called when context deadline is exceeded or if cancel() is called on parent context.
		case <-ctx.Done():
			wg.Wait()
			err = m.Storage.MarkMailing(ctx, mailing, models.MailingStatusCanceled)
			if err != nil {
				l.Error(fmt.Sprintf("FAIL: mark mailing as canceled\nMailing: %v; Error: %v", mailing.ID, err))
			}
			err = m.Storage.SaveStats(ctx, &models.MailingStats{
				ID:            mailing.ID,
				Fails:         fails,
				StartTime:     start,
				TimeExecuting: time.Since(start),
				Sent:          count + 1,
				Matches:       len(clients),
			})
			if err != nil {
				l.Error("Couldn't save stats of mailing")
			}
			return ctx.Err()
		default:
			wg.Add(1)
			go func(client *models.Client) {
				defer wg.Done()
				msg := newMessage(mailing.ID, client.ID)
				msg.ID, err = m.Storage.SaveMessage(ctx, msg)
				if err != nil {
					l.Error(fmt.Sprintf("FAIL: could not save message in storage\nMessage: %d; Client: %d; Error: %v\n",
						msg.ID, client.ID, err))
					m.mu.Lock()
					fails++
					m.mu.Unlock()
					return
				}
				for try := 0; try < _sendRetry; try++ {
					err = m.MessageSender.Send(ctx, msg.ID, client.PhoneNumber, mailing.Text)
					if err != nil {
						l.Info(fmt.Sprintf("Send failed, retrying in 1s...\nMessage: %d; Client: %d; Error: %v\n", msg.ID, client.ID, err))
						time.Sleep(time.Second)
						continue
					}
					break
				}
				if err != nil {
					l.Error(fmt.Sprintf("FAIL: could not send message in %d tries\nMessage: %d; Client: %d; Error: %v",
						_sendRetry, msg.ID, client.ID, err))
					m.mu.Lock()
					fails++
					m.mu.Unlock()
					err = m.Storage.MarkMessage(ctx, msg, models.SendStatusFailed)
					if err != nil {
						l.Error(fmt.Sprintf("FAIL: mark message as failed\nMessage: %d; Client: %d; Error: %v", msg.ID, client.ID, err))
					}
					return
				}
				err = m.Storage.MarkMessage(ctx, msg, models.SendStatusSuccess)
				if err != nil {
					l.Error(fmt.Sprintf("FAIL: mark message as success\nMessage: %d; Client: %d; Error: %v", msg.ID, client.ID, err))
				}
			}(client)
		}
	}
	wg.Wait()
	err = m.Storage.MarkMailing(ctx, mailing, models.MailingStatusDone)
	if err != nil {
		l.Error(fmt.Sprintf("FAIL: mark mailing as done\nMailing: %v; Error: %v", mailing.ID, err))
	}
	err = m.Storage.SaveStats(ctx, &models.MailingStats{
		ID:            mailing.ID,
		Fails:         fails,
		StartTime:     start,
		TimeExecuting: time.Since(start),
		Sent:          len(clients),
		Matches:       len(clients),
	})
	if err != nil {
		l.Error("Couldn't save stats of mailing")
	}
	return nil
}

func newMessage(mailingID uuid.UUID, clientID int64) *models.Message {
	return &models.Message{
		TimeStamp: time.Now(),
		Status:    models.SendStatusPending,
		MailingID: mailingID,
		ClientID:  clientID,
	}
}
