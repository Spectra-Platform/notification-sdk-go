package notification

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type EmailVerificationService struct {
	client *Client
}

type EmailVerificationMethod string

const (
	EmailVerificationMethodCode EmailVerificationMethod = "code"
	EmailVerificationMethodLink EmailVerificationMethod = "link"

	DefaultEmailVerificationExpiresInSeconds = 600
)

type CreateEmailVerificationInput struct {
	RecipientEmail   string
	Method           EmailVerificationMethod
	TemplateID       string
	TemplateVersion  int
	ExpiresInSeconds int
	Metadata         map[string]string
	RedirectURI      string
	IdempotencyKey   string
}

type ConfirmEmailVerificationInput struct {
	VerificationID string
	Code           string
	LinkToken      string
}

type EmailVerificationChallenge struct {
	VerificationID string                  `json:"verification_id,omitempty"`
	Status         string                  `json:"status,omitempty"`
	Method         EmailVerificationMethod `json:"method,omitempty"`
	ExpiresAt      time.Time               `json:"expires_at,omitempty"`
}

type EmailVerificationProof struct {
	VerificationID string     `json:"verification_id,omitempty"`
	Status         string     `json:"status,omitempty"`
	VerifiedAt     *time.Time `json:"verified_at,omitempty"`
}

var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]+$`)

func (s *EmailVerificationService) Create(ctx context.Context, input CreateEmailVerificationInput) (*EmailVerificationChallenge, error) {
	if err := validateCreateEmailVerificationInput(input); err != nil {
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

	var challenge EmailVerificationChallenge
	err := s.client.doJSON(
		ctx,
		http.MethodPost,
		s.client.projectEmailVerificationsURL(),
		newCreateEmailVerificationRequest(input),
		map[string]string{"Idempotency-Key": idempotencyKey},
		&challenge,
	)
	if err != nil {
		return nil, err
	}

	return &challenge, nil
}

func (s *EmailVerificationService) Confirm(ctx context.Context, input ConfirmEmailVerificationInput) (*EmailVerificationProof, error) {
	verificationID := strings.TrimSpace(input.VerificationID)
	if verificationID == "" {
		return nil, newValidationError("VerificationID is required")
	}

	hasCode := strings.TrimSpace(input.Code) != ""
	hasLinkToken := strings.TrimSpace(input.LinkToken) != ""
	if hasCode == hasLinkToken {
		return nil, newValidationError("exactly one of Code or LinkToken is required")
	}

	var proof EmailVerificationProof
	err := s.client.doJSON(
		ctx,
		http.MethodPost,
		s.client.emailVerificationConfirmationURL(verificationID),
		newConfirmEmailVerificationRequest(input),
		nil,
		&proof,
	)
	if err != nil {
		return nil, err
	}

	return &proof, nil
}

type createEmailVerificationRequest struct {
	RecipientEmail   string            `json:"recipient_email"`
	Method           string            `json:"method"`
	TemplateID       string            `json:"template_id"`
	TemplateVersion  *int              `json:"template_version,omitempty"`
	ExpiresInSeconds int               `json:"expires_in_seconds"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	RedirectURI      string            `json:"redirect_uri,omitempty"`
}

type confirmEmailVerificationRequest struct {
	Code      string `json:"code,omitempty"`
	LinkToken string `json:"link_token,omitempty"`
}

func validateCreateEmailVerificationInput(input CreateEmailVerificationInput) error {
	if strings.TrimSpace(input.RecipientEmail) == "" {
		return newValidationError("RecipientEmail is required")
	}
	if strings.TrimSpace(input.TemplateID) == "" {
		return newValidationError("TemplateID is required")
	}
	method := defaultEmailVerificationMethod(input.Method)
	if method != EmailVerificationMethodCode && method != EmailVerificationMethodLink {
		return newValidationError("Method must be code or link")
	}
	expiresInSeconds := defaultEmailVerificationExpiresInSeconds(input.ExpiresInSeconds)
	if expiresInSeconds < 60 || expiresInSeconds > 86400 {
		return newValidationError("ExpiresInSeconds must be between 60 and 86400")
	}
	if input.TemplateVersion < 0 {
		return newValidationError("TemplateVersion must be greater than or equal to 0")
	}
	if method == EmailVerificationMethodCode && strings.TrimSpace(input.RedirectURI) != "" {
		return newValidationError("RedirectURI is only supported for link verification")
	}
	if method == EmailVerificationMethodLink && strings.TrimSpace(input.RedirectURI) == "" {
		return newValidationError("RedirectURI is required for link verification")
	}
	return nil
}

func validateIdempotencyKey(key string) error {
	if len(key) < 16 || len(key) > 128 {
		return newValidationError("IdempotencyKey must be between 16 and 128 characters")
	}
	if !idempotencyKeyPattern.MatchString(key) {
		return newValidationError("IdempotencyKey may contain only letters, numbers, dot, underscore, colon, and hyphen")
	}
	return nil
}

func newCreateEmailVerificationRequest(input CreateEmailVerificationInput) createEmailVerificationRequest {
	request := createEmailVerificationRequest{
		RecipientEmail:   strings.TrimSpace(input.RecipientEmail),
		Method:           string(defaultEmailVerificationMethod(input.Method)),
		TemplateID:       strings.TrimSpace(input.TemplateID),
		ExpiresInSeconds: defaultEmailVerificationExpiresInSeconds(input.ExpiresInSeconds),
		Metadata:         input.Metadata,
		RedirectURI:      strings.TrimSpace(input.RedirectURI),
	}
	if input.TemplateVersion > 0 {
		request.TemplateVersion = &input.TemplateVersion
	}
	return request
}

func newConfirmEmailVerificationRequest(input ConfirmEmailVerificationInput) confirmEmailVerificationRequest {
	return confirmEmailVerificationRequest{
		Code:      strings.TrimSpace(input.Code),
		LinkToken: strings.TrimSpace(input.LinkToken),
	}
}

func defaultEmailVerificationMethod(method EmailVerificationMethod) EmailVerificationMethod {
	if method == "" {
		return EmailVerificationMethodCode
	}
	return method
}

func defaultEmailVerificationExpiresInSeconds(expiresInSeconds int) int {
	if expiresInSeconds == 0 {
		return DefaultEmailVerificationExpiresInSeconds
	}
	return expiresInSeconds
}

func generateIdempotencyKey() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}

	return "sdk:" + hex.EncodeToString(bytes[:]), nil
}
