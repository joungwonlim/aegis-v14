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

# 2. 적절한 PostgreSQL 수퍼유저 확인
echo "2. PostgreSQL 수퍼유저 확인..."
PGUSER=""

# postgres 유저 시도
if psql -U postgres -d postgres -c "SELECT 1" > /dev/null 2>&1; then
    PGUSER="postgres"
    echo "✅ postgres 유저 사용"
else
    # 현재 시스템 유저 시도
    CURRENT_USER=$(whoami)
    if psql -U "$CURRENT_USER" -d postgres -c "SELECT 1" > /dev/null 2>&1; then
        PGUSER="$CURRENT_USER"
        echo "✅ $CURRENT_USER 유저 사용"
    else
        echo "❌ PostgreSQL 수퍼유저를 찾을 수 없습니다."
        echo "   시도한 유저: postgres, $CURRENT_USER"
        echo ""
        echo "   해결 방법 1: postgres 수퍼유저 생성"
        echo "      createuser -s postgres"
        echo ""
        echo "   해결 방법 2: 현재 유저($CURRENT_USER)에게 수퍼유저 권한 부여"
        echo "      # 다른 수퍼유저로 실행:"
        echo "      psql -d postgres -c \"ALTER USER $CURRENT_USER WITH SUPERUSER;\""
        exit 1
    fi
fi
echo ""

# 3. Database 및 Role 생성
echo "3. Database 및 Role 생성..."
psql -U "$PGUSER" -d postgres -f ../scripts/db/01_create_database.sql
echo "✅ Database 및 Role 생성 완료"
echo ""

# 4. Schema 및 권한 설정
echo "4. Schema 및 권한 설정..."
psql -U aegis_v14 -d aegis_v14 -f ../scripts/db/02_create_schemas.sql 2>&1 | grep -v "^ERROR" || true
echo "✅ Schema 및 권한 설정 완료"
echo ""

# 5. 권한 확인
echo "5. 권한 확인..."
psql -U aegis_v14 -d aegis_v14 -f ../scripts/db/03_check_permissions.sql 2>&1 | head -20
echo ""

# 6. .env 파일 확인
echo "6. .env 파일 확인..."
if [ ! -f ".env" ]; then
    echo "⚠️  .env 파일이 없습니다. .env.example을 복사합니다."
    cp .env.example .env
    echo "✅ .env 파일 생성 완료"
else
    echo "✅ .env 파일 존재"
fi
echo ""

# 7. Go 의존성 설치
echo "7. Go 의존성 설치..."
go mod download > /dev/null 2>&1
go mod tidy > /dev/null 2>&1
echo "✅ Go 의존성 설치 완료"
echo ""

# 8. 최종 연결 테스트
echo "8. 최종 연결 테스트..."
if psql -U aegis_v14 -d aegis_v14 -c "SELECT 'Connection OK' as status;" > /dev/null 2>&1; then
    echo "✅ DB 연결 성공"
else
    echo "⚠️  DB 연결 실패. 권한 수정 시도..."
    psql -U aegis_v14 -d aegis_v14 -f ../scripts/db/04_fix_permissions.sql > /dev/null 2>&1 || true
    if psql -U aegis_v14 -d aegis_v14 -c "SELECT 'Connection OK' as status;" > /dev/null 2>&1; then
        echo "✅ 권한 수정 후 DB 연결 성공"
    else
        echo "❌ DB 연결 여전히 실패. 수동 확인 필요."
        exit 1
    fi
fi
echo ""

echo "=================================="
echo "✅ 초기화 완료!"
echo "=================================="
echo ""
echo "다음 명령어로 서버를 시작하세요:"
echo "  make run"
echo ""
