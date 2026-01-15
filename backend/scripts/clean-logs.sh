#!/bin/bash
# =============================================================================
# Aegis v14 - 로그 정리 스크립트
# =============================================================================

LOG_DIR="./logs"

echo "🧹 Aegis v14 로그 정리"
echo "====================="
echo ""
echo "현재 로그 파일 크기:"
ls -lh "$LOG_DIR" | grep -v "^total" | awk '{print $9, $5}'
echo ""
du -sh "$LOG_DIR"
echo ""
echo "정리 옵션:"
echo "  1. 7일 이전 로그 삭제"
echo "  2. 오늘 로그 제외 전체 삭제"
echo "  3. 100MB 이상 로그 압축"
echo "  4. 전체 로그 삭제 (주의!)"
echo "  5. 취소"
echo ""
read -p "선택 (1-5): " choice

case $choice in
  1)
    echo "📅 7일 이전 로그 삭제 중..."
    find "$LOG_DIR" -name "*.log*" -type f -mtime +7 -delete
    echo "✅ 완료"
    ;;
  2)
    echo "🗑️  오늘 제외 로그 삭제 중..."
    find "$LOG_DIR" -name "*.log*" -type f ! -mtime 0 -delete
    echo "✅ 완료"
    ;;
  3)
    echo "📦 100MB 이상 로그 압축 중..."
    find "$LOG_DIR" -name "*.log" -type f -size +100M -exec gzip {} \;
    echo "✅ 완료"
    ;;
  4)
    read -p "⚠️  정말로 전체 로그를 삭제하시겠습니까? (yes/no): " confirm
    if [ "$confirm" = "yes" ]; then
      rm -f "$LOG_DIR"/*.log*
      echo "✅ 전체 삭제 완료"
    else
      echo "❌ 취소됨"
    fi
    ;;
  5)
    echo "❌ 취소됨"
    exit 0
    ;;
  *)
    echo "잘못된 선택"
    exit 1
    ;;
esac

echo ""
echo "정리 후 로그 파일:"
ls -lh "$LOG_DIR" | grep -v "^total" | awk '{print $9, $5}'
du -sh "$LOG_DIR"
