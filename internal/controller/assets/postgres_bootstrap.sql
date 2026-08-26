-- 1) LIGHTSPEED STACK DATABASE CONFIGURATION ---------------------------------
\c :postgresql_lightspeed_stack_database

-- PostgreSQL 15+ removed the default CREATE/USAGE grants on the public schema.
-- Therefore we have to explicitly grant the permissions to the user.
GRANT USAGE, CREATE ON SCHEMA public TO :"postgresql_user";

-- lightspeed-stack requires rights to create additional schemas under its database
GRANT CREATE ON DATABASE :"postgresql_lightspeed_stack_database" TO :"postgresql_user";

-- pg_trgm is the trigram similarity extension for PostgreSQL (enables e.g. fuzzy text search)
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Unlike its cache/database backends, lightspeed-stack's quota module never
-- issues CREATE SCHEMA before creating its tables under the configured
-- namespace, so the "quota_handlers" schema must already exist.
CREATE SCHEMA IF NOT EXISTS quota_handlers AUTHORIZATION :"postgresql_user";
-------------------------------------------------------------------------------

-- 2) LLAMA STACK DATABASE CONFIGURATION --------------------------------------
-- Create postgresql_llama_stack_database.
SELECT format('CREATE DATABASE %I', :'postgresql_llama_stack_database')
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = :'postgresql_llama_stack_database')\gexec

-- Connect and configure postgresql_llama_stack_database
\c :postgresql_llama_stack_database

-- PostgreSQL 15+ removed the default CREATE/USAGE grants on the public schema.
-- Therefore we have to explicitly grant the permissions to the user.
GRANT USAGE, CREATE ON SCHEMA public TO :"postgresql_user";

-- pg_trgm is the trigram similarity extension for PostgreSQL (enables e.g. fuzzy text search)
CREATE EXTENSION IF NOT EXISTS pg_trgm;
-------------------------------------------------------------------------------
