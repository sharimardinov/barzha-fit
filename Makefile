SHELL := /bin/sh

-include .env

DB_TABLES ?= meals workout_days steps_days plans user_profiles user_targets bot_users
DB_DAYS ?= 60

.PHONY: db-clear db-clear-all db-seed

db-clear:
	@if [ -z "$(DATABASE_URL)" ]; then echo "DATABASE_URL is not set"; exit 1; fi
	@tables="$(if $(TABLES),$(TABLES),$(DB_TABLES))"; \
	tables=$$(echo $$tables | tr ' ' ','); \
	list=$$(psql "$(DATABASE_URL)" -t -A -v ON_ERROR_STOP=1 \
		-c "select string_agg(quote_ident(tablename), ',') from pg_tables where schemaname='public' and tablename = any(string_to_array('$$tables', ','));"); \
	if [ -z "$$list" ]; then echo "No matching tables to truncate"; exit 0; fi; \
	psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -c "truncate $$list restart identity;"

db-clear-all: db-clear

db-seed:
	@if [ -z "$(DATABASE_URL)" ]; then echo "DATABASE_URL is not set"; exit 1; fi
	@if [ -z "$(CHAT_ID)" ]; then echo "CHAT_ID is required (e.g. make db-seed CHAT_ID=123)"; exit 1; fi
	psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -v chat_id=$(CHAT_ID) -v days=$(DB_DAYS) -f scripts/seed.sql
