#!/bin/bash
# =============================================================================
# Aegis v14 - 로그 실시간 모니터링 스크립트
# =============================================================================

LOG_DIR="./logs"

# 색상 정의
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "🔍 Aegis v14 Runtime 로그 모니터링"
echo "================================="
echo ""
echo "필터링 옵션:"
echo "  1. 전체 로그 (info 레벨)"
echo "  2. 중요 이벤트만 (WRN, ERR, PriceSync, Exit)"
echo "  3. 에러만 (ERR, panic)"
echo "  4. PriceSync 관련"
echo "  5. Exit Engine 관련"
echo ""
read -p "선택 (1-5): " choice

case $choice in
  1)
    echo "📋 전체 로그 모니터링 중..."
    tail -f "$LOG_DIR/app.log"
    ;;
  2)
    echo "⚠️  중요 이벤트만 모니터링 중..."
    tail -f "$LOG_DIR/app.log" | grep -E "(WRN|ERR|PriceSync|Exit|subscriptions|Priorities)"
    ;;
  3)
    echo "🚨 에러만 모니터링 중..."
    tail -f "$LOG_DIR/app.log" | grep -E "(ERR|panic|fatal)"
    ;;
  4)
    echo "💰 PriceSync 모니터링 중..."
    tail -f "$LOG_DIR/app.log" | grep -E "(PriceSync|Priorities|subscriptions|WS|REST tier|Naver)"
    ;;
  5)
    echo "🚪 Exit Engine 모니터링 중..."
    tail -f "$LOG_DIR/app.log" | grep -E "(Exit|Position|Intent|Order|Trigger|HardStop)"
    ;;
  *)
    echo "잘못된 선택"
    exit 1
    ;;
esac
