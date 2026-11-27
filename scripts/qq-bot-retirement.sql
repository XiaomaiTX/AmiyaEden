-- Explicit maintenance only. Run with psql -X -d DB -v target_database=DB
-- -v target_schema=SCHEMA -f scripts/qq-bot-retirement.sql
-- Add -v apply=true only after reviewing preview and verifying backups.
\set ON_ERROR_STOP on
\set VERBOSITY terse
\if :{?target_database}
\else
\echo 'target_database is required'
SELECT 1/0;
\endif
\if :{?target_schema}
\else
\echo 'target_schema is required'
SELECT 1/0;
\endif
\if :{?apply}
\else
\set apply false
\endif

BEGIN;
\if :apply
\else
-- Read-only mode is set before the first query so a preview can never write.
SET TRANSACTION READ ONLY;
\endif
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL lock_timeout = '5s';
-- Fail rather than silently count or delete only rows visible to an RLS policy.
SET LOCAL row_security = off;
SET LOCAL client_min_messages = notice;
SELECT set_config('retirement.target_database', :'target_database', true) AS db,
       set_config('retirement.target_schema', :'target_schema', true) AS schema,
       set_config('retirement.apply', :'apply', true) AS mode \gset

DO $cleanup$
DECLARE
    target text := current_setting('retirement.target_schema');
    mode text := current_setting('retirement.apply');
    apply_changes boolean := (mode = 'true');
    -- The 16 retired models, then the 12 names in the legacy cleanup guide.
    bot_tables constant text[] := ARRAY[
        'qqbot_operation', 'qqbot_attempt', 'qqbot_inbox', 'qqbot_delivery',
        'qqbot_circuit', 'qqbot_setting', 'qq_notification_channel',
        'qq_notification', 'qq_notification_target', 'qq_structure_alert',
        'qq_group_rule', 'qq_group_application', 'qq_group_member',
        'qq_group_applied_event', 'qq_group_inspection', 'qq_group_inspection_item',
        'qq_group_governance_policy', 'qq_governance_event', 'qq_group_member_state',
        'qq_group_runtime_snapshot', 'qq_governance_reconcile_run',
        'qq_governance_reconcile_member', 'qq_governance_review',
        'qq_governance_action_task', 'qq_governance_action_log',
        'qq_governance_risk_control_state', 'qq_governance_alert', 'corp_structure_alert_state'
    ];
    old_keys constant text[] := ARRAY[
        'onebot.enabled', 'onebot.access_token', 'onebot.bot_qq', 'onebot.allowed_cidrs',
        'qq_governance.scan_interval_minutes', 'qq_governance.mismatch_confirmations',
        'qq_governance.mismatch_observation_hours', 'webhook.ob_target_type',
        'webhook.ob_target_id', 'webhook.ob_token', 'webhook.qq_group_ids',
        'dashboard.corporation_structures_alert_enabled', 'dashboard.corporation_structures_alert_group_ids'
    ];
    old_tasks constant text[] := ARRAY[
        'qq_group_business', 'qq_structure_alert',
        'qq_group_governance_reconcile', 'qq_governance_maintenance', 'corporation_structure_alert_scan'
    ];
    old_webhook_types constant text[] := ARRAY['onebot', 'qq_governance_onebot'];
    name text;
    eligible boolean;
    rel record;
    qualified text;
    drop_list text[] := ARRAY[]::text[];
    row_count bigint;
