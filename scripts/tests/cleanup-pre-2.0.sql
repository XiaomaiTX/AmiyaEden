-- Run only in a disposable PostgreSQL database with -v disposable=true
-- and -v target_database=DB. The fixture schemas are left for inspection.
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
SELECT current_database()::text = :'target_database' AS correct_database \gset
\if :correct_database
\else
SELECT 1/0;
\endif
\set target_schema cleanup_v2_fixture

CREATE SCHEMA cleanup_v2_fixture;
CREATE SCHEMA cleanup_v2_other;
CREATE TABLE cleanup_v2_other.eve_character_notification (id int);
CREATE TABLE cleanup_v2_other.qqbot_attempt (id int);
INSERT INTO cleanup_v2_other.eve_character_notification VALUES (2);
INSERT INTO cleanup_v2_other.qqbot_attempt VALUES (2);
SET search_path TO cleanup_v2_fixture, pg_catalog;
CREATE TABLE announcement (id int);
CREATE TABLE captain_refund (id int);
CREATE TABLE fuxi_hall_page (page_key text PRIMARY KEY);
CREATE TABLE fuxi_hall_card (page_key text REFERENCES fuxi_hall_page, title text, deleted_at timestamptz);
CREATE TABLE system_config (key text PRIMARY KEY, value text);
CREATE TABLE task_schedules (task_name text);
CREATE TABLE task_execution (task_name text);
CREATE TABLE eve_character_notification (id int);
CREATE TABLE qqbot_attempt (id int);
CREATE TABLE qq_official_attempt (id int);
CREATE TABLE shop_lottery_activity (id int PRIMARY KEY);
CREATE TABLE shop_lottery_prize (activity_id int REFERENCES shop_lottery_activity);
CREATE TABLE wallet_log (id int);
CREATE TABLE "user" (id int, name text, qq text, avatar text);
INSERT INTO fuxi_hall_page VALUES ('hall-of-fame'), ('leadership'), ('contributors');
INSERT INTO fuxi_hall_card VALUES ('contributors', 'unused field', '2026-01-01 00:00:00+00');
INSERT INTO system_config VALUES
    ('salary.recruitment', '10'), ('salary.welfare_delivery', '20'),
    ('salary.shop_delivery', '20'), ('salary.srp_payout', '10'),
    ('pap.admin_award', '20'), ('newbro.bonus_rate', '50'),
    ('mentor.mentee_max_account_age_days', '365'), ('pap.fc_salary', '100'),
    ('webhook.type', 'discord'), ('webhook.url', 'https://example.invalid'),
    ('webhook.enabled', 'true'), ('mumble.public_address', 'old-address'),
    ('mumble.public_port', '64738'), ('onebot.enabled', 'false');
INSERT INTO task_schedules VALUES ('captain_attribution_sync'), ('captain_reward_processing'),
    ('active_task'), ('qq_structure_alert'), ('qq_official_structure_scan');
INSERT INTO task_execution SELECT task_name FROM task_schedules;
INSERT INTO eve_character_notification VALUES (1);
INSERT INTO qqbot_attempt VALUES (1);
INSERT INTO qq_official_attempt VALUES (1);
INSERT INTO shop_lottery_activity VALUES (1);
INSERT INTO shop_lottery_prize VALUES (1);
INSERT INTO wallet_log VALUES (1);
INSERT INTO "user" VALUES (1, 'active user', '12345', 'unused field');

\unset apply
\ir ../cleanup-pre-2.0.sql
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM cleanup_v2_fixture.eve_character_notification WHERE id = 1)
        OR NOT EXISTS (SELECT FROM qqbot_attempt WHERE id = 1)
        OR NOT EXISTS (SELECT FROM shop_lottery_prize WHERE activity_id = 1)
        OR NOT EXISTS (SELECT FROM "user" WHERE id = 1 AND avatar = 'unused field')
        OR NOT EXISTS (SELECT FROM fuxi_hall_card WHERE title = 'unused field')
        OR (SELECT count(*) FROM system_config) <> 14
        OR (SELECT count(*) FROM task_schedules) <> 5
        OR (SELECT count(*) FROM task_execution) <> 5
        OR (SELECT count(*) FROM fuxi_hall_page) <> 3 THEN
        RAISE EXCEPTION 'default preview changed the fixture';
    END IF;
END $$;

