# Deployment Architecture

이 디렉터리는 DevOps 플랫폼 애플리케이션의 Kubernetes 배포 구성을 관리한다.

## 컴포넌트

| 컴포넌트      | 책임                            | 포트 |
| ------------- | ------------------------------- | ---- |
| Console       | 사용자 화면 및 서버 측 API 연결 | 3000 |
| Control Plane | 실행 정책, 승인, 작업 배정      | 8080 |
| Runner        | 작업 수신 및 실행 조율          | 없음 |

Console 서버와 Runner는 내부 Service 주소로 Control Plane에 접근한다.
브라우저는 Console의 동일 출처 API를 통해 접근하도록 구현한다.

## 디렉터리 책임

- `kubernetes/base`: 컴포넌트별 공통 Deployment와 Service
- `kubernetes/overlays`: 환경별 Namespace, 이미지, 복제 수
- `../infrastructure/terraform`: 클러스터, 네트워크, 외부 저장소 등 기반 자원

환경별 Namespace는 배포 대상을 구분한다.
Namespace만으로 네트워크 및 접근 권한 격리가 완성되는 것은 아니다.

## Hexagonal Architecture와의 관계

Hexagonal Architecture는 애플리케이션 내부의 의존성 경계를 정의한다.

- Domain: 비즈니스 규칙
- Application: 유스케이스 수행
- Ports: 외부와 상호작용하는 인터페이스
- Adapters: HTTP, 저장소, 실행 도구 등의 구현

`deploy/`는 이러한 애플리케이션의 실행 환경과 연결을 선언한다.
이 디렉터리 자체가 애플리케이션의 Adapter인 것은 아니다.

## 설정 계약

아래 환경변수는 애플리케이션 구현 시 지원해야 한다.

| 컴포넌트               | 변수              | 의미                          |
| ---------------------- | ----------------- | ----------------------------- |
| Console                | HOSTNAME / PORT   | 서버 바인딩 주소와 포트       |
| Console                | CONTROL_PLANE_URL | 서버 측 백엔드 주소           |
| Control Plane          | HTTP_ADDR         | HTTP 바인딩 주소              |
| Control Plane / Runner | LOG_LEVEL         | 로그 수준                     |
| Runner                 | CONTROL_PLANE_URL | 작업 요청 대상                |
| Runner                 | RUNNER_ID         | Pod UID 기반 실행 주체 식별자 |
| Runner                 | WORKSPACE_DIR     | 임시 작업 디렉터리            |

Runner의 식별자는 인증 수단이 아니다.
작업 요청 인증과 권한 검증은 별도로 구현한다.

## 구성 렌더링

저장소 루트에서 실행한다. 클러스터에는 적용하지 않는다.

```bash
kubectl kustomize deploy/kubernetes/overlays/dev
kubectl kustomize deploy/kubernetes/overlays/staging
kubectl kustomize deploy/kubernetes/overlays/prod
```

렌더링 성공은 YAML 조합이 가능하다는 의미다.
이미지 실행, API 연결, 클러스터 정책 적합성을 검증하지는 않는다.

## 실제 배포 연결 시 추가할 항목

- 실제 컨테이너 이미지와 고정된 digest
- 애플리케이션 설정 로딩
- DB, 산출물 저장소, 인증 및 Secret 연결
- 상태 확인 엔드포인트와 startup/readiness/liveness probe
- Runner 실행 Backend와 필요한 권한
- 작업 lease, 중복 처리 방지, 정상 종료 및 취소 처리
- 외부 접근을 위한 Ingress와 TLS
- 환경에 맞는 접근 제어 및 NetworkPolicy
- 측정에 근거한 리소스와 복제 수 설정

Runner의 emptyDir은 임시 공간이며 Pod 삭제 시 보존되지 않는다.
현재 리소스 값은 초기 예시이며 운영 용량 산정 결과가 아니다.

## 이미지 승격

동일한 빌드 산출물을 환경 간 승격한다.
환경마다 이미지를 다시 빌드하지 않는다.

환경별 overlay에서 검증된 이미지 digest를 지정한다.
GitOps Controller 연결은 후속 단계에서 구성한다.
