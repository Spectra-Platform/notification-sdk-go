# notification-sdk-go WORKLOG

## 2026-08-14 — Basic Email Go SDK surface

- 목적: Go server SDK 기본 UX를 서버 내부 Delivery 구조와 template 선행 생성 중심에서, 보내기/인증 코드 발송처럼 소비자가 하려는 일 중심으로 단순화.
- 결과: `client.Email.Send`와 `client.Email.Verifications.SendCode/SendLink/ConfirmCode/ConfirmLink`를 추가하고, README 기본 플로우에서 base URL override와 template mode를 advanced/legacy로 이동했다.
- 주요 변경:
  - `client.go`
  - `email.go`
  - `email_verifications.go`
  - `client_test.go`
  - `README.md`
  - `HANDOFF.md`
  - `docs/work-logs/2026-08-14-01-basic-email-go-sdk-surface.md`
- 검증:
  - `gofmt -w client.go email.go email_verifications.go client_test.go`, `go test ./...` 통과
- 미검증:
  - 실제 `delivery.spectra.kr` public E2E, SMTP mailbox render, 모두의캠프 backend 적용, Python/Django server SDK

## 2026-08-13 — Direct Email verification body SDK support

- 목적: 모두의캠프 Go backend가 template을 먼저 생성하지 않고 관리자 로그인 인증 이메일을 발송할 수 있도록 Delivery의 direct verification body 계약을 Go server SDK에 반영.
- 결과: `CreateEmailVerificationInput`에 `Subject`, `Text`, `HTML`을 추가하고 direct body와 template mode를 상호 배타적으로 직렬화·검증. direct body 예시를 README 기본 경로로 바꾸고 template은 고급 경로로 유지.
- 주요 변경:
  - `client.go`
  - `email_verifications.go`
  - `client_test.go`
  - `README.md`
  - `HANDOFF.md`
  - `docs/work-logs/2026-08-13-02-direct-email-verification-body.md`
- 검증:
  - `gofmt -w email_verifications.go client.go client_test.go`, `go test ./...`, `go vet ./...`, `git diff --check` 통과
- 미검증:
  - 실제 `delivery.spectra.kr` public E2E, SMTP mailbox render, 모두의캠프 backend 적용

## 2026-08-13 — Go server email verification SDK v0.1 scope

- 목적: 모두캠프 Go backend가 pre-auth 관리자 이메일 로그인에서 Project API Token 기반으로 Spectra Email Verification create/confirm을 호출할 수 있게 Go server SDK 첫 범위를 구현.
- 결과: `notification.NewClient`, `client.Delivery.EmailVerifications.Create/Confirm`, typed error, idempotency key 자동 생성, base URL/http client/timeout override를 구현.
- 주요 변경:
  - `go.mod`
  - `client.go`
  - `email_verifications.go`
  - `errors.go`
  - `client_test.go`
  - `README.md`
  - `HANDOFF.md`
  - `docs/work-logs/2026-08-13-01-go-email-verification-sdk.md`
- 검증:
  - `go test ./...` 통과
- 미검증:
  - 실제 `delivery.spectra.kr` public E2E
  - 모두캠프 Go backend 적용
  - release/tag/package distribution
