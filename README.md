# Spectra Notification SDK for Go

Go backend에서 Spectra Email/Notification 기능을 server-to-server로 호출하기 위한 SDK입니다.

기본 사용은 단순한 Email 작업부터 시작합니다. Console/Auth에서 Email 기능을 포함해 발급한 Server API Key는 서버에만 보관하고, SDK는 기본 Delivery endpoint를 자동으로 사용합니다.

## 설치

```bash
go get github.com/Spectra-Platform/notification-sdk-go
```

```go
import notification "github.com/Spectra-Platform/notification-sdk-go"
```

## 보안 경계

- 이 SDK는 Go 서버 전용입니다.
- `ServerAPIKey`는 서버 환경 변수나 secret manager에만 저장합니다.
- 브라우저, 모바일 앱, 프론트엔드 번들에는 Server API Key나 legacy Project API token을 넣지 않습니다.
- 사용자 앱의 자체 로그인 세션은 앱 백엔드가 별도로 발급해도 됩니다. 이 SDK는 Spectra Email/Notification 호출만 담당합니다.

## 클라이언트 생성

```go
client, err := notification.NewClient(notification.Config{
    ProjectID:    projectID,
    ServerAPIKey: serverAPIKey,
    Environment:  notification.EnvironmentTest,
})
if err != nil {
    // config validation error
}
```

## 이메일 보내기

```go
accepted, err := client.Email.Send(ctx, notification.SendEmailInput{
    To:      "user@example.com",
    Subject: "모두의캠프 가입 안내",
    Text:    "모두의캠프에 오신 것을 환영합니다.",
    HTML:    `<p>모두의캠프에 오신 것을 환영합니다.</p>`,
})
if err != nil {
    // handle typed SDK/API error
}

fmt.Println(accepted.RequestID)
```

보내는 사람은 Delivery의 Project sender 설정과 environment 정책을 따릅니다. test environment에서는 Platform sandbox sender를 사용할 수 있고, live 발송은 검증된 Project sender가 필요합니다.

기본 Email 입력은 받는 사람(`To`), 제목(`Subject`), plain text 본문(`Text`) 중심입니다. `HTML`과 `Metadata`는 선택 사항입니다. `IdempotencyKey`를 직접 넘기지 않으면 SDK가 자동 생성합니다.

## 이메일 인증 코드 보내기

```go
challenge, err := client.Email.Verifications.SendCode(ctx, notification.SendEmailCodeInput{
    To:      "admin@example.com",
    Subject: "모두의캠프 관리자 로그인 인증 코드",
    Text:    "인증 코드: {{verification_code}}\n만료: {{expires_at}}",
    HTML:    `<p>인증 코드: <strong>{{verification_code}}</strong></p>`,
})
if err != nil {
    // handle typed SDK/API error
}

fmt.Println(challenge.VerificationID)
```

Delivery가 `{{verification_code}}`, `{{verification_link}}`, `{{expires_at}}`, metadata placeholder를 치환합니다. 코드 방식에는 `{{verification_code}}`, 링크 방식에는 `{{verification_link}}`를 본문에 포함해야 합니다.

기본값:

- `Method`: `notification.EmailVerificationMethodCode`
- `ExpiresInSeconds`: `600`
- `IdempotencyKey`: SDK가 자동 생성

인증 코드를 확인할 때는 다음처럼 호출합니다.

```go
proof, err := client.Email.Verifications.ConfirmCode(ctx, notification.ConfirmEmailCodeInput{
    VerificationID: challenge.VerificationID,
    Code:           "123456",
})
if err != nil {
    // invalid, expired, rate limited, unauthorized 등 확인
}

fmt.Println(proof.Status)
```

링크 방식이 필요하면 `SendLink`와 `ConfirmLink`를 사용합니다.

```go
challenge, err := client.Email.Verifications.SendLink(ctx, notification.SendEmailLinkInput{
    To:          "admin@example.com",
    Subject:     "모두의캠프 관리자 로그인 링크",
    Text:        "로그인 링크: {{verification_link}}\n만료: {{expires_at}}",
    HTML:        `<p>로그인 링크: <a href="{{verification_link}}">열기</a></p>`,
    RedirectURI: "https://admin.example.com/email/callback",
})
```

## 오류 처리

모든 SDK/Platform 오류는 `*notification.Error`로 확인할 수 있습니다.

