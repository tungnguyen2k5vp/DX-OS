#!/bin/sh
set -eu

psql \
  --host=postgres \
  --username="$POSTGRES_ADMIN_USER" \
  --dbname="$DXOS_DB" \
  --set=reporting_user="$REPORTING_DB_USER" \
  --set=reporting_password="$REPORTING_DB_PASSWORD" \
  --set=database_name="$DXOS_DB" \
  --set=application_owner="$DXOS_DB_USER" \
  --file=/reporting/bootstrap-reporting-role.sql
