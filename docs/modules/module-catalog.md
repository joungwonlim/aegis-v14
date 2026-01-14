# 모듈 카탈로그 (Module Catalog)

> v14의 모든 모듈을 등록하고 상태를 추적합니다.

**Last Updated**: 2026-01-14

---

## 📋 개요

이 문서는 v14 시스템의 **모든 모듈의 SSOT(Single Source of Truth)**입니다.

### 목적
- 모듈별 독립 작업을 위한 명확한 경계 정의
- 개발 상태 및 준비도 추적
- 모듈 간 의존성 명시
- 개발 우선순위 관리

---

## 🏗️ 모듈 분류 체계

```
v14/
├── Infrastructure Layer (인프라 계층)
│   ├── External APIs      # 외부 API 연동
│   ├── Database           # 데이터 접근
│   └── Cache              # 캐싱 (Redis)
│
├── Core Runtime Layer (핵심 런타임 계층)
│   ├── PriceSync          # 현재가 동기화
│   ├── Exit Engine        # 자동 청산
│   ├── Reentry Engine     # 재진입
│   └── Execution          # 주문 실행
│
├── Strategy Layer (전략 계층)
│   ├── Universe           # 투자 가능 종목
│   ├── Signals            # 시그널 생성
│   ├── Ranking            # 종합 점수
│   └── Portfolio          # 포트폴리오 구성
│
├── Control Layer (제어 계층)
│   ├── Risk Management    # 리스크 관리
│   └── Monitoring         # 모니터링/알람
│
└── API Layer (API 계층)
    ├── BFF (Backend for Frontend)
    └── Admin API
```

---

## 📦 모듈 등록부

### Infrastructure Layer

#### 1. External APIs
| 속성 | 값 |
|------|-----|
| **ID** | `infra.external-apis` |
| **이름** | External APIs |
| **책임** | 외부 API 연동 (KIS WebSocket/REST, Naver Finance) |
| **위치** | `backend/internal/infra/external/` |
| **설계 문서** | `docs/modules/external-apis.md` |
| **상태** | ✅ 설계 완료 |
| **개발 준비도** | 🟢 Ready (인터페이스 정의 완료) |
| **의존성** | 없음 (최하위 레이어) |
| **제공 인터페이스** | `KISClient`, `NaverClient` |
| **소유권** | Infrastructure Team |

#### 2. Database
| 속성 | 값 |
|------|-----|
| **ID** | `infra.database` |
| **이름** | Database Access Layer |
| **책임** | PostgreSQL 데이터 접근, 트랜잭션 관리 |
| **위치** | `backend/internal/infra/database/` |
| **설계 문서** | `docs/database/schema.md`, `docs/database/access-control.md` |
| **상태** | ✅ 설계 완료 |
| **개발 준비도** | 🟢 Ready (스키마 정의 완료) |
| **의존성** | 없음 (최하위 레이어) |
| **제공 인터페이스** | Repository 인터페이스 (per domain) |
| **소유권** | Infrastructure Team |

#### 3. Cache (Redis)
| 속성 | 값 |
|------|-----|
| **ID** | `infra.cache` |
| **이름** | Cache Layer |
| **책임** | Redis 기반 캐싱 (읽기 가속, SSOT는 PostgreSQL) |
| **위치** | `backend/internal/infra/cache/` |
| **설계 문서** | `docs/architecture/architecture-improvements.md` (P1) |
| **상태** | ⬜ TODO |
| **개발 준비도** | 🟡 Pending (개선안 작성 완료, 상세 설계 필요) |
| **의존성** | `infra.database` (SSOT 읽기) |
| **제공 인터페이스** | `CacheService` |
| **소유권** | Infrastructure Team |

---

### Core Runtime Layer

