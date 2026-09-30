# Design_Pattern

### product structure

모노레포 안에서 Console / Control Plane / Runner / Infrastructure를 분리
Hexagonal은 핵심 정책이 있는 Control Plane에 우선 적용
Runner는 실행 책임에 집중

Language : Console = TypeScript, Control Plane·Runner = Go, Infrastructure = HCL

```
devops-platform/
├── apps/
│   └── console/                         # TypeScript · React / Next.js
│       ├── src/
│       │   ├── app/                     # 라우팅·앱 구성
│       │   ├── features/
│       │   │   ├── pipelines/
│       │   │   ├── runs/
│       │   │   ├── releases/
│       │   │   ├── environments/
│       │   │   └── approvals/
│       │   └── shared/
│       │       ├── api/                 # API 클라이언트 연결
│       │       ├── ui/
│       │       └── lib/
│       └── package.json
│
├── services/
│   └── control-plane/                   # Go · 정책 및 실행 제어
│       ├── cmd/
│       │   └── server/
│       │       └── main.go              # 의존성 조립·서버 시작
│       ├── internal/
│       │   ├── domain/
│       │   │   ├── pipeline/            # 정의·DAG 유효성 규칙
│       │   │   ├── run/                 # 실행·작업 상태 전이
│       │   │   ├── artifact/            # 산출물 식별·메타데이터
│       │   │   ├── release/             # 배포 버전·승격 규칙
│       │   │   ├── environment/         # 배포 대상·환경 정책
│       │   │   └── approval/            # 승인 조건·결정
│       │   ├── application/
│       │   │   ├── commands/            # 실행·승인·취소 유스케이스
│       │   │   ├── queries/             # 목록·상세 조회
│       │   │   └── orchestration/       # 실행 가능 작업 계산·배정
│       │   ├── ports/
│       │   │   ├── inbound/             # 유스케이스 인터페이스
│       │   │   └── outbound/            # 저장·작업 전달 등 인터페이스
│       │   └── adapters/
│       │       ├── inbound/
│       │       │   ├── http/
│       │       │   ├── webhook/
│       │       │   └── worker_events/   # 작업 결과 수신
│       │       └── outbound/
│       │           ├── postgres/
│       │           ├── task_dispatch/
│       │           ├── artifact_store/
│       │           └── scm/            # GitHub·GitLab 연동
│       ├── migrations/
│       ├── tests/
│       │   └── integration/
│       └── go.mod
│
├── workers/
│   └── runner/                          # Go · 작업 실행
│       ├── cmd/
│       │   └── runner/
│       │       └── main.go
│       ├── internal/
│       │   ├── agent/                   # 작업 수신·결과 보고
│       │   ├── execution/               # 실행·타임아웃·취소
│       │   ├── executor/                # 실행기 인터페이스
│       │   ├── adapters/
│       │   │   └── container/           # 격리된 컨테이너 실행
│       │   ├── workspace/               # 소스·임시 작업 공간
│       │   └── logs/                    # 로그 수집·전송
│       └── go.mod
│
├── contracts/                           # 언어 간 통신 계약의 원본
│   ├── http/
│   │   └── openapi.yaml
│   ├── events/                          # 작업 상태·완료 이벤트
│   └── tasks/
│       └── task-spec.schema.json        # 이미지·명령·입출력·제한
│
├── packages/
│   └── api-client-ts/                   # 계약에서 생성한 TS 클라이언트
│
├── toolchains/                          # 언어별 빌드 실행 환경
│   ├── c-cpp/
│   ├── go/
│   ├── java/
│   ├── python/
│   ├── typescript/
│   └── terraform/
│
├── infrastructure/
│   ├── terraform/
│   │   ├── modules/                     # 재사용 인프라 모듈
│   │   └── environments/
│   │       ├── dev/
│   │       ├── staging/
│   │       └── prod/
│   └── local/
│       └── compose.yaml                # 로컬 DB·저장소 등
│
├── deploy/                              # 애플리케이션 배포 선언
│   └── kubernetes/
│       ├── base/
│       └── overlays/
│           ├── dev/
│           ├── staging/
│           └── prod/
│
├── pipelines/                           # 플랫폼이 실행할 파이프라인 예시
│   └── examples/
├── tests/
│   └── e2e/                            # Console → 제어 → 실행 통합 검증
├── docs/
│   ├── architecture/
│   ├── adr/                            # 설계 결정과 근거
│   └── runbooks/
├── scripts/
├── .github/
│   └── workflows/                      # 이 저장소 자체의 CI/CD
├── go.work
├── pnpm-workspace.yaml
└── README.md
```

### dependency rules

1. domain은 HTTP·DB·클라우드 SDK·실행 도구를 알지 않는다.
2. application은 도메인과 Port를 사용해 유스케이스를 수행한다.
3. adapters가 Port를 구현하고, main.go에서 구현체를 연결한다.
4. Console·Control Plane·Runner는 서로의 내부 코드를 가져오지 않고 contracts에 정의한 계약으로 통신한다.
