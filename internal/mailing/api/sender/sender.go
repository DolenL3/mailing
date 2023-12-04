package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"mailing/internal/config"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Sender is MessageSender implementation via API.
type Sender struct {
	client *http.Client
	apiURL string
	config *config.SenderConfig
}

// New returns new MessageSender implementation via Sender.
func New(client *http.Client, config *config.SenderConfig) *Sender {
	return &Sender{
		client: client,
		// apiURL should be in config, parsed from json, as it is not a sensitive info
		// but for the sake of simplicity it is defined here.
		apiURL: "https://probe.fbrq.cloud/v1",
		config: config,
	}
}

type payloadSend struct {
	ID    int64  `json:"id"`
	Phone int64  `json:"phone"`
	Text  string `json:"text"`
}

// Send sends message via MessageSender.
func (s *Sender) Send(ctx context.Context, msgID int64, clientPhone int64, text string) error {
	l := zap.L()
	url := s.apiURL + "/send"
	url += fmt.Sprintf("/%d", msgID)

	// Form payload.
	payload := payloadSend{
		ID:    msgID,
		Phone: clientPhone,
		Text:  text,
	}
	payloadMarshaled, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "marshal payload")
	}

	// Form request.
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	s.client = &http.Client{Transport: tr}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payloadMarshaled))
	if err != nil {
		return errors.Wrap(err, "form request")
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.config.JWT))

	// Send request.
	l.Info(fmt.Sprintf("Sending request to %s...", url))
	resp, err := s.client.Do(req)
	if err != nil {
		l.Error("FAIL")
		return errors.Wrap(err, "send request")
	}
	defer resp.Body.Close()
	// Check response.
	if resp.StatusCode != http.StatusOK {
		l.Error("FAIL")
		l.Debug(fmt.Sprint("Status: ", resp.Status))
		l.Debug(fmt.Sprint(resp.Header))
		return errors.New("status is not 200")
	}
	l.Info("SUCCESS")
	return nil
}