#### 4. PriceSync
| 속성 | 값 |
|------|-----|
| **ID** | `runtime.price-sync` |
| **이름** | Price Synchronization |
| **책임** | 현재가 동기화 (KIS WS 실시간, REST fallback, Naver 보조) |
| **위치** | `backend/internal/runtime/pricesync/` |
| **설계 문서** | `docs/modules/price-sync.md` |
| **상태** | ✅ 설계 완료 |
| **개발 준비도** | 🟢 Ready |
| **의존성** | `infra.external-apis`, `infra.database` |
| **제공 인터페이스** | `PriceSyncService` |
| **소유권** | Runtime Team |
| **주의사항** | v10 사고 사례 - WS 장애 시 전체 시스템 멈춤. Fallback 필수. |

#### 5. Exit Engine
| 속성 | 값 |
|------|-----|
| **ID** | `runtime.exit-engine` |
| **이름** | Exit Engine |
| **책임** | 자동 청산 (Hybrid % + ATR, Control Gate, Profile System) |
| **위치** | `backend/internal/runtime/exit/` |
| **설계 문서** | `docs/modules/exit-engine.md` |
| **상태** | ✅ 설계 완료 |
| **개발 준비도** | 🟢 Ready |
| **의존성** | `runtime.price-sync`, `infra.database` |
| **제공 인터페이스** | `ExitEngineService` |
| **소유권** | Runtime Team |
| **주의사항** | v10 사고 사례 - 캐싱 버그로 청산 누락. 멱등성 필수. |

#### 6. Reentry Engine
| 속성 | 값 |
|------|-----|
| **ID** | `runtime.reentry-engine` |
| **이름** | Reentry Engine |
| **책임** | 재진입 전략 (ExitEvent 기반, Control Gate) |
| **위치** | `backend/internal/runtime/reentry/` |
| **설계 문서** | `docs/modules/reentry-engine.md` |
| **상태** | ✅ 설계 완료 |
| **개발 준비도** | 🟢 Ready |
| **의존성** | `runtime.exit-engine` (ExitEvent 구독), `infra.database` |
| **제공 인터페이스** | `ReentryEngineService` |
| **소유권** | Runtime Team |
| **주의사항** | Exit Engine과 디커플링됨. ExitEvent SSOT 기반 동작. |

#### 7. Execution
| 속성 | 값 |
|------|-----|
| **ID** | `runtime.execution` |
| **이름** | Execution Service |
| **책임** | 주문 제출/체결 관리 (ExitEvent 생성 SSOT) |
| **위치** | `backend/internal/runtime/execution/` |
| **설계 문서** | `docs/modules/execution-service.md` |
| **상태** | ✅ 설계 완료 |
| **개발 준비도** | 🟢 Ready |
| **의존성** | `infra.external-apis` (KIS 주문 API), `infra.database` |
| **제공 인터페이스** | `ExecutionService` |
| **소유권** | Runtime Team |
| **주의사항** | ExitEvent 생성의 SSOT. 중복 체결 방지 필수. |

---

### Strategy Layer

#### 8. Universe
| 속성 | 값 |
|------|-----|
| **ID** | `strategy.universe` |
| **이름** | Universe Selection |
| **책임** | 투자 가능 종목 선정 (유동성, 시가총액, 거래량 필터) |
| **위치** | `backend/internal/strategy/universe/` |
| **설계 문서** | `docs/modules/universe.md` |
| **상태** | ✅ 설계 완료 |
| **개발 준비도** | 🟢 Ready |
| **의존성** | `infra.database` |
| **제공 인터페이스** | `UniverseService` |
| **소유권** | Strategy Team |

#### 9. Signals
| 속성 | 값 |
|------|-----|
| **ID** | `strategy.signals` |
| **이름** | Signal Generation |
| **책임** | 팩터/이벤트 시그널 생성 (모멘텀, 가치, 이벤트 등) |
| **위치** | `backend/internal/strategy/signals/` |
| **설계 문서** | `docs/modules/signals.md` |
| **상태** | ✅ 설계 완료 |
| **개발 준비도** | 🟢 Ready (팩터 기반 평가 설계 완료) |
| **의존성** | `strategy.universe`, `infra.database` |
| **제공 인터페이스** | `SignalService` |
| **소유권** | Strategy Team |

