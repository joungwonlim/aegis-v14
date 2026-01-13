# Database (데이터베이스 설계)

이 폴더는 v14 시스템의 데이터베이스 설계 문서를 포함합니다.

---

## 📋 문서 목록

### 1. erd.md
- **목적**: ERD (Entity Relationship Diagram)
- **내용**:
  - 전체 테이블 관계도
  - Mermaid로 작성된 ERD
  - 주요 관계 설명

### 2. schema.md
- **목적**: 전체 테이블 스키마 정의
- **내용**:
  - 모든 테이블 정의
  - 컬럼 타입 및 제약사항
  - 인덱스 목록
  - Foreign Key 관계

### 3. indexes.md
- **목적**: 인덱스 전략
- **내용**:
  - 성능 최적화를 위한 인덱스
  - 각 인덱스의 목적 및 근거
  - 쿼리 패턴 분석

### 4. migration-plan.md
- **목적**: 마이그레이션 계획
- **내용**:
  - 마이그레이션 파일 순서
  - Rollback 전략
  - 데이터 마이그레이션 계획

---

## 🎯 데이터베이스 설계 원칙

### 1. 정규화 (Normalization)
- 최소 3NF 준수
- 중복 데이터 최소화
- 단, 성능을 위한 적절한 비정규화 허용

### 2. 명명 규칙 (Naming Convention)
```sql
-- 테이블: snake_case, 복수형
stocks, stock_prices, trading_signals

-- 컬럼: snake_case
stock_code, created_at, updated_at

-- 인덱스: idx_{table}_{column}
idx_stocks_market, idx_prices_stock_code_traded_at

-- Foreign Key: fk_{table}_{ref_table}
fk_prices_stocks
```

### 3. 타임스탬프 (Timestamps)
모든 테이블에 필수:
```sql
created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
```

### 4. Primary Key
- UUID 또는 Auto-increment ID 사용
- 비즈니스 키는 Unique 제약으로 별도 관리

---

## 📐 ERD 작성 가이드

### Mermaid 사용 (권장)

```markdown
\`\`\`mermaid
erDiagram
    STOCKS ||--o{ PRICES : has
    STOCKS {
        varchar code PK
        varchar name
        varchar market
        timestamp created_at
    }
    PRICES {
        uuid id PK
        varchar stock_code FK
        decimal price
        bigint volume
        timestamp traded_at
        timestamp created_at
    }
\`\`\`
```

---

## 🗄️ 테이블 카테고리

### 1. 마스터 데이터
- `stocks` - 종목 기본 정보
- `markets` - 시장 정보

### 2. 시계열 데이터
- `stock_prices` - 가격 데이터
- `stock_volumes` - 거래량 데이터

### 3. 시그널 데이터
- `signals` - 생성된 시그널
- `signal_scores` - 시그널 점수

### 4. 포트폴리오 데이터
- `portfolios` - 포트폴리오 구성
- `positions` - 포지션 정보

### 5. 거래 데이터
- `orders` - 주문 내역
- `trades` - 체결 내역

### 6. 감사 데이터
- `performance_logs` - 성과 분석
- `audit_logs` - 감사 로그

---

## 🔍 쿼리 패턴

설계 시 고려해야 할 주요 쿼리 패턴:

### 1. 실시간 조회
```sql
-- 최신 가격 조회 (자주 사용)
SELECT * FROM stock_prices
WHERE stock_code = ?
ORDER BY traded_at DESC
LIMIT 1;
```
→ 인덱스: `idx_prices_stock_code_traded_at`

### 2. 시계열 조회
```sql
-- 특정 기간 가격 조회
SELECT * FROM stock_prices
WHERE stock_code = ?
  AND traded_at BETWEEN ? AND ?
ORDER BY traded_at ASC;
```
→ 인덱스: `idx_prices_stock_code_traded_at`

### 3. 집계 쿼리
```sql
-- 일일 거래량 합계
SELECT stock_code, DATE(traded_at), SUM(volume)
FROM stock_prices
WHERE DATE(traded_at) = ?
GROUP BY stock_code, DATE(traded_at);
```
→ 인덱스: `idx_prices_traded_at`

---

## 📊 데이터 타입 가이드

| 용도 | PostgreSQL 타입 | 예시 |
|------|----------------|------|
| ID (Auto) | BIGSERIAL | 1, 2, 3... |
| ID (UUID) | UUID | `550e8400-e29b-41d4-a716-446655440000` |
| 종목코드 | VARCHAR(10) | `005930` |
| 금액 | DECIMAL(15,2) | `123456.78` |
| 거래량 | BIGINT | `1234567890` |
| 비율 | DECIMAL(5,2) | `3.25` (%) |
| 날짜 | DATE | `2024-01-13` |
| 날짜+시간 | TIMESTAMP | `2024-01-13 15:30:00` |
| 불린 | BOOLEAN | `true`, `false` |
| JSON | JSONB | `{"key": "value"}` |

---

## 🔄 마이그레이션 순서 예시

```
000001_create_stocks_table.sql
000002_create_prices_table.sql
000003_create_signals_table.sql
000004_add_indexes_to_prices.sql
000005_create_portfolios_table.sql
...
```

---

## ✅ 설계 검증 체크리스트

데이터베이스 설계 완료 시:

- [ ] ERD 다이어그램 작성
- [ ] 모든 테이블 스키마 정의
- [ ] Foreign Key 관계 정의
- [ ] 필수 인덱스 정의
- [ ] 쿼리 패턴 분석
- [ ] 마이그레이션 순서 정의
- [ ] 정규화 검증 (3NF)
- [ ] 성능 고려사항 검토

---

## 🔗 참고

- [CLAUDE.md](../../CLAUDE.md) - DB 설계 템플릿
- [modules/](../modules/) - 각 모듈의 데이터 요구사항
- [api/](../api/) - API 데이터 모델
- PostgreSQL Documentation