\set apply true
\ir ../cleanup-pre-2.0.sql
DO $$
BEGIN
    IF to_regclass('cleanup_v2_fixture.eve_character_notification') IS NOT NULL
        OR to_regclass('cleanup_v2_fixture.qqbot_attempt') IS NOT NULL
        OR to_regclass('shop_lottery_activity') IS NOT NULL OR to_regclass('shop_lottery_prize') IS NOT NULL
        OR EXISTS (SELECT FROM information_schema.columns WHERE table_schema = 'cleanup_v2_fixture'
                   AND ((table_name = 'user' AND column_name = 'avatar')
                        OR (table_name = 'fuxi_hall_card' AND column_name = 'title')))
        OR EXISTS (SELECT FROM system_config WHERE key IN (
            'pap.admin_award', 'newbro.bonus_rate', 'mentor.mentee_max_account_age_days', 'onebot.enabled'))
        OR EXISTS (SELECT FROM task_schedules WHERE task_name NOT IN ('active_task', 'qq_official_structure_scan'))
        OR EXISTS (SELECT FROM task_execution WHERE task_name IN ('captain_attribution_sync', 'captain_reward_processing'))
        OR EXISTS (SELECT FROM fuxi_hall_page WHERE page_key = 'leadership') THEN
        RAISE EXCEPTION 'retired objects or records remain';
    END IF;
    IF NOT EXISTS (SELECT FROM fuxi_hall_page WHERE page_key = 'hall-of-fame')
        OR NOT EXISTS (SELECT FROM fuxi_hall_page WHERE page_key = 'contributors')
        OR NOT EXISTS (SELECT FROM fuxi_hall_card WHERE page_key = 'contributors' AND deleted_at IS NOT NULL)
        OR NOT EXISTS (SELECT FROM system_config WHERE key = 'mumble.public_address' AND value = 'old-address')
        OR NOT EXISTS (SELECT FROM system_config WHERE key = 'mumble.public_port' AND value = '64738')
        OR NOT EXISTS (SELECT FROM system_config WHERE key = 'pap.fc_salary' AND value = '100')
        OR NOT EXISTS (SELECT FROM system_config WHERE key = 'webhook.type' AND value = 'discord')
        OR NOT EXISTS (SELECT FROM system_config WHERE key = 'webhook.url' AND value = 'https://example.invalid')
        OR NOT EXISTS (SELECT FROM system_config WHERE key = 'webhook.enabled' AND value = 'true')
        OR NOT EXISTS (SELECT FROM task_schedules WHERE task_name = 'active_task')
        OR NOT EXISTS (SELECT FROM task_schedules WHERE task_name = 'qq_official_structure_scan')
        OR NOT EXISTS (SELECT FROM task_execution WHERE task_name = 'active_task')
        OR NOT EXISTS (SELECT FROM task_execution WHERE task_name = 'qq_structure_alert')
        OR NOT EXISTS (SELECT FROM task_execution WHERE task_name = 'qq_official_structure_scan')
        OR NOT EXISTS (SELECT FROM qq_official_attempt WHERE id = 1)
        OR NOT EXISTS (SELECT FROM wallet_log WHERE id = 1)
        OR NOT EXISTS (SELECT FROM "user" WHERE id = 1 AND name = 'active user' AND qq = '12345')
        OR NOT EXISTS (SELECT FROM cleanup_v2_other.eve_character_notification WHERE id = 2)
        OR NOT EXISTS (SELECT FROM cleanup_v2_other.qqbot_attempt WHERE id = 2) THEN
        RAISE EXCEPTION 'protected records changed';
    END IF;
END $$;

INSERT INTO system_config VALUES ('mumble.public_nodes', '[]');
\ir ../cleanup-pre-2.0.sql
\ir ../cleanup-pre-2.0.sql
DO $$
BEGIN
    IF EXISTS (SELECT FROM system_config WHERE key IN ('mumble.public_address', 'mumble.public_port'))
        OR NOT EXISTS (SELECT FROM system_config WHERE key = 'mumble.public_nodes' AND value = '[]')
        OR (SELECT count(*) FROM system_config) <> 9 THEN
        RAISE EXCEPTION 'conditional Mumble cleanup or repeat apply failed';
    END IF;
END $$;

-- Running just the retirement script disables a legacy Webhook, but does not
-- include the general pre-2.0 cleanup.
UPDATE system_config SET value = 'qq_governance_onebot' WHERE key = 'webhook.type';
INSERT INTO system_config VALUES ('pap.admin_award', '20'), ('webhook.fleet_template', 'keep');
\ir ../qq-bot-retirement.sql
DO $$
BEGIN
    IF EXISTS (SELECT FROM system_config WHERE key IN ('webhook.type', 'webhook.url'))
        OR NOT EXISTS (SELECT FROM system_config WHERE key = 'webhook.enabled' AND value = 'false')
        OR NOT EXISTS (SELECT FROM system_config WHERE key = 'webhook.fleet_template' AND value = 'keep')
        OR NOT EXISTS (SELECT FROM system_config WHERE key = 'pap.admin_award' AND value = '20') THEN
        RAISE EXCEPTION 'standalone retirement cleanup changed unexpected settings';
    END IF;
END $$;
