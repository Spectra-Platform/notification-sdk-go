# 2026-08-14 — Server API Key alias for Email Go SDK

## 작업 목적과 이해한 내용

- Console/Auth가 기능별 token preset 대신 하나의 Server API Key에 기능을 포함하는 구조로 바뀐다.
- Go server SDK 기본 문서와 public config는 `email_sender` 같은 내부 preset이나 legacy Project API token 용어가 아니라, Email 기능이 포함된 Server API Key를 기준으로 설명해야 한다.
- 기존 Go SDK 사용자가 `Config.ProjectAPIToken`을 쓰고 있을 수 있으므로 source compatibility는 유지해야 한다.

## 문제 진단 또는 기존 동작

- README 기본 예시와 보안 경계가 `ProjectAPIToken`과 Project API Token 중심이었다.
- `Config`에는 `ProjectAPIToken`만 있어 새 Console/Auth credential model을 코드에서 직관적으로 표현하기 어려웠다.
- 기존 typed error surface는 유지해야 하며, config validation failure는 계속 `VALIDATION_ERROR`로 반환되어야 했다.

## 적용한 내용과 주요 변경 파일

- `client.go`
  - `Config.ServerAPIKey`를 추가했다.
  - `Config.ProjectAPIToken`은 legacy alias로 유지했다.
  - `ServerAPIKey`가 없고 `ProjectAPIToken`만 있으면 기존처럼 Authorization bearer 값으로 사용한다.
  - 두 값이 모두 주어졌는데 다르면 HTTP 요청 전에 `VALIDATION_ERROR`를 반환한다.
  - package/user-agent version 문구는 변경하지 않았다.
- `client_test.go`
  - Server API Key 단독, legacy alias 단독, alias 일치, 누락, 불일치 validation test를 추가했다.
  - 기존 request 테스트 helper를 `ServerAPIKey` 기본 경로로 바꿔 Authorization header 호환을 확인했다.
- `README.md`
  - 기본 client 생성 예시를 `ServerAPIKey`와 `client.Email.Send` 중심으로 유지했다.
  - legacy Project API token 용어는 호환 alias 섹션에만 남겼다.
- `HANDOFF.md`
  - 현재 기준 credential을 Email 기능이 포함된 Server API Key로 정리했다.
  - `ProjectAPIToken` alias와 불일치 validation을 확정 결정으로 기록했다.

## 실행한 검증과 결과

- `gofmt -w client.go client_test.go` — 통과
- `go test ./...` — 통과
- `go vet ./...` — 통과
- `go build ./...` — 통과
- `git diff --check` — 통과

## 남은 작업, 미검증 항목 또는 주의사항

- 실제 `delivery.spectra.kr` public E2E는 수행하지 않았다.
- 실제 Console/Auth에서 발급한 Server API Key로 Delivery runtime 검증은 수행하지 않았다.
- 모두의캠프 Go backend 적용은 별도 소비자 작업이다.
- 서버/API가 반환하는 `PROJECT_TOKEN_UNAUTHORIZED` code 이름은 현재 Delivery/Auth 오류 계약 호환으로 유지했다. README 의미 설명만 Server API Key 기준으로 바꿨다.
