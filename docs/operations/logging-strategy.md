# 로깅 전략 (Logging Strategy)

> **목적**: 모든 시스템 동작을 추적 가능하게 만들어 디버깅과 모니터링을 용이하게 함

**Last Updated**: 2026-01-14

---

## 🎯 핵심 원칙

### 1. 구조화된 로깅 (Structured Logging)
```
모든 로그는 JSON 형식으로 기록
→ 파싱 가능, 검색 가능, 분석 가능
```

### 2. 로그 레벨 전략
| 레벨 | 용도 | 예시 |
|------|------|------|
| DEBUG | 개발/디버깅 정보 | "SQL query: SELECT * FROM stocks WHERE..." |
| INFO | 일반 정보 | "Server started on :8099", "User logged in" |
| WARN | 경고 (서비스 계속) | "Rate limit approaching", "Cache miss" |
| ERROR | 에러 (복구 시도 가능) | "Failed to fetch data, retrying..." |
| FATAL | 치명적 에러 (서비스 중단) | "Database connection failed" |

### 3. 로그 출력 대상
- **개발 환경**: 콘솔 (pretty format) + 파일
- **프로덕션**: 파일 only (JSON format)

---

## 📁 로그 파일 구조

```
logs/
├── app.log              # 일반 애플리케이션 로그
├── error.log            # ERROR 이상만
├── query.log            # DB 쿼리 로그
└── access.log           # HTTP 접근 로그
```

### 파일 Rotation 정책
- **크기**: 100MB 도달 시 rotation
- **보관**: 최근 30일
- **압축**: 7일 이상 된 로그는 gzip 압축

---

## 🔍 로그 구조 (JSON)

### 기본 필드
```json
{
  "timestamp": "2026-01-14T12:00:00Z",
  "level": "info",
  "message": "User logged in",
  "service": "aegis-v14-api",
  "version": "1.0.0"
}
```

### HTTP 요청 로그
```json
{
  "timestamp": "2026-01-14T12:00:00Z",
  "level": "info",
  "message": "HTTP request",
  "request_id": "req-123456",
  "method": "GET",
  "path": "/api/stocks",
  "status": 200,
  "duration_ms": 45,
  "ip": "192.168.1.1",
  "user_agent": "Mozilla/5.0..."
}
```

### 데이터베이스 쿼리 로그
```json
{
  "timestamp": "2026-01-14T12:00:00Z",
  "level": "debug",
  "message": "Database query",
  "request_id": "req-123456",
  "query": "SELECT * FROM stocks WHERE market = $1",
  "args": ["KOSPI"],
  "duration_ms": 12,
  "rows_affected": 100
}
```

### 에러 로그
```json
{
  "timestamp": "2026-01-14T12:00:00Z",
  "level": "error",
  "message": "Failed to fetch stock data",
  "request_id": "req-123456",
  "error": "connection timeout",
  "stack_trace": "...",
  "context": {
    "stock_code": "005930",
    "retry_count": 3
  }
}
```

---

## 🛠️ 구현 위치

### 로거 초기화
- **위치**: `internal/pkg/logger/logger.go`
- **책임**: zerolog 설정, 파일 핸들러, rotation 설정

### HTTP 미들웨어
- **위치**: `internal/api/middleware/logging.go`
- **책임**: 모든 HTTP 요청/응답 자동 로깅

### Request ID 미들웨어
- **위치**: `internal/api/middleware/request_id.go`
- **책임**: 각 요청에 고유 ID 부여, 컨텍스트 전파

### 데이터베이스 로거
- **위치**: `internal/infra/database/postgres/logger.go`
- **책임**: pgx 쿼리 로깅, 성능 측정

---

## 🎬 사용 예시

### 코드에서 로거 사용
```go
// 기본 로깅
log.Info().Msg("Server started")

// 구조화된 로깅
log.Info().
    Str("user_id", "user123").
    Int("count", 10).
    Msg("Stocks fetched")

// 에러 로깅 (스택 트레이스 포함)
log.Error().
    Err(err).
    Str("request_id", requestID).
    Str("stock_code", code).
    Msg("Failed to fetch stock")

// 컨텍스트에서 request_id 가져오기
requestID := ctx.Value("request_id").(string)
log.Info().
    Str("request_id", requestID).
    Msg("Processing request")
```

