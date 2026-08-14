package notification

import (
	"context"
	"net/http"
	"strings"
	"time"
)

type EmailService struct {
	client        *Client
	Verifications *EmailVerificationService
}

type SendEmailInput struct {
	To             string
	Subject        string
	Text           string
	HTML           string
	Metadata       map[string]string
	IdempotencyKey string
}

type EmailDeliveryRequest struct {
	RequestID string    `json:"request_id,omitempty"`
	Status    string    `json:"status,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type sendEmailRequest struct {
	RecipientEmail string            `json:"recipient_email"`
	Subject        string            `json:"subject"`
	Text           string            `json:"text"`
	HTML           string            `json:"html,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

func (s *EmailService) Send(ctx context.Context, input SendEmailInput) (*EmailDeliveryRequest, error) {
	if err := validateSendEmailInput(input); err != nil {
		return nil, err
	}

	idempotencyKey := strings.TrimSpace(input.IdempotencyKey)
	if idempotencyKey == "" {
		generatedKey, err := generateIdempotencyKey()
		if err != nil {
			return nil, &Error{
				Code:    "IDEMPOTENCY_KEY_GENERATION_FAILED",
				Message: err.Error(),
			}
		}
		idempotencyKey = generatedKey
	}
	if err := validateIdempotencyKey(idempotencyKey); err != nil {
		return nil, err
	}

	var request EmailDeliveryRequest
	err := s.client.doJSON(
		ctx,
		http.MethodPost,
		s.client.projectEmailDeliveryRequestsURL(),
		newSendEmailRequest(input),
		map[string]string{"Idempotency-Key": idempotencyKey},
		&request,
	)
	if err != nil {
		return nil, err
	}

	return &request, nil
}

func validateSendEmailInput(input SendEmailInput) error {
	if strings.TrimSpace(input.To) == "" {
		return newValidationError("To is required")
	}
	if strings.TrimSpace(input.Subject) == "" {
		return newValidationError("Subject is required")
	}
	if strings.ContainsAny(input.Subject, "\r\n") {
		return newValidationError("Subject cannot contain CR or LF")
	}
	if strings.TrimSpace(input.Text) == "" {
		return newValidationError("Text is required")
	}
	return nil
}

func newSendEmailRequest(input SendEmailInput) sendEmailRequest {
	return sendEmailRequest{
		RecipientEmail: strings.TrimSpace(input.To),
		Subject:        strings.TrimSpace(input.Subject),
		Text:           strings.TrimSpace(input.Text),
		HTML:           strings.TrimSpace(input.HTML),
		Metadata:       input.Metadata,
	}
}