#### 10. Ranking
| 속성 | 값 |
|------|-----|
| **ID** | `strategy.ranking` |
| **이름** | Ranking Engine |
| **책임** | 종합 점수 산출 (시그널 가중치 합산) |
| **위치** | `backend/internal/strategy/ranking/` |
| **설계 문서** | `docs/modules/ranking.md` |
| **상태** | ⬜ TODO |
| **개발 준비도** | 🔴 Blocked (설계 문서 미작성) |
| **의존성** | `strategy.signals` |
| **제공 인터페이스** | `RankingService` |
| **소유권** | Strategy Team |

#### 11. Portfolio
| 속성 | 값 |
|------|-----|
| **ID** | `strategy.portfolio` |
| **이름** | Portfolio Construction |
| **책임** | 포트폴리오 구성 (종목 선택, 비중 할당) |
| **위치** | `backend/internal/strategy/portfolio/` |
| **설계 문서** | `docs/modules/portfolio.md` |
| **상태** | ⬜ TODO |
| **개발 준비도** | 🔴 Blocked (설계 문서 미작성) |
| **의존성** | `strategy.ranking` |
| **제공 인터페이스** | `PortfolioService` |
| **소유권** | Strategy Team |

---

### Control Layer

#### 12. Risk Management
| 속성 | 값 |
|------|-----|
| **ID** | `control.risk` |
| **이름** | Risk Management |
| **책임** | 리스크 관리 (포지션 한도, 손실 한도, 집중도 관리) |
| **위치** | `backend/internal/control/risk/` |
| **설계 문서** | `docs/modules/risk-management.md` |
| **상태** | ⬜ TODO |
| **개발 준비도** | 🔴 Blocked (설계 문서 미작성) |
| **의존성** | `strategy.portfolio`, `runtime.execution` |
| **제공 인터페이스** | `RiskService` |
| **소유권** | Control Team |

#### 13. Monitoring
| 속성 | 값 |
|------|-----|
| **ID** | `control.monitoring` |
| **이름** | Monitoring & Alerting |
| **책임** | 시스템 모니터링, 알람, 로깅 |
| **위치** | `backend/internal/control/monitoring/` |
| **설계 문서** | `docs/modules/monitoring.md` |
| **상태** | ⬜ TODO |
| **개발 준비도** | 🔴 Blocked (설계 문서 미작성) |
| **의존성** | 모든 모듈 (횡단 관심사) |
| **제공 인터페이스** | `MonitoringService` |
| **소유권** | Control Team |

---

### API Layer

#### 14. BFF (Backend for Frontend)
| 속성 | 값 |
|------|-----|
| **ID** | `api.bff` |
| **이름** | Backend for Frontend |
| **책임** | 프론트엔드용 API 제공 (REST/GraphQL) |
| **위치** | `backend/internal/api/` |
| **설계 문서** | `docs/api/*.md` |
| **상태** | ⬜ TODO |
| **개발 준비도** | 🔴 Blocked (API 설계 미작성) |
| **의존성** | 모든 서비스 레이어 |
| **제공 인터페이스** | HTTP REST API |
| **소유권** | API Team |

---

## 📊 모듈 개발 상태 대시보드

### 레이어별 진행률

| 레이어 | 완료 | 진행 중 | TODO | 진행률 |
|--------|------|---------|------|--------|
| Infrastructure | 2/3 | 0/3 | 1/3 | 67% |
| Core Runtime | 4/4 | 0/4 | 0/4 | 100% ✅ |
| Strategy | 2/4 | 0/4 | 2/4 | 50% |
| Control | 0/2 | 0/2 | 2/2 | 0% |
| API | 0/1 | 0/1 | 1/1 | 0% |
| **Total** | **8/14** | **0/14** | **6/14** | **57%** |

### 개발 준비도별 현황

