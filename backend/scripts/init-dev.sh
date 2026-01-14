#!/bin/bash
# Aegis v14 - 개발 환경 자동 초기화 스크립트
# 이 스크립트는 모든 권한 문제를 자동으로 해결합니다.

set -e  # 에러 발생 시 즉시 중단

echo "=================================="
echo "Aegis v14 개발 환경 초기화"
echo "=================================="
echo ""

# 1. PostgreSQL 실행 확인
echo "1. PostgreSQL 실행 확인..."
if ! pg_isready -h localhost -p 5432 > /dev/null 2>&1; then
    echo "❌ PostgreSQL이 실행되지 않았습니다."
    echo "   실행 방법: brew services start postgresql"
    exit 1
fi
echo "✅ PostgreSQL 실행 중"
echo ""

# 2. Database 및 Role 생성
echo "2. Database 및 Role 생성..."
psql -U postgres -c "SELECT 1" > /dev/null 2>&1
if [ $? -ne 0 ]; then
    echo "❌ postgres 유저로 접속할 수 없습니다."
    echo "   해결: createuser -s postgres (superuser 생성)"
    exit 1
fi

psql -U postgres -f ../scripts/db/01_create_database.sql
echo "✅ Database 및 Role 생성 완료"
echo ""

# 3. Schema 및 권한 설정
echo "3. Schema 및 권한 설정..."
psql -U aegis_v14 -d aegis_v14 -f ../scripts/db/02_create_schemas.sql
echo "✅ Schema 및 권한 설정 완료"
echo ""

# 4. 권한 확인
echo "4. 권한 확인..."
psql -U aegis_v14 -d aegis_v14 -f ../scripts/db/03_check_permissions.sql
echo ""

# 5. .env 파일 확인
echo "5. .env 파일 확인..."
if [ ! -f ".env" ]; then
    echo "⚠️  .env 파일이 없습니다. .env.example을 복사합니다."
    cp .env.example .env
    echo "✅ .env 파일 생성 완료"
else
    echo "✅ .env 파일 존재"
fi
echo ""

# 6. Go 의존성 설치
echo "6. Go 의존성 설치..."
go mod download
go mod tidy
echo "✅ Go 의존성 설치 완료"
echo ""

# 7. 최종 연결 테스트
echo "7. 최종 연결 테스트..."
psql -U aegis_v14 -d aegis_v14 -c "SELECT 'Connection OK' as status;" || {
    echo "❌ DB 연결 실패"
    echo "   권한 수정 시도..."
    psql -U aegis_v14 -d aegis_v14 -f ../scripts/db/04_fix_permissions.sql
    echo "✅ 권한 수정 완료. 다시 테스트..."
    psql -U aegis_v14 -d aegis_v14 -c "SELECT 'Connection OK' as status;"
}
echo "✅ DB 연결 성공"
echo ""

echo "=================================="
echo "✅ 초기화 완료!"
echo "=================================="
echo ""
echo "다음 명령어로 서버를 시작하세요:"
echo "  make run"
echo ""
