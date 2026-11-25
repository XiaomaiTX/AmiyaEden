-- Run only in a disposable PostgreSQL database with -v disposable=true
-- and -v target_database=DB. The fixture schema is left for inspection.
\set ON_ERROR_STOP on
\if :{?disposable}
\else
\set disposable false
\endif
\if :disposable
\else
\echo 'Use a disposable database and pass -v disposable=true.'
SELECT 1/0;
\endif
SELECT current_database() = :'target_database' AS correct_database \gset
\if :correct_database
\else
SELECT 1/0;
\endif
\set target_schema cleanup_v2_fixture

CREATE SCHEMA cleanup_v2_fixture;
SET search_path TO cleanup_v2_fixture, pg_catalog;
CREATE TABLE announcement (id int);
CREATE TABLE captain_refund (id int);
CREATE TABLE fuxi_hall_page (page_key text PRIMARY KEY);
CREATE TABLE fuxi_hall_card (page_key text REFERENCES fuxi_hall_page, title text);
CREATE TABLE system_config (key text PRIMARY KEY, value text);
CREATE TABLE task_schedules (task_name text);
CREATE TABLE task_executions (task_name text);
CREATE TABLE eve_character_notification (id int);
CREATE TABLE qqbot_attempt (id int);
CREATE TABLE wallet_log (id int);
CREATE TABLE "user" (id int, qq text, avatar text);
INSERT INTO fuxi_hall_page VALUES ('hall-of-fame'), ('leadership'), ('contributors');
INSERT INTO fuxi_hall_card VALUES ('contributors', 'unused field');
INSERT INTO system_config VALUES
    ('salary.recruitment', '10'), ('salary.welfare_delivery', '20'),
    ('salary.shop_delivery', '20'), ('salary.srp_payout', '10'),
    ('pap.admin_award', '20'), ('pap.fc_salary', '100'), ('onebot.enabled', 'false'),
    ('webhook.type', 'discord'), ('webhook.url', 'https://example.invalid'),
    ('mumble.public_address', 'old-address');
INSERT INTO task_schedules VALUES ('qq_structure_alert'), ('captain_attribution_sync'), ('active_task');
INSERT INTO task_executions SELECT task_name FROM task_schedules;
INSERT INTO wallet_log VALUES (1);
INSERT INTO "user" VALUES (1, '12345', 'unused field');

\set apply false
\ir ../cleanup-pre-2.0.sql
DO $$
BEGIN
    IF to_regclass('qqbot_attempt') IS NULL OR to_regclass('eve_character_notification') IS NULL
        OR (SELECT count(*) FROM system_config) <> 10
        OR (SELECT count(*) FROM task_schedules) <> 3
        OR (SELECT count(*) FROM task_executions) <> 3
        OR (SELECT count(*) FROM fuxi_hall_page) <> 3 THEN
        RAISE EXCEPTION 'preview changed the fixture';
    END IF;
END $$;

\set apply true
\ir ../cleanup-pre-2.0.sql
\ir ../cleanup-pre-2.0.sql
DO $$
BEGIN
    IF to_regclass('qqbot_attempt') IS NOT NULL OR to_regclass('eve_character_notification') IS NOT NULL
        OR EXISTS (SELECT FROM information_schema.columns WHERE table_schema = 'cleanup_v2_fixture'
                   AND ((table_name = 'user' AND column_name = 'avatar')
                        OR (table_name = 'fuxi_hall_card' AND column_name = 'title')))
        OR EXISTS (SELECT FROM system_config WHERE key IN ('pap.admin_award', 'onebot.enabled'))
        OR EXISTS (SELECT FROM task_schedules WHERE task_name IN ('qq_structure_alert', 'captain_attribution_sync'))
        OR EXISTS (SELECT FROM task_executions WHERE task_name = 'captain_attribution_sync')
        OR EXISTS (SELECT FROM fuxi_hall_page WHERE page_key = 'leadership') THEN
        RAISE EXCEPTION 'legacy objects remain';
    END IF;
    IF NOT EXISTS (SELECT FROM fuxi_hall_page WHERE page_key = 'contributors')
        OR NOT EXISTS (SELECT FROM fuxi_hall_card WHERE page_key = 'contributors')
        OR NOT EXISTS (SELECT FROM system_config WHERE key = 'mumble.public_address' AND value = 'old-address')
        OR NOT EXISTS (SELECT FROM system_config WHERE key = 'pap.fc_salary' AND value = '100')
        OR NOT EXISTS (SELECT FROM system_config WHERE key = 'webhook.type' AND value = 'discord')
        OR NOT EXISTS (SELECT FROM system_config WHERE key = 'webhook.url' AND value = 'https://example.invalid')
        OR NOT EXISTS (SELECT FROM task_schedules WHERE task_name = 'active_task')
        OR NOT EXISTS (SELECT FROM task_executions WHERE task_name = 'qq_structure_alert')
        OR NOT EXISTS (SELECT FROM wallet_log WHERE id = 1)
        OR NOT EXISTS (SELECT FROM "user" WHERE id = 1 AND qq = '12345') THEN
        RAISE EXCEPTION 'protected records changed';
    END IF;
END $$;
