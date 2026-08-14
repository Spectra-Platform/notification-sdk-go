# notification-sdk-go HANDOFF

마지막 코드 대조: 2026-08-14

## 목적과 소유 범위

`notification-sdk-go`는 Go backend에서 Spectra Email/Notification 기능을 server-to-server로 호출하기 위한 Go SDK입니다.

첫 공개 범위는 Secret Key 기반 단건 Email 발송과 Email Verification create/confirm입니다. 모두캠프 관리자 이메일 로그인처럼 로그인 전(pre-auth) 상태에서 브라우저 Notification SDK를 사용할 수 없고, Secret Key를 서버에만 보관해야 하는 경우를 대상으로 합니다.

## 확정된 결정

- Go SDK는 JS browser SDK와 public API를 그대로 복사하지 않습니다.
- 제품 용어와 오류 의미는 JS Notification SDK/Delivery 계약과 맞추되, Go답게 `context.Context`, typed struct, typed error, `http.Client`, timeout, base URL override를 제공합니다.
- 이 SDK는 server SDK입니다. 새 코드에서는 `Config.SecretKey`를 사용하며 브라우저/모바일 번들에서 사용하면 안 됩니다.
- `Config.ServerAPIKey`와 `Config.ProjectAPIToken`은 이전 Go SDK 사용자를 위한 deprecated alias입니다. credential 필드를 둘 이상 지정하면 값이 모두 같아야 하며, 다르면 SDK가 `VALIDATION_ERROR`를 반환합니다.
- 기본 public surface는 `client.Email.Send`와 `client.Email.Verifications.SendCode/SendLink/ConfirmCode/ConfirmLink`입니다.
- `client.Email.Send`는 Secret Key 단건 Email enqueue endpoint를 호출하며 `To`, `Subject`,
  `Text`, optional `HTML`, optional `Metadata`를 받습니다. 보내는 사람은 per-request 입력이 아니라
  Delivery Project sender 설정과 environment 정책으로 결정됩니다.
- Email Verification 기본 wrapper는 direct body (`Subject`, `Text`, optional `HTML`, optional
  `Metadata`)를 사용합니다. `TemplateID`와 optional `TemplateVersion`은 저장·게시된 template을
  재사용하는 advanced/legacy 경로입니다.
- direct body와 template mode는 상호 배타적입니다. SDK는 mixed mode를 `VALIDATION_ERROR`로 먼저
  막고, Delivery API도 `422`로 검증합니다.
- direct body는 Secret Key server grant에서만 지원합니다. browser/mobile app-user flow가
  Email verification을 노출하는 경우에는 별도 소비자 권한 경계와 template 기반 흐름을 따라야 합니다.
- `client.EmailVerifications`와 `client.Delivery.EmailVerifications`는 기존 호환 surface로 유지하지만 문서 기본 예시는 `client.Email`을 사용합니다.
- 기본 endpoint는 `https://delivery.spectra.kr`이며, 일반 기본 플로우에서는 URL 환경 변수를 요구하지 않습니다. local/internal Docker network용 `BaseURL` override는 advanced configuration으로만 안내합니다.
- `Create`는 `Idempotency-Key`를 사용합니다. 사용자가 주입하지 않으면 SDK가 `sdk:<random>` 형식으로 자동 생성합니다.
- Go/Python/Django/JS server SDK는 앞으로 같은 개념의 서버 Email interface를 가져야 합니다. 현재 Python/Django server SDK는 이 저장소에서 생성하지 않았고 후속 필요 작업으로 남깁니다.

## 현재 구현 경계

구현됨:

- Go module: `github.com/Spectra-Platform/notification-sdk-go`
- `notification.NewClient(notification.Config{...})`
- Secret Key bearer Authorization
- deprecated config alias: `Config.ServerAPIKey`, `Config.ProjectAPIToken`
- Secret Key 단건 Email send: `client.Email.Send`
- Email verification create/confirm
- Email verification 기본 wrapper: `SendCode`, `SendLink`, `ConfirmCode`, `ConfirmLink`
- 기본 `Method=code`, 기본 `ExpiresInSeconds=600`
- direct `Subject`/`Text`/optional `HTML`/optional `Metadata` request encoding
- template `TemplateID`/optional `TemplateVersion` request encoding
- code/link 입력 검증
- idempotency key 자동 생성 및 제약 검증
- API error envelope parsing
- request id, retry-after parsing
- network/cancel/malformed response typed error
- `httptest` 기반 단위 테스트

아직 구현/검증하지 않음:

- 실제 `delivery.spectra.kr` public E2E
- 모두캠프 Go backend 적용
- Python/Django server SDK
- push 발송, device registration 등 Notification 전체 영역
- GitHub release/tag/package distribution

## 외부 의존성

- Delivery API endpoint:
  - `POST /platform/v1/projects/{project_id}/email/delivery-requests`
  - `POST /platform/v1/projects/{project_id}/email/verifications`
  - `POST /platform/v1/projects/{project_id}/email/verifications/{verification_id}/confirmations`
- Secret Key:
  - Console/Auth에서 프로젝트별로 발급한 서버 전용 secret을 사용합니다.
  - 발송 가능 서비스는 key 자체가 아니라 Console 대시보드 서비스 토글로 관리합니다.
  - key project와 path project가 일치해야 합니다.
- environment는 Secret Key 검증 결과에 의해 결정됩니다. SDK의 `Environment`는 소비자 설정 구분과 향후 endpoint 선택을 위한 값이며 현재 request body에 보내지 않습니다.

## 변경 시 함께 확인할 계약

- `delivery-platform` email verification request/response/error contract
- `delivery-platform` project Email delivery request/response/error contract
- `auth-platform` Secret Key validation contract
- `platform-docs` server SDK 문서
- JS Notification SDK와 오류 의미 alignment
- 모두캠프 Go backend adapter 적용 방식

## 주의사항

- Secret Key 원문을 README, WORKLOG, test fixture, log에 남기지 않습니다.
- browser/mobile SDK에 Secret Key나 provider credential을 넣는 방향으로 안내하지 않습니다.
- Delivery API는 현재 템플릿 또는 verification 리소스 미존재를 `EMAIL_RESOURCE_NOT_FOUND`로 반환합니다. 템플릿 발행 충돌은 운영 템플릿 API의 `EMAIL_TEMPLATE_PUBLISH_CONFLICT`이며, 일반 email verification create/confirm 흐름의 not published 의미와 1:1로 분리되어 있지는 않습니다.
- direct body placeholder render, SMTP mailbox 도착, verified sender DNS와 public endpoint E2E는
  Delivery producer/runtime의 별도 검증 범위입니다. SDK unit test는 요청 contract만 확인합니다.
- Project Email의 `From`은 SDK request field가 아닙니다. caller가 보내는 사람을 바꾸려면 Delivery
  Project sender 설정/API 계약을 사용해야 하며, SDK 기본 `SendEmailInput`에 per-request sender를
  임의로 추가하지 않습니다.