```go
accepted, err := client.Email.Send(ctx, input)
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
| `DELIVERY_QUEUE_UNAVAILABLE` | Delivery queue가 일시적으로 요청을 받을 수 없음 |
| `PROJECT_TOKEN_UNAUTHORIZED` | Server API Key 누락, 만료, 폐기, Email 기능 미포함, project mismatch |
| `EMAIL_RESOURCE_NOT_FOUND` | 템플릿 또는 이메일 인증 리소스를 찾을 수 없음 |
| `EMAIL_TEMPLATE_PUBLISH_CONFLICT` | 템플릿 발행 충돌. 일반 발송/확인 호출보다 운영 템플릿 API에서 주로 발생 |
| `AUTH_INTROSPECTION_UNAVAILABLE` | Auth introspection 일시 불가 |
| `IDEMPOTENCY_CONFLICT` | 같은 Idempotency-Key로 다른 create 요청을 보냄 |

서버가 `Retry-After`를 내려주면 `notification.Error.RetryAfter`에 반영됩니다.

## Idempotency

`Email.Send`와 verification create 계열 호출은 `Idempotency-Key`를 사용합니다. 직접 넘기지 않으면 SDK가 `sdk:<random>` 형식으로 자동 생성합니다.

동일한 키와 동일한 요청은 같은 작업을 재사용할 수 있습니다. 동일한 키로 다른 요청을 보내면 Platform이 `IDEMPOTENCY_CONFLICT`를 반환합니다.

## 고급 설정

기본 endpoint는 `https://delivery.spectra.kr`입니다. 일반 사용자는 별도 Delivery URL을 넘기지 않습니다. 로컬 Docker network나 내부 테스트에서는 `BaseURL`을 override할 수 있습니다.

```go
client, err := notification.NewClient(notification.Config{
    ProjectID:    projectID,
    ServerAPIKey: serverAPIKey,
    Environment:  notification.EnvironmentTest,
    BaseURL:      "http://delivery-api:8080",
    Timeout:      5 * time.Second,
})
```

기존 transport를 쓰고 싶으면 `HTTPClient`를 넘깁니다.

```go
client, err := notification.NewClient(notification.Config{
    ProjectID:    projectID,
    ServerAPIKey: serverAPIKey,
    HTTPClient:   myHTTPClient,
})
```

## 고급/legacy credential alias

새 코드에서는 `Config.ServerAPIKey`를 사용하세요. 이전 버전의 `Config.ProjectAPIToken`은 호환을 위한 legacy alias로 유지됩니다.

```go
client, err := notification.NewClient(notification.Config{
    ProjectID:       projectID,
    ProjectAPIToken: legacyProjectAPIToken,
})
```

`ServerAPIKey`와 `ProjectAPIToken`을 둘 다 넘기면 두 값은 같아야 합니다. 다르면 SDK가 HTTP 요청을 보내기 전에 `VALIDATION_ERROR`를 반환합니다.

## 고급/legacy template mode

저장·게시한 template을 재사용해야 하는 경우에만 legacy create API를 사용합니다. 기본 README/example에서는 template 선행 생성을 요구하지 않습니다.

```go
challenge, err := client.EmailVerifications.Create(ctx, notification.CreateEmailVerificationInput{
    RecipientEmail:   "admin@example.com",
    Method:           notification.EmailVerificationMethodCode,
    TemplateID:       templateID,
    TemplateVersion:  1,
    ExpiresInSeconds: 600,
    IdempotencyKey:   requestID,
})
```

direct body 필드(`Subject`, `Text`, `HTML`)와 `TemplateID`는 함께 지정할 수 없습니다. SDK는 이 경우 요청을 보내지 않고 `VALIDATION_ERROR`를 반환하며, Delivery API도 같은 mixed mode를 `422`로 거절합니다. `TemplateVersion`은 `TemplateID`를 지정한 고급 경로에서만 사용할 수 있습니다.

이전 예시의 `client.Delivery.EmailVerifications.Create/Confirm`과 `client.EmailVerifications.Create/Confirm`은 호환을 위해 유지됩니다. 새 코드에서는 `client.Email.Send`와 `client.Email.Verifications.SendCode/ConfirmCode`를 먼저 사용하세요.

## 현재 범위

구현됨:

- Go server SDK client
- Email 기능이 포함된 Server API Key 기반 Authorization
- 기본 public surface: `client.Email.Send`
- Email verification wrapper: `client.Email.Verifications.SendCode/SendLink/ConfirmCode/ConfirmLink`
- 기존 호환 surface: `client.EmailVerifications.Create/Confirm`, `client.Delivery.EmailVerifications.Create/Confirm`
- direct body (`Subject`/`Text`/optional `HTML`/optional `Metadata`)와 template mode
- 자동 idempotency key
- typed error, request id, retry-after parsing
- `http.Client`, timeout, base URL override
- httptest 기반 단위 테스트

아직 별도 검증 필요:

- 실제 `delivery.spectra.kr` public E2E
- 모두캠프 Go backend 적용
- Python/Django server SDK
- push 발송, device registration 등 Notification 전체 영역