### HTTP 요청 자동 로깅
```
미들웨어가 자동으로 로깅:
→ 요청 시작
→ 요청 완료 (duration, status)
→ 에러 발생 시 상세 로그
```

---

## 🔧 설정

### .env 설정
```bash
# Logging
LOG_LEVEL=info              # debug, info, warn, error
LOG_FORMAT=json             # json, pretty
LOG_FILE_ENABLED=true       # 파일 로깅 활성화
LOG_FILE_PATH=./logs        # 로그 파일 디렉토리
LOG_ROTATION_SIZE=100       # MB 단위
LOG_RETENTION_DAYS=30       # 보관 기간
```

### 개발 환경 권장 설정
```bash
LOG_LEVEL=debug
LOG_FORMAT=pretty
LOG_FILE_ENABLED=true
```

### 프로덕션 권장 설정
```bash
LOG_LEVEL=info
LOG_FORMAT=json
LOG_FILE_ENABLED=true
```

---

## 🐛 디버깅 시나리오

### 시나리오 1: HTTP 요청 추적
1. HTTP 요청 로그에서 request_id 확인
2. request_id로 모든 관련 로그 검색
3. 시간순으로 정렬하여 흐름 파악

```bash
# request_id로 로그 필터링
cat logs/app.log | grep "req-123456" | jq
```

### 시나리오 2: 성능 문제 디버깅
1. access.log에서 느린 요청 찾기 (duration_ms > 1000)
2. request_id로 query.log 확인
3. 느린 쿼리 식별 및 최적화

```bash
# 1초 이상 걸린 요청 찾기
cat logs/access.log | jq 'select(.duration_ms > 1000)'
```

### 시나리오 3: 에러 원인 파악
1. error.log에서 에러 확인
2. request_id로 전체 컨텍스트 추적
3. stack_trace와 context 분석

```bash
# 특정 에러 검색
cat logs/error.log | jq 'select(.error | contains("timeout"))'
```

---

## 📊 로그 분석 도구 (향후)

### ELK Stack 연동 가능
- **Elasticsearch**: 로그 저장 및 검색
- **Logstash**: 로그 수집 및 변환
- **Kibana**: 시각화 및 대시보드

### Grafana Loki 연동 가능
- 경량 로그 aggregation
- Prometheus와 통합

---

## ⚠️ 주의사항

### 민감 정보 로깅 금지
```go
❌ 금지:
log.Info().Str("password", password).Msg("User login")
log.Debug().Str("api_key", apiKey).Msg("API call")

✅ 허용:
log.Info().Str("user_id", userID).Msg("User login")
log.Debug().Str("api_key_prefix", apiKey[:8]+"...").Msg("API call")
```

### 로그 레벨 적절히 선택
```
DEBUG: 개발 중에만 필요한 상세 정보
INFO: 프로덕션에서도 유용한 일반 정보
WARN: 주의가 필요하지만 서비스는 계속
ERROR: 문제 발생, 복구 시도 필요
FATAL: 서비스 중단 수준의 치명적 에러
```

### 성능 고려
- DEBUG 레벨은 프로덕션에서 비활성화
- 과도한 로깅은 성능 저하 유발
- 대용량 데이터는 길이 제한

---

## 📝 체크리스트

새 기능 추가 시 로깅 체크:
- [ ] 주요 시작/종료 지점에 INFO 로그
- [ ] 외부 API 호출 시 DEBUG 로그
- [ ] 에러 발생 시 ERROR 로그 (context 포함)
- [ ] 성능 측정이 필요한 곳에 duration 로깅
- [ ] request_id를 컨텍스트로 전파
- [ ] 민감 정보 제외 확인

---

## 참고 문서

- [zerolog GitHub](https://github.com/rs/zerolog)
- [로그 파일 관리](./log-rotation.md) (향후 작성)

---

**Version**: 1.0.0
**Last Updated**: 2026-01-14
