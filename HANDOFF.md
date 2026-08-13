# notification-sdk-go HANDOFF

마지막 코드 대조: 2026-08-13

## 목적과 소유 범위

`notification-sdk-go`는 Go backend에서 Spectra Notification/Delivery 기능을 server-to-server로 호출하기 위한 Go SDK입니다.

첫 공개 범위는 Email Verification create/confirm입니다. 모두캠프 관리자 이메일 로그인처럼 로그인 전(pre-auth) 상태에서 브라우저 Notification SDK를 사용할 수 없고, Project API Token을 서버에만 보관해야 하는 경우를 대상으로 합니다.

## 확정된 결정

- Go SDK는 JS browser SDK와 public API를 그대로 복사하지 않습니다.
- 제품 용어와 오류 의미는 JS Notification SDK/Delivery 계약과 맞추되, Go답게 `context.Context`, typed struct, typed error, `http.Client`, timeout, base URL override를 제공합니다.
- 이 SDK는 server SDK입니다. `ProjectAPIToken`을 받으며 브라우저/모바일 번들에서 사용하면 안 됩니다.
- 첫 범위는 `client.Delivery.EmailVerifications.Create/Confirm`입니다.
- `client.EmailVerifications`는 짧은 alias로 유지하지만 문서 예시는 `client.Delivery.EmailVerifications`를 기본으로 사용합니다.
- 기본 endpoint는 `https://delivery.spectra.kr`이며, local/internal Docker network용 `BaseURL` override를 허용합니다.
- `Create`는 `Idempotency-Key`를 사용합니다. 사용자가 주입하지 않으면 SDK가 `sdk:<random>` 형식으로 자동 생성합니다.

## 현재 구현 경계

구현됨:

- Go module: `github.com/Spectra-Platform/notification-sdk-go`
- `notification.NewClient(notification.Config{...})`
- Project API Token bearer Authorization
- Email verification create/confirm
- 기본 `Method=code`, 기본 `ExpiresInSeconds=600`
- code/link 입력 검증
- idempotency key 자동 생성 및 제약 검증
- API error envelope parsing
- request id, retry-after parsing
- network/cancel/malformed response typed error
- `httptest` 기반 단위 테스트

아직 구현/검증하지 않음:

- 실제 `delivery.spectra.kr` public E2E
- 모두캠프 Go backend 적용
- push 발송, device registration 등 Notification 전체 영역
- GitHub release/tag/package distribution

## 외부 의존성

- Delivery API endpoint:
  - `POST /platform/v1/projects/{project_id}/email/verifications`
  - `POST /platform/v1/projects/{project_id}/email/verifications/{verification_id}/confirmations`
- Project API Token:
  - audience: `email`
  - scope: `email.verification.send`
  - token project와 path project가 일치해야 합니다.
- environment는 Project API Token introspection 결과에 의해 결정됩니다. SDK의 `Environment`는 소비자 설정 구분과 향후 endpoint 선택을 위한 값이며 현재 request body에 보내지 않습니다.

## 변경 시 함께 확인할 계약

- `delivery-platform` email verification request/response/error contract
- `auth-platform` Project API Token introspection and audience/scope contract
- `platform-docs` server SDK 문서
- JS Notification SDK와 오류 의미 alignment
- 모두캠프 Go backend adapter 적용 방식

## 주의사항

- Project API Token 원문을 README, WORKLOG, test fixture, log에 남기지 않습니다.
- browser/mobile SDK에 Project API Token을 넣는 방향으로 안내하지 않습니다.
- Delivery API는 현재 템플릿 또는 verification 리소스 미존재를 `EMAIL_RESOURCE_NOT_FOUND`로 반환합니다. 템플릿 발행 충돌은 운영 템플릿 API의 `EMAIL_TEMPLATE_PUBLISH_CONFLICT`이며, 일반 email verification create/confirm 흐름의 not published 의미와 1:1로 분리되어 있지는 않습니다.
