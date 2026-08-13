# 2026-08-13 — Direct Email verification body Go SDK

## 작업 목적과 이해한 내용

- Delivery API가 Email Verification create에서 direct body와 template mode를 모두 지원하도록 확장됐다.
- 모두의캠프 Go backend는 template을 선행 생성하지 않고 관리자 로그인 인증 이메일을 발송할 수 있어야 한다.
- Project API Token은 Go 서버에만 보관하며 browser/mobile client에 노출하지 않는다.

## 문제 진단 또는 기존 동작

- 기존 `CreateEmailVerificationInput`은 `TemplateID`를 필수로 강제해 direct body 요청을 만들 수 없었다.
- request encoder가 `template_id`를 항상 보냈으므로 새 Delivery OpenAPI 계약의 direct mode와 맞지 않았다.

## 적용한 내용과 주요 변경 파일

- `email_verifications.go`
  - `Subject`, `Text`, `HTML` public input을 추가했다.
  - direct body는 `subject`/`text`를 필수로 하고 optional `html`/`metadata`를 보낸다.
  - template mode는 `template_id`와 optional `template_version`을 계속 지원한다.
  - direct/template mixed mode, `TemplateID` 없는 `TemplateVersion`, 불완전한 direct body는 HTTP 전
    `VALIDATION_ERROR`로 막는다.
- `client_test.go`
  - direct body JSON encoding과 template field omission을 검증했다.
  - mixed/incomplete mode가 HTTP 요청을 보내지 않는지 검증했다.
- `README.md`, `HANDOFF.md`, `WORKLOG.md`
  - direct body를 기본 server SDK 사용 경로로 문서화하고 권한 경계와 template advanced path를 기록했다.
- `client.go`
  - patch release User-Agent를 `0.1.1`로 조정했다.

## 실행한 검증과 결과

- `gofmt -w email_verifications.go client.go client_test.go` — 통과
- `go test ./...` — 통과
- `go vet ./...` — 통과
- `git diff --check` — 통과

## 남은 작업, 미검증 항목 또는 주의사항

- SDK 테스트는 local `httptest` contract 검증이다. public Delivery endpoint, verified sender,
  SMTP mailbox rendering과 metadata/placeholder 실제 치환은 별도 runtime E2E가 필요하다.
- direct body를 app-user token으로 보내면 Delivery API가 거절한다. 이 SDK의 Project API Token은
  server process의 환경 변수 또는 secret manager에만 둔다.
