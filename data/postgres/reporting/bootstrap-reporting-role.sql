\set ON_ERROR_STOP on

SELECT format(
    'CREATE ROLE %I LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT',
    :'reporting_user',
    :'reporting_password'
)
WHERE NOT EXISTS (
    SELECT 1 FROM pg_roles WHERE rolname = :'reporting_user'
)
\gexec

SELECT format(
    'ALTER ROLE %I WITH LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT',
    :'reporting_user',
    :'reporting_password'
)
\gexec

SELECT format('ALTER ROLE %I SET default_transaction_read_only = on', :'reporting_user')
\gexec

SELECT format('REVOKE ALL ON DATABASE %I FROM %I', :'database_name', :'reporting_user')
\gexec

SELECT format('GRANT CONNECT ON DATABASE %I TO %I', :'database_name', :'reporting_user')
\gexec

REVOKE ALL ON SCHEMA public FROM PUBLIC;

SELECT format('REVOKE ALL ON SCHEMA public FROM %I', :'reporting_user')
\gexec

SELECT format('GRANT USAGE ON SCHEMA reporting TO %I', :'reporting_user')
\gexec

SELECT format('GRANT SELECT ON ALL TABLES IN SCHEMA reporting TO %I', :'reporting_user')
\gexec

SELECT format(
    'ALTER DEFAULT PRIVILEGES FOR ROLE %I IN SCHEMA reporting GRANT SELECT ON TABLES TO %I',
    :'application_owner',
    :'reporting_user'
)
\gexec
