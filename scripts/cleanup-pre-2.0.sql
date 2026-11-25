-- Manual post-upgrade cleanup. Run with psql -X -d DB
-- -v target_database=DB -v target_schema=SCHEMA -f scripts/cleanup-pre-2.0.sql
-- Preview by default; add -v apply=true only after backups and review.
\set cleanup_pre_2_0 true
\ir qq-bot-retirement.sql
\unset cleanup_pre_2_0
