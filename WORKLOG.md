# notification-sdk-go WORKLOG

## 2026-08-14 — Secret Key credential surface

- 목적: 사용자-facing credential 명칭을 `Secret Key`로 확정한 방향에 맞춰 Go server SDK의 새 public config와 README 기본 예시를 정리.
- 결과: `Config.SecretKey`를 기본 credential field로 추가하고, 기존 `Config.ServerAPIKey`/`Config.ProjectAPIToken`은 deprecated alias로 유지했다. 여러 credential field가 동시에 주어지면 값이 모두 같아야 하며, 다르면 SDK가 HTTP 요청 전 `VALIDATION_ERROR`를 반환한다.
- 주요 변경:
  - `client.go`
  - `client_test.go`
  - `README.md`
  - `HANDOFF.md`
  - `docs/work-logs/2026-08-14-03-secret-key-credential-surface.md`
- 검증:
  - `gofmt -w client.go client_test.go`, `go test ./...`, `go vet ./...`, `go build ./...`, `git diff --check` 통과
  - README/HANDOFF/client/test 범위 legacy exact phrase grep에서 기본 사용자-facing 문구는 제거됐고 deprecated alias 문맥만 남음
- 미검증:
  - 실제 `delivery.spectra.kr` public E2E, 실제 Secret Key 발급/재발급 runtime, 모두의캠프 backend 적용

## 2026-08-14 — Server API Key alias for Email Go SDK

- 목적: Console/Auth가 기능별 token preset 대신 하나의 Server API Key에 기능을 포함하는 구조로 바뀌는 데 맞춰 Go server SDK의 기본 설정명과 문서 용어를 정리.
- 결과: `Config.ServerAPIKey`를 기본 credential field로 추가하고, 기존 `Config.ProjectAPIToken`은 legacy alias로 유지했다. 두 값이 모두 주어졌는데 다르면 SDK가 HTTP 요청 전에 `VALIDATION_ERROR`를 반환한다.
- 주요 변경:
  - `client.go`
  - `client_test.go`
  - `README.md`
  - `HANDOFF.md`
  - `docs/work-logs/2026-08-14-02-server-api-key-alias.md`
- 검증:
  - `gofmt -w client.go client_test.go`, `go test ./...`, `go vet ./...`, `go build ./...`, `git diff --check` 통과
- 미검증:
  - 실제 `delivery.spectra.kr` public E2E, 실제 Server API Key 발급/검증 runtime, 모두의캠프 backend 적용

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
