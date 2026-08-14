# 2026-08-14 — Secret Key credential surface

## 작업 목적과 이해한 내용

- 사용자-facing credential 명칭은 `Secret Key`로 확정됐다.
- Go server SDK의 새 public config/example은 `projectId + secretKey` 중심이어야 한다.
- 기존 Go SDK 사용자는 `Config.ServerAPIKey` 또는 `Config.ProjectAPIToken`을 쓰고 있을 수 있으므로 source compatibility는 유지해야 한다.

## 문제 진단 또는 기존 동작

- README와 HANDOFF의 기본 credential 설명이 이전 서버 key 명칭을 기준으로 되어 있었다.
- `Config`의 기본 새 필드는 `ServerAPIKey`였고, 더 오래된 `ProjectAPIToken`만 alias로 유지되고 있었다.
- 기존 alias를 제거하면 소비자 서버 코드가 깨질 수 있으므로 새 기본 필드를 추가하고 alias resolution만 확장하는 방식이 안전했다.

## 적용한 내용과 주요 변경 파일

- `client.go`
  - `Config.SecretKey`를 기본 credential field로 추가했다.
  - `Config.ServerAPIKey`와 `Config.ProjectAPIToken`은 deprecated alias로 유지했다.
  - credential 필드를 둘 이상 넘기면 값이 모두 같아야 하며, 다르면 HTTP 요청 전 `VALIDATION_ERROR`를 반환한다.
  - Authorization bearer 값은 내부 `secretKey`로 정규화해 사용한다.
  - user agent를 `spectra-notification-go/0.1.3`으로 갱신했다.
- `client_test.go`
  - Secret Key 기본 경로, 두 legacy alias 단독 경로, alias 일치/불일치 validation을 검증한다.
- `README.md`, `HANDOFF.md`
  - 기본 예시와 보안 경계를 `Config.SecretKey`와 `Secret Key` 중심으로 갱신했다.
  - 기존 credential field는 deprecated alias 섹션과 구현 경계에만 남겼다.
- `WORKLOG.md`
  - 이번 변경과 검증 결과를 저장소 인덱스에 추가했다.

## 실행한 검증과 결과

- `gofmt -w client.go client_test.go` — 통과
- `go test ./...` — 통과
- `go vet ./...` — 통과
- `go build ./...` — 통과
- `git diff --check` — 통과
- README/HANDOFF/client/test 범위 legacy exact phrase grep — 기본 사용자-facing 문구에는 남지 않고, deprecated `Config.ServerAPIKey`/`Config.ProjectAPIToken` alias 문맥만 남음

## 남은 작업, 미검증 항목 또는 주의사항

- 실제 `delivery.spectra.kr` public E2E는 수행하지 않았다.
- 실제 Console/Auth에서 Secret Key 발급/재발급 runtime은 SDK 저장소 범위 밖이라 검증하지 않았다.
- Delivery/Auth 서버가 반환하는 `PROJECT_TOKEN_UNAUTHORIZED` error code 이름은 현재 서버 계약 호환으로 유지했다. README 설명만 Secret Key 기준으로 바꿨다.