BEGIN
    IF mode NOT IN ('true', 'false') THEN
        RAISE EXCEPTION 'apply must be true or false';
    END IF;
    IF current_database()::text <> current_setting('retirement.target_database') THEN
        RAISE EXCEPTION 'connected database does not match target_database';
    END IF;
    IF target IN ('information_schema', '') OR target LIKE 'pg\_%'
        OR NOT EXISTS (SELECT FROM pg_namespace WHERE nspname::text = target) THEN
        RAISE EXCEPTION 'target_schema must be an existing non-system schema';
    END IF;
    RAISE NOTICE 'Mode: %, database: %, schema: %', mode, current_database(), target;

    FOREACH name IN ARRAY bot_tables || ARRAY['system_config', 'task_schedules', 'task_schedule'] LOOP
        SELECT c.oid, c.relkind INTO rel FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname::text = target AND c.relname::text = name;
        IF NOT FOUND THEN
            RAISE NOTICE '%: absent', name;
            CONTINUE;
        END IF;
        IF rel.relkind <> 'r' THEN
            RAISE EXCEPTION 'allowlisted object % is not an ordinary table', name;
        END IF;
        qualified := format('%I.%I', target, name);
        -- Shared tables are only touched through columns this script understands.
        IF name = 'system_config' THEN
            IF NOT EXISTS (SELECT FROM pg_attribute WHERE attrelid = rel.oid AND attname = 'key' AND NOT attisdropped)
                OR NOT EXISTS (SELECT FROM pg_attribute WHERE attrelid = rel.oid AND attname = 'value' AND NOT attisdropped) THEN
                RAISE EXCEPTION 'table % does not have the expected key/value columns', name;
            END IF;
        ELSIF name = 'task_schedules' OR name = 'task_schedule' THEN
            IF NOT EXISTS (SELECT FROM pg_attribute WHERE attrelid = rel.oid AND attname = 'task_name' AND NOT attisdropped) THEN
                RAISE EXCEPTION 'table % does not have the expected task_name column', name;
            END IF;
        END IF;
        IF apply_changes THEN
            EXECUTE format('LOCK TABLE ONLY %s IN ACCESS EXCLUSIVE MODE', qualified);
            IF to_regclass(qualified)::oid IS DISTINCT FROM rel.oid THEN
                RAISE EXCEPTION 'allowlisted object % changed during cleanup', name;
            END IF;
        END IF;
        IF EXISTS (SELECT FROM pg_inherits WHERE inhrelid = rel.oid OR inhparent = rel.oid) THEN
            RAISE EXCEPTION 'refusing inherited or partitioned table %', name;
        END IF;

        IF name = ANY(bot_tables) THEN
            EXECUTE format('SELECT count(*) FROM ONLY %s', qualified) INTO row_count;
            RAISE NOTICE 'Drop table % (% rows)', name, row_count;
            drop_list := array_append(drop_list, qualified);
        ELSIF name = 'system_config' THEN
            EXECUTE format('SELECT count(*) FROM ONLY %s WHERE key = ANY($1)', qualified)
                INTO row_count USING old_keys;
            RAISE NOTICE '%: % legacy config rows to remove', name, row_count;
            EXECUTE format('SELECT EXISTS (SELECT FROM ONLY %s WHERE key = ''webhook.type'' AND value = ANY($1))', qualified)
                INTO eligible USING old_webhook_types;
            IF eligible THEN
                EXECUTE format('SELECT count(*) FROM ONLY %s WHERE key IN (''webhook.type'', ''webhook.url'')', qualified)
                    INTO row_count;
                RAISE NOTICE '%: % retired Webhook type/URL rows to remove; set webhook.enabled=false', name, row_count;
            ELSE
                RAISE NOTICE '%: preserve Webhook type, URL and enabled setting', name;
            END IF;
            IF apply_changes THEN
                EXECUTE format('DELETE FROM ONLY %s WHERE key = ANY($1)', qualified) USING old_keys;
                IF eligible THEN
                    EXECUTE format('INSERT INTO %s (key, value) VALUES (''webhook.enabled'', ''false'') '
                        'ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value', qualified);
                    EXECUTE format('DELETE FROM ONLY %s WHERE key IN (''webhook.type'', ''webhook.url'')', qualified);
                END IF;
            END IF;
        ELSE
            EXECUTE format('SELECT count(*) FROM ONLY %s WHERE task_name = ANY($1)', qualified)
                INTO row_count USING old_tasks;
            RAISE NOTICE '%: % retired schedule overrides to remove; executions untouched', name, row_count;
            IF apply_changes THEN
                EXECUTE format('DELETE FROM ONLY %s WHERE task_name = ANY($1)', qualified) USING old_tasks;
            END IF;
        END IF;
    END LOOP;

    IF apply_changes AND cardinality(drop_list) > 0 THEN
        -- One statement supports dependencies among allowlisted tables, but fails on external dependencies.
        EXECUTE 'DROP TABLE ' || array_to_string(drop_list, ', ') || ' RESTRICT';
    END IF;
END;
$cleanup$;
\if :{?cleanup_pre_2_0}
\if :cleanup_pre_2_0
\ir cleanup-pre-2.0-extra.sql
\endif
\endif
\if :apply
COMMIT;
\else
ROLLBACK;
\echo 'Preview only: no changes committed.'
\endif
