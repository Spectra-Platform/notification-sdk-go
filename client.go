package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultBaseURL = "https://delivery.spectra.kr"

	defaultTimeout   = 10 * time.Second
	defaultUserAgent = "spectra-notification-go/0.1.2"
)

type Environment string

const (
	EnvironmentTest Environment = "test"
	EnvironmentLive Environment = "live"
)

type Config struct {
	ProjectID       string
	ProjectAPIToken string
	Environment     Environment
	BaseURL         string
	HTTPClient      *http.Client
	Timeout         time.Duration
	UserAgent       string
}

type Client struct {
	projectID       string
	projectAPIToken string
	environment     Environment
	baseURL         string
	httpClient      *http.Client
	userAgent       string

	Email              *EmailService
	Delivery           *DeliveryService
	EmailVerifications *EmailVerificationService
}

type DeliveryService struct {
	EmailVerifications *EmailVerificationService
}

func NewClient(config Config) (*Client, error) {
	projectID := strings.TrimSpace(config.ProjectID)
	if projectID == "" {
		return nil, newValidationError("ProjectID is required")
	}

	projectAPIToken := strings.TrimSpace(config.ProjectAPIToken)
	if projectAPIToken == "" {
		return nil, newValidationError("ProjectAPIToken is required")
	}

	environment := config.Environment
	if environment == "" {
		environment = EnvironmentTest
	}
	if environment != EnvironmentTest && environment != EnvironmentLive {
		return nil, newValidationError("Environment must be either test or live")
	}

	baseURL := strings.TrimSpace(config.BaseURL)
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	parsedBaseURL, err := url.ParseRequestURI(baseURL)
	if err != nil || parsedBaseURL.Scheme == "" || parsedBaseURL.Host == "" {
		return nil, newValidationError("BaseURL must be an absolute URL")
	}
	baseURL = strings.TrimRight(baseURL, "/")

	httpClient := config.HTTPClient
	if httpClient == nil {
		timeout := config.Timeout
		if timeout <= 0 {
			timeout = defaultTimeout
		}
		httpClient = &http.Client{Timeout: timeout}
	}

	userAgent := strings.TrimSpace(config.UserAgent)
	if userAgent == "" {
		userAgent = defaultUserAgent
	}

	client := &Client{
		projectID:       projectID,
		projectAPIToken: projectAPIToken,
		environment:     environment,
		baseURL:         baseURL,
		httpClient:      httpClient,
		userAgent:       userAgent,
	}
	emailVerifications := &EmailVerificationService{client: client}
	client.Email = &EmailService{client: client, Verifications: emailVerifications}
	client.EmailVerifications = emailVerifications
	client.Delivery = &DeliveryService{EmailVerifications: emailVerifications}

	return client, nil
}

func (c *Client) projectEmailDeliveryRequestsURL() string {
	return fmt.Sprintf(
		"%s/platform/v1/projects/%s/email/delivery-requests",
		c.baseURL,
		url.PathEscape(c.projectID),
	)
}

func (c *Client) doJSON(ctx context.Context, method string, requestURL string, input any, headers map[string]string, output any) error {
	if ctx == nil {
		ctx = context.Background()
	}

	requestBody, err := json.Marshal(input)
	if err != nil {
		return &Error{
			Code:    "REQUEST_ENCODING_FAILED",
			Message: err.Error(),
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(requestBody))
	if err != nil {
		return &Error{
			Code:    "REQUEST_CREATION_FAILED",
			Message: err.Error(),
			Err:     err,
		}
	}

	req.Header.Set("Authorization", "Bearer "+c.projectAPIToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Spectra-Project-Id", c.projectID)
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return &Error{
				Code:    "REQUEST_CANCELLED",
				Message: ctxErr.Error(),
				Err:     ctxErr,
			}
		}
		return newNetworkUnavailableError(err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return newMalformedResponseError(resp.StatusCode, resp.Header.Get("X-Request-ID"), err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseAPIError(resp, responseBody)
	}

	return parseDataEnvelope(resp.StatusCode, resp.Header.Get("X-Request-ID"), responseBody, output)
}

func (c *Client) projectEmailVerificationsURL() string {
	return fmt.Sprintf(
		"%s/platform/v1/projects/%s/email/verifications",
		c.baseURL,
		url.PathEscape(c.projectID),
	)
}

func (c *Client) emailVerificationConfirmationURL(verificationID string) string {
	return fmt.Sprintf(
		"%s/platform/v1/projects/%s/email/verifications/%s/confirmations",
		c.baseURL,
		url.PathEscape(c.projectID),
		url.PathEscape(verificationID),
	)
}

func parseDataEnvelope(httpStatus int, requestID string, responseBody []byte, output any) error {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		return newMalformedResponseError(httpStatus, requestID, err)
	}

	data := bytes.TrimSpace(envelope.Data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return newMalformedResponseError(httpStatus, requestID, nil)
	}

	if output == nil {
		return nil
	}

	if err := json.Unmarshal(data, output); err != nil {
		return newMalformedResponseError(httpStatus, requestID, err)
	}

	return nil
}

func parseAPIError(resp *http.Response, responseBody []byte) error {
	apiError := &Error{
		Code:       fmt.Sprintf("HTTP_%d", resp.StatusCode),
		Message:    http.StatusText(resp.StatusCode),
		HTTPStatus: resp.StatusCode,
		RequestID:  resp.Header.Get("X-Request-ID"),
		RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
	}

	var envelope struct {
		Error struct {
			Code      string `json:"code"`
			Message   string `json:"message"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	if err := json.Unmarshal(responseBody, &envelope); err == nil {
		if envelope.Error.Code != "" {
			apiError.Code = envelope.Error.Code
		}
		if envelope.Error.Message != "" {
			apiError.Message = envelope.Error.Message
		}
		if envelope.Error.RequestID != "" {
			apiError.RequestID = envelope.Error.RequestID
		}
	}

	return apiError
}

func parseRetryAfter(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}

	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}

	if retryAt, err := http.ParseTime(value); err == nil {
		duration := time.Until(retryAt)
		if duration <= 0 {
			return 0
		}
		return duration
	}

	return 0
}
