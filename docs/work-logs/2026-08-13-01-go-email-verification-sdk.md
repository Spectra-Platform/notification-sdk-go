# 2026-08-13 — Go server email verification SDK

## 작업 목적

모두캠프 Go backend가 로그인 전(pre-auth) 관리자 이메일 인증 코드 발송/검증을 Spectra Platform의 공식 Go server SDK로 처리할 수 있도록 `notification-sdk-go` 첫 구현 범위를 만들었습니다.

브라우저 Notification SDK는 Project API Token을 보관할 수 없기 때문에 이 작업은 server-to-server SDK로 분리했습니다.

## 기존 상태

- 저장소는 public repo로 생성되어 있었고 `README.md`만 있었습니다.
- Go module, client, error model, email verification API surface, 테스트와 handoff/worklog 문서가 없었습니다.
- 모두캠프는 임시 REST adapter로 Delivery Email Verification을 직접 호출해야 하는 상태였습니다.

## 적용한 내용

- Go module 추가:
  - `github.com/Spectra-Platform/notification-sdk-go`
- Client config:
  - `ProjectID`
  - `ProjectAPIToken`
  - `Environment`
  - `BaseURL`
  - `HTTPClient`
  - `Timeout`
  - `UserAgent`
- Public API:
  - `client.Delivery.EmailVerifications.Create(ctx, input)`
  - `client.Delivery.EmailVerifications.Confirm(ctx, input)`
  - `client.EmailVerifications` alias
- Email verification create:
  - 기본 method `code`
  - 기본 만료 `600`초
  - optional `TemplateVersion`
  - optional `Metadata`
  - link mode용 `RedirectURI`
  - 자동 `Idempotency-Key`
- Email verification confirm:
  - `Code` 또는 `LinkToken` 중 정확히 하나만 허용
- Error model:
  - `*notification.Error`
  - `Code`
  - `HTTPStatus`
  - `RequestID`
  - `RetryAfter`
  - `Unwrap`
  - `IsCode`
- Transport:
  - Project API Token bearer Authorization
  - `X-Spectra-Project-Id`
  - JSON data/error envelope parsing
  - network/cancel/malformed response typed error
- 문서:
  - README 사용 예시와 보안 경계
  - HANDOFF 저장소 기준
  - WORKLOG 인덱스

## 주요 변경 파일

- `go.mod`
- `client.go`
- `email_verifications.go`
- `errors.go`
- `client_test.go`
- `README.md`
- `HANDOFF.md`
- `WORKLOG.md`

## 검증

```bash
go test ./...
```

결과:

```text
ok  	github.com/Spectra-Platform/notification-sdk-go
```

## 미검증 항목

- 실제 `delivery.spectra.kr` public E2E
- 실제 Project API Token을 사용한 create/confirm
- 모두캠프 Go backend 적용
- GitHub release/tag/package distribution

## 주의사항

- Project API Token은 Go 서버 환경에만 보관해야 합니다.
- browser/mobile SDK로 이 SDK를 대체하면 안 됩니다.
- Delivery OpenAPI의 email verification 구현 상태 표기와 실제 runtime 상태가 일부 다를 수 있어, public E2E 전에는 완료로 포장하지 않습니다.
