#!/usr/bin/env bash
set -Eeuo pipefail

create_database_and_role() {
  local database_name="$1"
  local role_name="$2"
  local role_password="$3"

  psql \
    --username "$POSTGRES_USER" \
    --dbname "$POSTGRES_DB" \
    --set=ON_ERROR_STOP=1 \
    --set=database_name="$database_name" \
    --set=role_name="$role_name" \
    --set=role_password="$role_password" <<'EOSQL'
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'role_name', :'role_password')
WHERE NOT EXISTS (
  SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = :'role_name'
)
\gexec

SELECT format('CREATE DATABASE %I OWNER %I', :'database_name', :'role_name')
WHERE NOT EXISTS (
  SELECT 1 FROM pg_catalog.pg_database WHERE datname = :'database_name'
)
\gexec
EOSQL
}

create_database_and_role "$DXOS_DB" "$DXOS_DB_USER" "$DXOS_DB_PASSWORD"
create_database_and_role "$KEYCLOAK_DB" "$KEYCLOAK_DB_USER" "$KEYCLOAK_DB_PASSWORD"
create_database_and_role "$NEXTCLOUD_DB" "$NEXTCLOUD_DB_USER" "$NEXTCLOUD_DB_PASSWORD"
create_database_and_role "$METABASE_DB" "$METABASE_DB_USER" "$METABASE_DB_PASSWORD"
