package notification

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewClientValidatesServerAPIKeyAliases(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		wantKey     string
		wantMessage string
	}{
		{
			name: "server api key",
			config: Config{
				ProjectID:    "project_123",
				ServerAPIKey: "key_123",
			},
			wantKey: "key_123",
		},
		{
			name: "legacy project api token",
			config: Config{
				ProjectID:       "project_123",
				ProjectAPIToken: "legacy_key_123",
			},
			wantKey: "legacy_key_123",
		},
		{
			name: "matching aliases",
			config: Config{
				ProjectID:       "project_123",
				ServerAPIKey:    " key_123 ",
				ProjectAPIToken: "key_123",
			},
			wantKey: "key_123",
		},
		{
			name: "missing server api key",
			config: Config{
				ProjectID: "project_123",
			},
			wantMessage: "ServerAPIKey is required",
		},
		{
			name: "conflicting aliases",
			config: Config{
				ProjectID:       "project_123",
				ServerAPIKey:    "key_123",
				ProjectAPIToken: "other_key_123",
			},
			wantMessage: "ServerAPIKey and ProjectAPIToken must match when both are set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if tt.wantMessage == "" {
				if err != nil {
					t.Fatalf("NewClient returned error: %v", err)
				}
				if client.serverAPIKey != tt.wantKey {
					t.Fatalf("client.serverAPIKey = %q, want %q", client.serverAPIKey, tt.wantKey)
				}
				return
			}

			if err == nil {
				t.Fatal("NewClient returned nil error")
			}
			var sdkError *Error
			if !errors.As(err, &sdkError) {
				t.Fatalf("error type = %T, want *Error", err)
			}
			if sdkError.Code != CodeValidationError {
				t.Fatalf("sdkError.Code = %q, want %s", sdkError.Code, CodeValidationError)
			}
			if sdkError.Message != tt.wantMessage {
				t.Fatalf("sdkError.Message = %q, want %q", sdkError.Message, tt.wantMessage)
			}
		})
	}
}

