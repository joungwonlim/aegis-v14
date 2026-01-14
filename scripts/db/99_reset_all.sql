-- =====================================================
-- v14 완전 초기화 스크립트
-- =====================================================
-- ⚠️ 경고: 이 스크립트는 aegis_v14 데이터베이스를 완전히 삭제합니다!
-- ⚠️ 모든 데이터가 손실됩니다!
--
-- 실행: psql -U postgres -f scripts/db/99_reset_all.sql
-- =====================================================

\echo ''
\echo '========================================='
\echo '⚠️  WARNING: DATABASE RESET'
\echo '========================================='
\echo ''
\echo 'This will DELETE aegis_v14 database!'
\echo 'All data will be LOST!'
\echo ''
\echo 'Press Ctrl+C to cancel, or Enter to continue...'
\echo ''

\prompt 'Type "RESET" to confirm: ' confirm

\if :{?confirm}
    \if :confirm = 'RESET'
        \echo ''
        \echo 'Proceeding with reset...'
        \echo ''

        -- 1. 모든 연결 종료
        SELECT pg_terminate_backend(pid)
        FROM pg_stat_activity
        WHERE datname = 'aegis_v14' AND pid <> pg_backend_pid();

        -- 2. Database 삭제 및 재생성
        DROP DATABASE IF EXISTS aegis_v14;
        CREATE DATABASE aegis_v14
            WITH
            ENCODING = 'UTF8'
            LC_COLLATE = 'en_US.UTF-8'
            LC_CTYPE = 'en_US.UTF-8'
            TEMPLATE = template0
            OWNER = aegis_v14;

        \echo ''
        \echo '========================================='
        \echo 'Database reset complete!'
        \echo '========================================='
        \echo ''
        \echo 'Next steps:'
        \echo '  1. Run 02_create_schemas.sql'
        \echo '     psql -U aegis_v14 -d aegis_v14 -f scripts/db/02_create_schemas.sql'
        \echo ''
        \echo '  2. Run migrations'
        \echo '     migrate -path backend/migrations -database $DATABASE_URL up'
        \echo ''
    \else
        \echo ''
        \echo 'Reset cancelled (invalid confirmation).'
        \echo ''
    \endif
\else
    \echo ''
    \echo 'Reset cancelled (no confirmation).'
    \echo ''
\endif
