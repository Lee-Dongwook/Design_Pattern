# Infrastructure as Code

Terraform은 애플리케이션이 사용하는 기반 자원의 구성을 담당한다.

## 현재 구현 범위

현재는 환경별 Root Module과 공통 환경 설정 모듈을 제공한다.

- 프로젝트 및 환경 값 검증
- 자원 이름 접두사 생성
- 소유자 및 환경 메타데이터 생성

현재 구성에는 Provider와 resource가 없다.
plan을 실행해도 클러스터, DB, 네트워크는 생성되지 않는다.
출력값 변경만 계획될 수 있다.

## 구조

- modules/environment-context: 공통 환경 설정과 검증
- environments/dev: 개발 환경 진입점
- environments/staging: 검증 환경 진입점
- environments/prod: 운영 환경 진입점

환경별 디렉터리를 사용한다.
현재 구조에서는 Terraform workspace로 환경을 추가 분리하지 않는다.

## 관리 경계

| 영역                     | 책임                           |
| ------------------------ | ------------------------------ |
| infrastructure/local     | 로컬 개발용 DB 및 Registry     |
| infrastructure/terraform | 기반 자원 프로비저닝           |
| deploy/kubernetes        | 애플리케이션 배포 및 Namespace |

동일한 자원을 Terraform과 배포 도구에서 중복 관리하지 않는다.

## 검증

저장소 루트에서 실행한다.

```bash
terraform fmt -recursive infrastructure/terraform

terraform -chdir=infrastructure/terraform/environments/dev init -backend=false
terraform -chdir=infrastructure/terraform/environments/dev validate
terraform -chdir=infrastructure/terraform/environments/dev plan
```

staging과 prod도 각 디렉터리에서 독립적으로 검증한다.

## Provider 연결 시 구현할 모듈

실제 공급자를 정한 후 필요한 모듈을 추가한다.

- network
- cluster
- database
- artifact storage
- container registry

공급자의 자원 모델에 맞춰 입력과 출력을 정의한다.
리소스를 구현하기 전부터 동일한 인터페이스를 강제하지 않는다.

## State 및 자격 증명

현재 backend 선언이 없으므로 기본 local backend를 사용한다.
-backend=false는 init의 backend 초기화를 생략하는 옵션이다.
원격 State가 구성되었다는 의미가 아니다.

실제 공유 인프라를 생성하기 전에 다음을 구성한다.

- 환경별로 분리된 원격 State
- State 잠금과 접근 제어
- 실행 주체의 공급자 인증
- Provider 버전 제약

비밀번호와 토큰은 terraform.tfvars에 커밋하지 않는다.
State와 저장된 plan 파일에도 민감 정보가 포함될 수 있다.

Provider 도입 후 생성되는 .terraform.lock.hcl은 커밋한다.