func TestCreateEmailVerificationSendsRequestAndParsesAcceptedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/platform/v1/projects/project_123/email/verifications" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token_123" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Idempotency-Key") != "idem_12345678901" {
			t.Fatalf("Idempotency-Key = %q", r.Header.Get("Idempotency-Key"))
		}
		if r.Header.Get("X-Spectra-Project-Id") != "project_123" {
			t.Fatalf("X-Spectra-Project-Id = %q", r.Header.Get("X-Spectra-Project-Id"))
		}
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
			t.Fatalf("Content-Type = %q", got)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Fatalf("Accept = %q", r.Header.Get("Accept"))
		}
		if r.Header.Get("User-Agent") != "test-agent" {
			t.Fatalf("User-Agent = %q", r.Header.Get("User-Agent"))
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		assertEqual(t, body["recipient_email"], "user@example.com")
		assertEqual(t, body["method"], "link")
		assertEqual(t, body["template_id"], "email-login")
		assertEqual(t, body["template_version"], float64(1))
		assertEqual(t, body["redirect_uri"], "https://app.example.com/verify")
		assertEqual(t, body["expires_in_seconds"], float64(600))
		metadata, ok := body["metadata"].(map[string]any)
		if !ok {
			t.Fatalf("metadata type = %T", body["metadata"])
		}
		assertEqual(t, metadata["tenant"], "spectra")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"data":{"verification_id":"ver_123","method":"link","status":"pending","expires_at":"2026-08-13T12:00:00Z"}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	client.userAgent = "test-agent"

	challenge, err := client.Delivery.EmailVerifications.Create(context.Background(), CreateEmailVerificationInput{
		RecipientEmail:   "user@example.com",
		Method:           EmailVerificationMethodLink,
		TemplateID:       "email-login",
		TemplateVersion:  1,
		ExpiresInSeconds: 600,
		Metadata:         map[string]string{"tenant": "spectra"},
		RedirectURI:      "https://app.example.com/verify",
		IdempotencyKey:   "idem_12345678901",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if challenge.VerificationID != "ver_123" {
		t.Fatalf("challenge.VerificationID = %q", challenge.VerificationID)
	}
	if challenge.Status != "pending" {
		t.Fatalf("challenge.Status = %q", challenge.Status)
	}
	if challenge.Method != EmailVerificationMethodLink {
		t.Fatalf("challenge.Method = %q", challenge.Method)
	}
	if challenge.ExpiresAt.IsZero() {
		t.Fatal("challenge.ExpiresAt is zero")
	}
}

func TestEmailSendSendsDirectBodyAndParsesAcceptedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/platform/v1/projects/project_123/email/delivery-requests" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token_123" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Idempotency-Key") != "idem_12345678901" {
			t.Fatalf("Idempotency-Key = %q", r.Header.Get("Idempotency-Key"))
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		assertEqual(t, body["recipient_email"], "user@example.com")
		assertEqual(t, body["subject"], "Welcome")
		assertEqual(t, body["text"], "Welcome to Spectra.")
		assertEqual(t, body["html"], "<p>Welcome to <strong>Spectra</strong>.</p>")
		metadata, ok := body["metadata"].(map[string]any)
		if !ok {
			t.Fatalf("metadata type = %T", body["metadata"])
		}
		assertEqual(t, metadata["campaign"], "welcome")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"data":{"request_id":"req_123","status":"accepted","created_at":"2026-08-14T09:00:00Z"}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	request, err := client.Email.Send(context.Background(), SendEmailInput{
		To:             "user@example.com",
		Subject:        "Welcome",
		Text:           "Welcome to Spectra.",
		HTML:           "<p>Welcome to <strong>Spectra</strong>.</p>",
		Metadata:       map[string]string{"campaign": "welcome"},
		IdempotencyKey: "idem_12345678901",
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if request.RequestID != "req_123" {
		t.Fatalf("request.RequestID = %q", request.RequestID)
	}
	if request.Status != "accepted" {
		t.Fatalf("request.Status = %q", request.Status)
	}
	if request.CreatedAt.IsZero() {
		t.Fatal("request.CreatedAt is zero")
	}
}

func TestEmailSendValidatesInputWithoutHTTPRequest(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	tests := []struct {
		name  string
		input SendEmailInput
	}{
		{name: "missing to", input: SendEmailInput{Subject: "Hello", Text: "Body"}},
		{name: "missing subject", input: SendEmailInput{To: "user@example.com", Text: "Body"}},
		{name: "subject header injection", input: SendEmailInput{To: "user@example.com", Subject: "Hello\r\nBcc: test@example.com", Text: "Body"}},
		{name: "missing text", input: SendEmailInput{To: "user@example.com", Subject: "Hello"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.Email.Send(context.Background(), tt.input)
			if err == nil {
				t.Fatal("Send returned nil error")
			}
			if !IsCode(err, CodeValidationError) {
				t.Fatalf("error code = %v, want %s", err, CodeValidationError)
			}
		})
	}
	if got := atomic.LoadInt32(&requestCount); got != 0 {
		t.Fatalf("requestCount = %d, want 0", got)
	}
}

func TestEmailVerificationSendCodeSendsDirectBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		assertEqual(t, body["recipient_email"], "admin@example.com")
		assertEqual(t, body["method"], "code")
		assertEqual(t, body["subject"], "모두의캠프 관리자 로그인 인증 코드")
		assertEqual(t, body["text"], "인증 코드: {{verification_code}}\n만료: {{expires_at}}")
		assertEqual(t, body["html"], "<p>인증 코드: <strong>{{verification_code}}</strong></p>")
		assertEqual(t, body["expires_in_seconds"], float64(600))
		metadata, ok := body["metadata"].(map[string]any)
		if !ok {
			t.Fatalf("metadata type = %T", body["metadata"])
		}
		assertEqual(t, metadata["consumer"], "modo-camp")
		assertEqual(t, metadata["purpose"], "admin_login")
		for _, field := range []string{"template_id", "template_version", "redirect_uri"} {
			if _, ok := body[field]; ok {
				t.Fatalf("%s should be omitted for direct mode", field)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"data":{"verification_id":"ver_direct","method":"code","status":"pending","expires_at":"2026-08-13T12:00:00Z"}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	challenge, err := client.Email.Verifications.SendCode(context.Background(), SendEmailCodeInput{
		To:               "admin@example.com",
		Subject:          "모두의캠프 관리자 로그인 인증 코드",
		Text:             "인증 코드: {{verification_code}}\n만료: {{expires_at}}",
		HTML:             "<p>인증 코드: <strong>{{verification_code}}</strong></p>",
		ExpiresInSeconds: 600,
		Metadata: map[string]string{
			"consumer": "modo-camp",
			"purpose":  "admin_login",
		},
		IdempotencyKey: "idem_12345678901",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if challenge.VerificationID != "ver_direct" {
		t.Fatalf("challenge.VerificationID = %q", challenge.VerificationID)
	}
}

func TestCreateEmailVerificationValidatesSourceModeWithoutHTTPRequest(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	tests := []struct {
		name  string
		input CreateEmailVerificationInput
	}{
		{
			name: "mixed template and direct body",
			input: CreateEmailVerificationInput{
				RecipientEmail: "admin@example.com",
				TemplateID:     "admin-login-code",
				Subject:        "Login code",
			},
		},
		{
			name: "template version without template id",
			input: CreateEmailVerificationInput{
				RecipientEmail:  "admin@example.com",
				TemplateVersion: 1,
			},
		},
		{
			name: "direct body without subject",
			input: CreateEmailVerificationInput{
				RecipientEmail: "admin@example.com",
				Text:           "인증 코드: {{verification_code}}",
			},
		},
		{
			name: "direct body without text",
			input: CreateEmailVerificationInput{
				RecipientEmail: "admin@example.com",
				Subject:        "로그인 코드",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.Delivery.EmailVerifications.Create(context.Background(), tt.input)
			if err == nil {
				t.Fatal("Create returned nil error")
			}
			if !IsCode(err, CodeValidationError) {
				t.Fatalf("error code = %v, want %s", err, CodeValidationError)
			}
		})
	}
	if got := atomic.LoadInt32(&requestCount); got != 0 {
		t.Fatalf("requestCount = %d, want 0", got)
	}
}

func TestCreateEmailVerificationAutoGeneratesIdempotencyKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Idempotency-Key")
		if key == "" {
			t.Fatal("Idempotency-Key is empty")
		}
		if len(key) != 36 {
			t.Fatalf("Idempotency-Key length = %d, want 36", len(key))
		}
		if !strings.HasPrefix(key, "sdk:") {
			t.Fatalf("Idempotency-Key = %q, want sdk prefix", key)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		assertEqual(t, body["method"], "code")
		assertEqual(t, body["expires_in_seconds"], float64(600))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"data":{"verification_id":"ver_auto","method":"code","status":"pending","expires_at":"2026-08-13T12:00:00Z"}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	if _, err := client.EmailVerifications.Create(context.Background(), CreateEmailVerificationInput{
		RecipientEmail: "user@example.com",
		TemplateID:     "email-login",
	}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
}

func TestConfirmEmailVerificationSendsRequestAndParsesOKResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/platform/v1/projects/project_123/email/verifications/ver_123/confirmations" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token_123" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		assertEqual(t, body["code"], "123456")
		if _, ok := body["link_token"]; ok {
			t.Fatalf("link_token should be omitted when Code is used")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"verification_id":"ver_123","status":"verified","verified_at":"2026-08-13T12:01:00Z"}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	proof, err := client.Email.Verifications.ConfirmCode(context.Background(), ConfirmEmailCodeInput{
		VerificationID: "ver_123",
		Code:           "123456",
	})
	if err != nil {
		t.Fatalf("Confirm returned error: %v", err)
	}
	if proof.VerificationID != "ver_123" {
		t.Fatalf("proof.VerificationID = %q", proof.VerificationID)
	}
	if proof.VerifiedAt == nil || proof.VerifiedAt.IsZero() {
		t.Fatalf("proof.VerifiedAt = %v", proof.VerifiedAt)
	}
}

func TestConfirmEmailVerificationValidatesExactlyOneSecretWithoutHTTPRequest(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	_, err := client.EmailVerifications.Confirm(context.Background(), ConfirmEmailVerificationInput{
		VerificationID: "ver_123",
		Code:           "123456",
		LinkToken:      "link_token_123",
	})
	if err == nil {
		t.Fatal("Confirm returned nil error")
	}

	var sdkError *Error
	if !errors.As(err, &sdkError) {
		t.Fatalf("error type = %T, want *Error", err)
	}
	if sdkError.Code != CodeValidationError {
		t.Fatalf("sdkError.Code = %q", sdkError.Code)
	}
	if got := atomic.LoadInt32(&requestCount); got != 0 {
		t.Fatalf("requestCount = %d, want 0", got)
	}
}

func TestRateLimitErrorParsesRetryAfterAndRequestID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":"RATE_LIMITED","message":"too many requests","request_id":"req_123"}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	_, err := client.EmailVerifications.Create(context.Background(), CreateEmailVerificationInput{
		RecipientEmail:   "user@example.com",
		Method:           EmailVerificationMethodCode,
		TemplateID:       "email-login",
		ExpiresInSeconds: 600,
		IdempotencyKey:   "idem_12345678901",
	})
	if err == nil {
		t.Fatal("Create returned nil error")
	}

	var sdkError *Error
	if !errors.As(err, &sdkError) {
		t.Fatalf("error type = %T, want *Error", err)
	}
	if sdkError.Code != "RATE_LIMITED" {
		t.Fatalf("sdkError.Code = %q", sdkError.Code)
	}
	if sdkError.Message != "too many requests" {
		t.Fatalf("sdkError.Message = %q", sdkError.Message)
	}
	if sdkError.HTTPStatus != http.StatusTooManyRequests {
		t.Fatalf("sdkError.HTTPStatus = %d", sdkError.HTTPStatus)
	}
	if sdkError.RequestID != "req_123" {
		t.Fatalf("sdkError.RequestID = %q", sdkError.RequestID)
	}
	if sdkError.RetryAfter != 7*time.Second {
		t.Fatalf("sdkError.RetryAfter = %s", sdkError.RetryAfter)
	}
}

func TestMalformedResponseReturnsMalformedResponseError(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid json", body: `{"data":`},
		{name: "missing data", body: `{}`},
		{name: "null data", body: `{"data":null}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := newTestClient(t, server.URL)
			_, err := client.EmailVerifications.Create(context.Background(), CreateEmailVerificationInput{
				RecipientEmail:   "user@example.com",
				Method:           EmailVerificationMethodCode,
				TemplateID:       "email-login",
				ExpiresInSeconds: 600,
				IdempotencyKey:   "idem_12345678901",
			})
			if err == nil {
				t.Fatal("Create returned nil error")
			}

			var sdkError *Error
			if !errors.As(err, &sdkError) {
				t.Fatalf("error type = %T, want *Error", err)
			}
			if sdkError.Code != CodeMalformedResponse {
				t.Fatalf("sdkError.Code = %q", sdkError.Code)
			}
			if sdkError.HTTPStatus != http.StatusAccepted {
				t.Fatalf("sdkError.HTTPStatus = %d", sdkError.HTTPStatus)
			}
		})
	}
}

func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()

	client, err := NewClient(Config{
		ProjectID:    "project_123",
		ServerAPIKey: "token_123",
		Environment:  EnvironmentTest,
		BaseURL:      baseURL,
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	return client
}

func assertEqual(t *testing.T, got any, want any) {
	t.Helper()

	if got != want {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
