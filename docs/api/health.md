# Health Check API

> **목적**: 시스템 전체 및 개별 컴포넌트의 상태를 확인하기 위한 API

**Last Updated**: 2026-01-14

---

## 📋 개요

Health Check API는 시스템의 가용성과 상태를 모니터링하기 위해 사용됩니다.
- 로드 밸런서의 health check 대상
- 모니터링 시스템의 상태 확인
- 개발/디버깅 시 시스템 상태 점검

---

## 🌐 엔드포인트

### GET /health

**목적**: 전체 시스템 상태 확인 (간단한 liveness check)

#### Request
```http
GET /health HTTP/1.1
Host: localhost:8099
```

#### Response

**200 OK** (시스템 정상):
```json
{
  "status": "healthy",
  "timestamp": "2026-01-14T12:00:00Z"
}
```

**503 Service Unavailable** (시스템 비정상):
```json
{
  "status": "unhealthy",
  "timestamp": "2026-01-14T12:00:00Z"
}
```

#### 특징
- 인증 불필요
- 데이터베이스 연결 체크하지 않음 (빠른 응답)
- 로드 밸런서의 liveness probe용

---

### GET /health/ready

**목적**: 시스템 준비 상태 확인 (readiness check)

#### Request
```http
GET /health/ready HTTP/1.1
Host: localhost:8099
```

#### Response

**200 OK** (시스템 준비 완료):
```json
{
  "status": "ready",
  "timestamp": "2026-01-14T12:00:00Z",
  "checks": {
    "database": "ok",
    "redis": "ok"
  }
}
```

**503 Service Unavailable** (시스템 준비 안됨):
```json
{
  "status": "not_ready",
  "timestamp": "2026-01-14T12:00:00Z",
  "checks": {
    "database": "ok",
    "redis": "error"
  },
  "message": "Redis connection failed"
}
```

#### 특징
- 인증 불필요
- 데이터베이스, Redis 등 의존성 체크
- 로드 밸런서의 readiness probe용
- 하나라도 실패하면 503 반환

---

### GET /api/health/detailed

**목적**: 상세한 시스템 상태 정보 조회

#### Request
```http
GET /api/health/detailed HTTP/1.1
Host: localhost:8099
```

#### Response