| 준비도 | 개수 | 모듈 |
|--------|------|------|
| 🟢 Ready | 8 | external-apis, database, price-sync, exit-engine, reentry-engine, execution, universe, signals |
| 🟡 Pending | 1 | cache |
| 🔴 Blocked | 5 | ranking, portfolio, risk, monitoring, bff |

---

## 🎯 개발 우선순위

### Phase 1: Infrastructure 완성 (P0)
```
✅ external-apis (완료)
✅ database (완료)
⬜ cache (설계 필요)
```

### Phase 2: Core Runtime 운영 (P0)
```
✅ price-sync (완료)
✅ exit-engine (완료)
✅ reentry-engine (완료)
✅ execution (완료)
```

### Phase 3: API Layer (P1)
```
⬜ BFF 설계 및 구현
```

### Phase 4: Strategy Layer (P2)
```
✅ universe (설계 완료)
✅ signals (설계 완료)
⬜ ranking
⬜ portfolio
```

### Phase 5: Control Layer (P2)
```
⬜ risk
⬜ monitoring
```

---

## 🔧 모듈 독립 개발 규칙

### 1. 인터페이스 우선 설계
```go
// ✅ CORRECT - 인터페이스 먼저 정의
type PriceSyncService interface {
    GetCurrentPrice(ctx context.Context, symbol string) (Price, error)
    Subscribe(ctx context.Context, symbols []string) error
}

// Exit Engine은 PriceSyncService 인터페이스에만 의존
type ExitEngine struct {
    priceSync PriceSyncService  // 인터페이스에 의존
}
```

### 2. Mock/Stub 제공
각 모듈은 테스트용 Mock 구현을 제공해야 합니다.

```go
// price-sync/mock/mock.go
type MockPriceSyncService struct {
    GetCurrentPriceFunc func(ctx context.Context, symbol string) (Price, error)
}

func (m *MockPriceSyncService) GetCurrentPrice(ctx context.Context, symbol string) (Price, error) {
    if m.GetCurrentPriceFunc != nil {
        return m.GetCurrentPriceFunc(ctx, symbol)
    }
    return Price{}, nil
}
```

### 3. 의존성 주입 (Dependency Injection)
```go
// ✅ CORRECT - 생성자에서 의존성 주입
func NewExitEngine(
    priceSync PriceSyncService,  // 인터페이스
    repo Repository,              // 인터페이스
) *ExitEngine {
    return &ExitEngine{
        priceSync: priceSync,
        repo: repo,
    }
}
```

### 4. 순환 참조 금지
```
❌ 금지:
ExitEngine → ReentryEngine → ExitEngine

✅ 허용:
ExitEngine → ExitEvent (이벤트 발행)
ReentryEngine → ExitEvent (이벤트 구독)
```

---

## 📝 모듈 추가 프로세스

1. **이 문서에 모듈 등록**
   - 모듈 ID, 이름, 책임, 의존성 등 명시

2. **설계 문서 작성**
   - `docs/modules/{module-name}.md` 생성
   - 인터페이스 정의 필수

3. **`docs/_index.md` 업데이트**
   - 문서 등록부에 추가

4. **의존성 검증**
   - `docs/architecture/module-dependencies.md` 업데이트
   - 순환 참조 확인

---

## ⚠️ 주의사항

### 모듈 독립성 체크리스트

각 모듈은 다음 조건을 만족해야 합니다:

- [ ] 명확한 인터페이스 정의
- [ ] 다른 모듈의 구현체가 아닌 인터페이스에 의존
- [ ] Mock/Stub 구현 제공
- [ ] 단독 빌드/테스트 가능
- [ ] 순환 참조 없음
- [ ] 단일 책임 원칙 준수

---

## 🔍 참고 문서

- [모듈 의존성 맵](../architecture/module-dependencies.md) (TODO)
- [모듈 개발 가이드](./development-guide.md) (TODO)
- [인터페이스 계약서](각 모듈 문서 참고)

---

**Version**: 1.0.0
**Last Updated**: 2026-01-14
