# 2026-08-14 — Basic Email Go SDK surface

## 작업 목적과 이해한 내용

- Email Go SDK 기본 사용법이 `Delivery.EmailVerifications.Create`와 template/base URL 설명 중심이라 서버 내부 구조와 고급 설정이 먼저 보였다.
- 기본 인터페이스는 단건 Email 발송과 Email 인증 코드처럼 소비자가 하려는 일 중심이어야 한다.
- Delivery 기본 endpoint는 SDK가 사용하고, local/test override는 advanced configuration으로만 남긴다.
- Python/Django server SDK는 현재 없으므로 임의 생성하지 않고 후속 필요 작업으로 기록한다.

## 문제 진단 또는 기존 동작

- `client.EmailVerifications` alias는 있었지만 README 기본 예시는 `client.Delivery.EmailVerifications.Create`를 사용했다.
- 일반 단건 Email 발송 endpoint는 Delivery에 구현되어 있었지만 Go SDK public surface에는 없었다.
- README의 클라이언트 생성 직후에 `BaseURL` override와 template mode가 노출되어 기본 사용자가 local Delivery URL이나 template 선행 생성을 알아야 하는 흐름처럼 보였다.

## 적용한 내용과 주요 변경 파일

- `client.go`
  - `client.Email` facade를 추가했다.
  - 기본 user agent를 `spectra-notification-go/0.1.2`로 갱신했다.
- `email.go`
  - `client.Email.Send(ctx, SendEmailInput{To, Subject, Text, HTML, Metadata})`를 추가했다.
  - `POST /platform/v1/projects/{project_id}/email/delivery-requests` 요청과 `request_id/status/created_at` 응답 parsing을 구현했다.
  - `To`, `Subject`, `Text` 기본 validation과 자동 idempotency key를 적용했다.
- `email_verifications.go`
  - `client.Email.Verifications.SendCode/SendLink/ConfirmCode/ConfirmLink` wrapper를 추가했다.
  - 기존 `Create/Confirm`, `client.EmailVerifications`, `client.Delivery.EmailVerifications`는 호환 경로로 유지했다.
- `client_test.go`
  - 단건 Email send request/response와 validation 테스트를 추가했다.
  - 기본 Email verification direct body 테스트를 `client.Email.Verifications.SendCode` 경로로 갱신했다.
- `README.md`, `HANDOFF.md`, `WORKLOG.md`
  - 기본 README/example에서 base URL override와 template mode를 빼고 advanced/legacy 섹션으로 이동했다.
  - 보내는 사람은 per-request SDK 필드가 아니라 Delivery Project sender 설정으로 결정된다는 경계를 기록했다.
  - Python/Django server SDK 필요성을 남겼다.

## 실행한 검증과 결과

- `gofmt -w client.go email.go email_verifications.go client_test.go` — 통과
- `go test ./...` — 통과

## 남은 작업, 미검증 항목 또는 주의사항

- 실제 `delivery.spectra.kr` public E2E와 SMTP mailbox rendering은 수행하지 않았다.
- 모두의캠프 Go backend 적용은 별도 소비자 작업이다.
- Python/Django server SDK는 아직 없다. Go/Python/Django/JS server SDK의 Email interface 명명과 입력 개념을 후속으로 맞춰야 한다.
- per-request `From`은 현재 Delivery 단건 Email request schema에 없다. 발신자는 Delivery Project sender 설정/API 계약으로 다뤄야 한다.