**200 OK**:
```json
{
  "data": {
    "status": "healthy",
    "version": "1.0.0",
    "uptime_seconds": 3600,
    "timestamp": "2026-01-14T12:00:00Z",
    "components": {
      "database": {
        "status": "healthy",
        "response_time": "5ms",
        "details": {
          "active_conns": 3,
          "idle_conns": 7,
          "total_conns": 10,
          "max_conns": 25
        }
      },
      "redis": {
        "status": "healthy",
        "response_time": "2ms",
        "details": {
          "pool_size": 10,
          "idle_conns": 8
        }
      }
    }
  },
  "meta": {
    "request_id": "req-abc123",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

**200 OK** (일부 컴포넌트 degraded):
```json
{
  "data": {
    "status": "degraded",
    "version": "1.0.0",
    "uptime_seconds": 3600,
    "timestamp": "2026-01-14T12:00:00Z",
    "components": {
      "database": {
        "status": "degraded",
        "response_time": "50ms",
        "details": {
          "active_conns": 23,
          "idle_conns": 0,
          "total_conns": 25,
          "max_conns": 25
        },
        "message": "Connection pool nearly exhausted"
      },
      "redis": {
        "status": "healthy",
        "response_time": "2ms"
      }
    }
  },
  "meta": {
    "request_id": "req-abc123",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

#### 특징
- 인증 불필요 (향후 추가 가능)
- 각 컴포넌트의 상세 상태 포함
- 개발/디버깅에 유용
- 전체 시스템 상태: `healthy`, `degraded`, `unhealthy`

---

## 🔍 상태 정의

### 전체 시스템 상태
| 상태 | 조건 | HTTP 코드 |
|------|------|----------|
| `healthy` | 모든 컴포넌트 정상 | 200 |
| `degraded` | 일부 컴포넌트 degraded, 서비스 가능 | 200 |
| `unhealthy` | 핵심 컴포넌트 실패, 서비스 불가 | 503 |
| `ready` | 모든 컴포넌트 준비 완료 | 200 |
| `not_ready` | 하나 이상 준비 안됨 | 503 |

### 컴포넌트 상태
| 상태 | 의미 |
|------|------|
| `healthy` | 정상 동작 |
| `degraded` | 동작하지만 성능 저하 (예: connection pool 부족) |
| `unhealthy` | 동작 불가 |

---

## 📊 체크 항목

### Database
- **체크 방법**: Ping + connection pool stats
- **정상 조건**:
  - Ping 성공
  - Response time < 100ms
  - Available connections > 0
- **Degraded 조건**:
  - Response time 100ms ~ 1s
  - Active connections >= MaxConns - 2
- **비정상 조건**:
  - Ping 실패
  - Response time > 1s

### Redis (향후 추가)
- **체크 방법**: PING 명령
- **정상 조건**: PONG 응답, response time < 50ms
- **Degraded 조건**: Response time 50ms ~ 500ms
- **비정상 조건**: 응답 없음 또는 response time > 500ms

---

## 🎯 사용 시나리오

### 로드 밸런서 설정
```yaml
# Kubernetes liveness probe
livenessProbe:
  httpGet:
    path: /health
    port: 8099
  initialDelaySeconds: 10
  periodSeconds: 10

# Kubernetes readiness probe
readinessProbe:
  httpGet:
    path: /health/ready
    port: 8099
  initialDelaySeconds: 5
  periodSeconds: 5
```

### 모니터링 시스템
```bash
# Prometheus metrics (향후)
curl http://localhost:8099/metrics

# 상세 상태 확인
curl http://localhost:8099/api/health/detailed
```

### 개발/디버깅
```bash
# 빠른 상태 확인
curl http://localhost:8099/health

# 준비 상태 확인
curl http://localhost:8099/health/ready

# 상세 정보
curl http://localhost:8099/api/health/detailed | jq
```

---

## ⚙️ 구현 위치

### Handler
- **위치**: `internal/api/handlers/health.go`
- **책임**: Health check 로직

```go
type HealthHandler struct {
    dbPool *postgres.Pool
    // redis *redis.Client (향후)
    startTime time.Time
    version string
}

func (h *HealthHandler) Health(c *gin.Context)
func (h *HealthHandler) Ready(c *gin.Context)
func (h *HealthHandler) Detailed(c *gin.Context)
```

### Router
- **위치**: `internal/api/router.go`
- **경로**:
  - `GET /health` → HealthHandler.Health
  - `GET /health/ready` → HealthHandler.Ready
  - `GET /api/health/detailed` → HealthHandler.Detailed

---

## 📝 예시 요청/응답

### 예시 1: 전체 시스템 정상
```bash
$ curl http://localhost:8099/health
{
  "status": "healthy",
  "timestamp": "2026-01-14T12:00:00Z"
}
```

### 예시 2: 준비 상태 확인
```bash
$ curl http://localhost:8099/health/ready
{
  "status": "ready",
  "timestamp": "2026-01-14T12:00:00Z",
  "checks": {
    "database": "ok"
  }
}
```

### 예시 3: 상세 정보 (정상)
```bash
$ curl http://localhost:8099/api/health/detailed
{
  "data": {
    "status": "healthy",
    "version": "1.0.0",
    "uptime_seconds": 3600,
    "timestamp": "2026-01-14T12:00:00Z",
    "components": {
      "database": {
        "status": "healthy",
        "response_time": "5ms",
        "details": {
          "active_conns": 3,
          "idle_conns": 7,
          "total_conns": 10,
          "max_conns": 25
        }
      }
    }
  },
  "meta": {
    "request_id": "req-abc123",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

### 예시 4: 데이터베이스 문제 (degraded)
```bash
$ curl http://localhost:8099/api/health/detailed
{
  "data": {
    "status": "degraded",
    "version": "1.0.0",
    "uptime_seconds": 7200,
    "timestamp": "2026-01-14T12:00:00Z",
    "components": {
      "database": {
        "status": "degraded",
        "response_time": "150ms",
        "details": {
          "active_conns": 24,
          "idle_conns": 1,
          "total_conns": 25,
          "max_conns": 25
        },
        "message": "Connection pool nearly exhausted"
      }
    }
  },
  "meta": {
    "request_id": "req-abc123",
    "timestamp": "2026-01-14T12:00:00Z"
  }
}
```

---

## ✅ 체크리스트

Health Check API 구현 시:
- [ ] `/health` 엔드포인트 구현 (liveness)
- [ ] `/health/ready` 엔드포인트 구현 (readiness)
- [ ] `/api/health/detailed` 엔드포인트 구현
- [ ] Database health check 통합
- [ ] 적절한 HTTP 상태 코드 반환
- [ ] 응답 시간 측정
- [ ] 로깅 추가 (요청은 로깅하지만 verbose하지 않게)

---

## 🔗 관련 문서

- [API 공통 스펙](./common.md)
- [Database 연결](../../backend/internal/infra/database/postgres/health.go)

---

**Version**: 1.0.0
**Last Updated**: 2026-01-14
