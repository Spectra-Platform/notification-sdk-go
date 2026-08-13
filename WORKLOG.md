# notification-sdk-go WORKLOG

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
