# 로그 관리 가이드

## 📋 현재 설정

```env
LOG_LEVEL=info              # info, warn, error만 기록 (debug 제외)
LOG_FORMAT=pretty           # 컬러 포맷팅
LOG_FILE_ENABLED=true       # 파일 저장 활성화
LOG_FILE_PATH=./logs        # 로그 디렉토리
LOG_ROTATION_SIZE=100       # 100MB 단위 자동 로테이션
LOG_RETENTION_DAYS=30       # 30일 보관
```

## 🚀 사용법

### 1. Runtime 실행 (로그 파일 저장)

```bash
# 백그라운드 실행 (권장)
nohup ./runtime > /dev/null 2>&1 &

# 또는 tmux/screen 사용
tmux new -s aegis
./runtime
# Ctrl+B, D로 detach
```

### 2. 로그 실시간 모니터링

```bash
# 대화형 모니터링
./scripts/monitor-logs.sh

# 직접 tail 사용
tail -f logs/app.log                                    # 전체
tail -f logs/app.log | grep -E "ERR|WRN"               # 에러/경고만
tail -f logs/app.log | grep "PriceSync"                # PriceSync만
tail -f logs/app.log | grep "Exit"                     # Exit Engine만
```

### 3. 로그 검색

```bash
# 특정 심볼 검색
grep "000660" logs/app.log

# 에러 검색 (최근 100줄)
grep "ERR" logs/app.log | tail -100

# 시간대별 검색
grep "06:40:" logs/app.log

# PriorityManager 동작 확인
grep "Priorities refreshed" logs/app.log
```

### 4. 로그 정리

```bash
# 대화형 정리
./scripts/clean-logs.sh

# 수동 정리
find logs/ -name "*.log" -mtime +7 -delete      # 7일 이전 삭제
gzip logs/*.log                                  # 압축
```

## 📊 로그 레벨 변경

### 운영 환경 (권장)
```env
LOG_LEVEL=info    # INF, WRN, ERR만
```

### 개발/디버깅
```env
LOG_LEVEL=debug   # DBG 포함 (매우 자세함, 디스크 소모 큼)
```

### 프로덕션
```env
LOG_LEVEL=warn    # WRN, ERR만 (최소)
```

## 🔍 중요 로그 패턴

### PriceSync 정상 작동 확인
```
INF PriorityManager configured
INF Priorities refreshed holdings=X closing=X orders=X
INF WS subscriptions updated ws_total=X subscribed=X
INF REST tiers updated tier0=X tier1=X tier2=X
INF ✅ PriceSync subscriptions initialized
```

### 가격 업데이트 확인
```
DBG Processed WS tick symbol=005930 price=65000
INF Tier prices processed tier=0 total=40 success=40
```

### Exit Engine 동작 확인
```
DBG Evaluating positions count=5
INF Exit trigger fired symbol=005930 trigger=TP1
INF Intent created intent_id=xxx position_id=yyy
```

### 에러 확인
```
ERR Position evaluation failed error="price is stale"
ERR KIS price fetch failed, trying Naver fallback
WRN Price too old age_seconds=4800
```

## 💡 팁

1. **로그 레벨은 info 유지** (debug는 디스크 소모 큼)
2. **tmux/screen으로 백그라운드 실행**
3. **monitor-logs.sh로 필터링된 로그만 확인**
4. **주 1회 clean-logs.sh 실행**
5. **중요 이벤트는 Telegram 알림 활용** (별도 설정 필요)

## 🚨 문제 해결

### 로그 파일이 너무 클 때
```bash
# 즉시 압축
gzip logs/app.log

# 로그 레벨 올리기
# .env: LOG_LEVEL=warn
```

### 디스크 공간 부족
```bash
# 긴급 정리
./scripts/clean-logs.sh
# 옵션 2 선택 (오늘 제외 전체 삭제)
```

### 특정 에러 추적
```bash
# 에러 발생 시간대 확인
grep "ERR.*000660" logs/app.log | tail -20

# 해당 시간대 전후 로그 확인
grep "06:40:" logs/app.log | grep -A 5 -B 5 "000660"
```
