# Spectra Notification SDK for Go

Go backend에서 Spectra Notification/Delivery 기능을 server-to-server로 호출하기 위한 SDK입니다.

첫 범위는 이메일 인증 코드 발송과 확인입니다. 모두캠프 관리자 이메일 로그인처럼 로그인 전(pre-auth) 상태에서 브라우저 SDK를 사용할 수 없고, Project API Token을 서버에만 보관해야 하는 경우를 대상으로 합니다.

## 설치

```bash
go get github.com/Spectra-Platform/notification-sdk-go
```

```go
import notification "github.com/Spectra-Platform/notification-sdk-go"
```

## 보안 경계

- 이 SDK는 Go 서버 전용입니다.
- `ProjectAPIToken`은 서버 환경 변수나 secret manager에만 저장합니다.
- 브라우저, 모바일 앱, 프론트엔드 번들에는 Project API Token을 넣지 않습니다.
- 사용자 앱의 자체 로그인 세션은 앱 백엔드가 별도로 발급해도 됩니다. 이 SDK는 Spectra Notification 호출만 담당합니다.

## 클라이언트 생성

```go
client, err := notification.NewClient(notification.Config{
    ProjectID:       projectID,
    ProjectAPIToken: projectAPIToken,
    Environment:     notification.EnvironmentTest,
})
if err != nil {
    // config validation error
}
```

기본 `BaseURL`은 `https://delivery.spectra.kr`입니다. 로컬 Docker network나 내부 테스트에서는 override할 수 있습니다.

```go
client, err := notification.NewClient(notification.Config{
    ProjectID:       projectID,
    ProjectAPIToken: projectAPIToken,
    Environment:     notification.EnvironmentTest,
    BaseURL:         "http://delivery-api:8080",
    Timeout:         5 * time.Second,
})
```

기존 transport를 쓰고 싶으면 `HTTPClient`를 넘깁니다.

```go
client, err := notification.NewClient(notification.Config{
    ProjectID:       projectID,
    ProjectAPIToken: projectAPIToken,
    HTTPClient:      myHTTPClient,
})
```

## 이메일 인증 코드 보내기

```go
challenge, err := client.Delivery.EmailVerifications.Create(ctx, notification.CreateEmailVerificationInput{
    RecipientEmail: "admin@example.com",
    TemplateID:     templateID,
    Metadata: map[string]string{
        "consumer": "modo-camp",
        "purpose":  "admin_login",
    },
})
if err != nil {
    // handle typed SDK/API error
}

fmt.Println(challenge.VerificationID)
```

기본값:

- `Method`: `notification.EmailVerificationMethodCode`
- `ExpiresInSeconds`: `600`
- `IdempotencyKey`: SDK가 자동 생성

명시적으로 지정할 수도 있습니다.

```go
challenge, err := client.Delivery.EmailVerifications.Create(ctx, notification.CreateEmailVerificationInput{
    RecipientEmail:   "admin@example.com",
    Method:           notification.EmailVerificationMethodCode,
    TemplateID:       templateID,
    TemplateVersion:  1,
    ExpiresInSeconds: 600,
    IdempotencyKey:   requestID,
})
```

링크 방식이 필요하면 `RedirectURI`를 함께 넘깁니다.

```go
challenge, err := client.Delivery.EmailVerifications.Create(ctx, notification.CreateEmailVerificationInput{
    RecipientEmail:   "admin@example.com",
    Method:           notification.EmailVerificationMethodLink,
    TemplateID:       templateID,
    ExpiresInSeconds: 900,
    RedirectURI:      "https://admin.example.com/email/callback",
})
```

## 이메일 인증 코드 확인

```go
proof, err := client.Delivery.EmailVerifications.Confirm(ctx, notification.ConfirmEmailVerificationInput{
    VerificationID: challenge.VerificationID,
    Code:           "123456",
})
if err != nil {
    // invalid, expired, rate limited, unauthorized 등 확인
}

fmt.Println(proof.Status)
```

링크 토큰 확인이 필요한 경우 `Code` 대신 `LinkToken`만 넘깁니다.

## 오류 처리

모든 SDK/Platform 오류는 `*notification.Error`로 확인할 수 있습니다.

```go
challenge, err := client.Delivery.EmailVerifications.Create(ctx, input)
if err != nil {
    var sdkErr *notification.Error
    if errors.As(err, &sdkErr) {
        log.Printf("spectra notification error code=%s http=%d request_id=%s retry_after=%s",
            sdkErr.Code,
            sdkErr.HTTPStatus,
            sdkErr.RequestID,
            sdkErr.RetryAfter,
        )
    }
}
```

주요 코드:

| Code | 의미 |
| --- | --- |
| `VALIDATION_ERROR` | SDK 입력값이 잘못되어 HTTP 요청을 보내지 않음 |
| `VALIDATION_FAILED` | Delivery API가 요청 body/path/header 검증에 실패함 |
| `NETWORK_UNAVAILABLE` | Spectra Delivery endpoint에 도달하지 못함 |
| `MALFORMED_RESPONSE` | Delivery 응답이 SDK 계약과 맞지 않음 |
| `EMAIL_VERIFICATION_INVALID` | 잘못된 인증 코드 또는 링크 토큰 |
| `EMAIL_VERIFICATION_EXPIRED` | 인증 만료 |
| `EMAIL_VERIFICATION_CONSUMED` | 이미 사용된 인증 |
| `EMAIL_VERIFICATION_LOCKED` | 시도 횟수 초과 등으로 잠김 |
| `DELIVERY_RATE_LIMITED` | 발송 또는 확인 rate limit |
| `PROJECT_TOKEN_UNAUTHORIZED` | Project API Token 누락, 만료, 폐기, audience/scope/project mismatch |
| `EMAIL_RESOURCE_NOT_FOUND` | 템플릿 또는 이메일 인증 리소스를 찾을 수 없음 |
| `EMAIL_TEMPLATE_PUBLISH_CONFLICT` | 템플릿 발행 충돌. 일반 발송/확인 호출보다 운영 템플릿 API에서 주로 발생 |
| `AUTH_INTROSPECTION_UNAVAILABLE` | Auth introspection 일시 불가 |
| `IDEMPOTENCY_CONFLICT` | 같은 Idempotency-Key로 다른 create 요청을 보냄 |

서버가 `Retry-After`를 내려주면 `notification.Error.RetryAfter`에 반영됩니다.

## Idempotency

`Create`는 `Idempotency-Key`를 사용합니다. 직접 넘기지 않으면 SDK가 `sdk:<random>` 형식으로 자동 생성합니다.

동일한 키와 동일한 요청은 같은 인증 challenge를 재사용할 수 있습니다. 동일한 키로 다른 요청을 보내면 Platform이 `IDEMPOTENCY_CONFLICT`를 반환합니다.

## 현재 범위

구현됨:

- Go server SDK client
- Project API Token 기반 Authorization
- Email verification create/confirm
- 자동 idempotency key
- typed error, request id, retry-after parsing
- `http.Client`, timeout, base URL override
- httptest 기반 단위 테스트

아직 별도 검증 필요:

- 실제 `delivery.spectra.kr` public E2E
- 모두캠프 Go backend 적용
- push 발송, device registration 등 Notification 전체 영역
