-- =====================================================
-- v14 권한 확인 스크립트
-- =====================================================
-- 실행: psql -U aegis_v14 -d aegis_v14 -f scripts/db/03_check_permissions.sql
-- =====================================================

\echo ''
\echo '========================================='
\echo 'Database Permission Check'
\echo '========================================='

-- 1. Schema 권한 확인
\echo ''
\echo '1. Schema Permissions:'
\echo '-------------------------------------'

SELECT
    nsp.nspname AS schema_name,
    rol.rolname AS owner,
    pg_catalog.has_schema_privilege('aegis_v14', nsp.nspname, 'CREATE') AS can_create,
    pg_catalog.has_schema_privilege('aegis_v14', nsp.nspname, 'USAGE') AS can_usage
FROM pg_namespace nsp
JOIN pg_roles rol ON nsp.nspowner = rol.oid
WHERE nsp.nspname IN ('market', 'trade', 'system')
ORDER BY nsp.nspname;

-- 2. 테이블 권한 확인 (테이블이 있는 경우)
\echo ''
\echo '2. Table Permissions (if tables exist):'
\echo '-------------------------------------'

SELECT
    schemaname,
    tablename,
    tableowner,
    pg_catalog.has_table_privilege('aegis_v14', schemaname || '.' || tablename, 'SELECT') AS can_select,
    pg_catalog.has_table_privilege('aegis_v14', schemaname || '.' || tablename, 'INSERT') AS can_insert,
    pg_catalog.has_table_privilege('aegis_v14', schemaname || '.' || tablename, 'UPDATE') AS can_update,
    pg_catalog.has_table_privilege('aegis_v14', schemaname || '.' || tablename, 'DELETE') AS can_delete
FROM pg_tables
WHERE schemaname IN ('market', 'trade', 'system')
ORDER BY schemaname, tablename;

-- 3. Default Privileges 확인
\echo ''
\echo '3. Default Privileges:'
\echo '-------------------------------------'

SELECT
    pg_get_userbyid(defaclrole) AS grantor,
    nspname AS schema,
    CASE defaclobjtype
        WHEN 'r' THEN 'table'
        WHEN 'S' THEN 'sequence'
        WHEN 'f' THEN 'function'
        WHEN 'T' THEN 'type'
    END AS object_type,
    defaclacl AS privileges
FROM pg_default_acl a
JOIN pg_namespace n ON a.defaclnamespace = n.oid
WHERE nspname IN ('market', 'trade', 'system')
ORDER BY nspname, object_type;

-- 4. Role 정보
\echo ''
\echo '4. Roles:'
\echo '-------------------------------------'

SELECT
    rolname,
    rolsuper AS is_superuser,
    rolinherit AS can_inherit,
    rolcreaterole AS can_create_role,
    rolcreatedb AS can_create_db,
    rolcanlogin AS can_login
FROM pg_roles
WHERE rolname LIKE 'aegis_v14%'
ORDER BY rolname;

\echo ''
\echo '========================================='
\echo 'Check complete!'
\echo '========================================='
\echo ''
