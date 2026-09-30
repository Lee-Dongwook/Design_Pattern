# Local Infrastructure

호스트에서 실행하는 Console, Control Plane, Runner의 개발 의존성을 제공한다.

## 제공 서비스

| 서비스     | 기본 접속 주소 | 용도                        |
| ---------- | -------------- | --------------------------- |
| PostgreSQL | localhost:5432 | 실행 정보와 메타데이터 저장 |
| Registry   | localhost:5001 | 컨테이너 이미지 저장        |

포트는 호스트의 loopback 주소에만 공개한다.
Registry는 로컬 개발용 HTTP 구성이다.

## 실행

이 디렉터리에서 최초 1회 실행한다.

```bash
cp .env.example .env
```

설정을 확인하고 실행한다.

```bash
docker compose config --quiet
docker compose up -d
docker compose ps
```

## 확인

```bash
docker compose exec postgres sh -c \
  'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT 1;"'

curl -fsS http://localhost:5001/v2/
```

Registry 포트를 변경했다면 curl 주소도 변경한다.

## 로그 및 종료

```bash
docker compose logs -f
docker compose down
```

일반 down은 데이터 볼륨을 유지한다.
볼륨까지 삭제하면 DB와 이미지 데이터가 제거된다.

## 애플리케이션 연결

호스트에서 실행하는 애플리케이션은 localhost의 공개 포트를 사용한다.

Control Plane을 PostgreSQL 모드로 실행하려면 아래처럼 `DATABASE_URL`을 설정한다.

```bash
export DATABASE_URL='postgres://devops:local-dev-only@localhost:5432/devops?sslmode=disable'
cd ../../services/control-plane
go run ./cmd/server
```

`DATABASE_URL`이 없으면 Control Plane은 기존과 동일하게 인메모리 저장소로 동작한다.

Runner는 기본적으로 각 task를 Docker 컨테이너에서 실행한다. 실행 컨테이너는
네트워크를 사용하지 않고, read-only 파일 시스템·권한 제거·CPU/메모리/PID 제한을 적용한다.
따라서 Docker Desktop 또는 Docker Engine이 실행 중이어야 한다.

```bash
cd ../../workers/runner
CONTROL_PLANE_URL=http://localhost:8080 go run ./cmd/runner
```

호스트에서 직접 명령을 실행하는 `RUNNER_EXECUTOR=local`은 개발 확인 외에는 사용하지 않는다.

추후 같은 Compose 네트워크에 애플리케이션을 추가하면
postgres:5432, registry:5000 주소를 사용한다.

DB 마이그레이션은 services/control-plane/migrations에서 관리한다.
.env 값을 변경해도 기존 PostgreSQL 데이터 볼륨의 계정이 자동 변경되지는 않는다.

## 범위

- Kubernetes 클러스터를 생성하지 않는다.
- Kubernetes에서 localhost는 각 Pod 또는 노드 자신을 가리킨다.
- 로컬 Registry를 Kubernetes와 연결할 때는 노드에서 접근 가능한 주소와
  컨테이너 런타임의 Registry 설정이 별도로 필요하다.
- 일반 산출물 및 로그용 Object Storage는 후속 단계에서 추가한다.
- 이미지의 메이저 태그는 로컬 편의를 위한 설정이다.
  재현 가능한 환경이 필요하면 검증한 digest로 고정한다.
