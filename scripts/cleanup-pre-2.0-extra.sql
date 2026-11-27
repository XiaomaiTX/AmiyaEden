-- Included by cleanup-pre-2.0.sql through qq-bot-retirement.sql in the same transaction.
DO $cleanup$
DECLARE
    target text := current_setting('retirement.target_schema');
    apply_changes boolean := current_setting('retirement.apply') = 'true';
    gone_tables constant text[] := ARRAY[
        'newbro_captain_affiliation', 'captain_bounty_attribution',
        'captain_bounty_sync_state', 'captain_reward_settlement', 'newbro_player_state',
        'shop_lottery_record', 'shop_lottery_prize', 'shop_lottery_activity',
        'shop_redeem_code', 'srp_payout_mail_log', 'eve_character_notification'
    ];
    required_tables constant text[] := ARRAY[
        'announcement', 'captain_refund', 'system_config', 'fuxi_hall_page', 'fuxi_hall_card'
    ];
    cleanup_tables constant text[] := gone_tables || ARRAY[
        'fleet_config_fitting', 'user', 'eve_character', 'hall_of_fame_card',
        'fuxi_hall_card', 'ticket', 'system_config',
        'task_schedule', 'task_schedules', 'task_execution', 'task_executions'
    ];
    old_keys text[] := ARRAY[
        'pap.admin_award', 'newbro.max_character_sp', 'newbro.multi_character_sp',
        'newbro.multi_character_threshold', 'newbro.refresh_interval_days',
        'newbro.bonus_rate', 'mentor.mentee_max_character_sp', 'mentor.mentee_max_account_age_days'
    ];
    salary_keys constant text[] := ARRAY[
        'salary.recruitment', 'salary.welfare_delivery', 'salary.shop_delivery', 'salary.srp_payout'
    ];
    captain_tasks constant text[] := ARRAY['captain_attribution_sync', 'captain_reward_processing'];
    config_table text := format('%I.system_config', target);
    page_table text := format('%I.fuxi_hall_page', target);
    card_table text := format('%I.fuxi_hall_card', target);
    page_filter text;
    hall_keys constant text[] := ARRAY['leadership', 'contributors'];
    relation_name text;
    qualified text;
    relation_oid oid;
    required_columns text[];
    validated_tables jsonb := '{}'::jsonb;
    entry record;
    drops text[] := ARRAY[]::text[];
    affected bigint;
    ready boolean;
BEGIN
    -- Validate relations and (in apply mode) lock them before reading migration
    -- markers or touching data. The operator must still verify startup completed.
    FOR relation_name IN
        SELECT DISTINCT name FROM unnest(required_tables || cleanup_tables) AS tables(name) ORDER BY name
    LOOP
        qualified := format('%I.%I', target, relation_name);
        relation_oid := to_regclass(qualified)::oid;
        IF relation_oid IS NULL THEN
            IF relation_name = ANY(required_tables) THEN
                RAISE EXCEPTION 'upgrade to 2.0.0+ and complete startup first (missing %)', qualified;
            END IF;
            CONTINUE;
        END IF;
        IF NOT EXISTS (SELECT FROM pg_class WHERE oid = relation_oid AND relkind = 'r') THEN
            RAISE EXCEPTION 'allowlisted object % is not an ordinary table', qualified;
        END IF;
        IF apply_changes THEN
            EXECUTE format('LOCK TABLE ONLY %s IN ACCESS EXCLUSIVE MODE', qualified);
            IF to_regclass(qualified)::oid IS DISTINCT FROM relation_oid THEN
                RAISE EXCEPTION '% changed during cleanup', qualified;
            END IF;
        END IF;
        IF EXISTS (SELECT FROM pg_inherits WHERE inhrelid = relation_oid OR inhparent = relation_oid) THEN
            RAISE EXCEPTION 'refusing inherited or partitioned table %', qualified;
        END IF;

        required_columns := CASE
            WHEN relation_name = 'system_config' THEN ARRAY['key', 'value']
            WHEN relation_name IN ('fuxi_hall_page', 'fuxi_hall_card') THEN ARRAY['page_key']
            WHEN relation_name IN ('task_schedule', 'task_schedules', 'task_execution', 'task_executions')
                THEN ARRAY['task_name']
        END;
        IF EXISTS (
            SELECT FROM unnest(required_columns) AS required(column_name)
            WHERE NOT EXISTS (
                SELECT FROM pg_attribute WHERE attrelid = relation_oid
                    AND attname::text = required.column_name AND attnum > 0 AND NOT attisdropped
            )
        ) THEN
            RAISE EXCEPTION 'table % does not have the expected columns: %',
                qualified, array_to_string(required_columns, ', ');
        END IF;
        validated_tables := validated_tables || jsonb_build_object(relation_name, relation_oid);
    END LOOP;

    EXECUTE format('SELECT count(DISTINCT key) FROM ONLY %s WHERE key = ANY($1)', config_table)
        INTO affected USING salary_keys;
    IF affected <> cardinality(salary_keys) THEN
        RAISE EXCEPTION 'operation salary migration incomplete; legacy settings not removed';
    END IF;
    EXECUTE format('SELECT EXISTS (SELECT FROM ONLY %s WHERE page_key = ''hall-of-fame'')', page_table)
        INTO ready;
    IF NOT ready THEN
        RAISE EXCEPTION 'Fuxi Hall migration incomplete (missing hall-of-fame page)';
    END IF;
    EXECUTE format('SELECT EXISTS (SELECT FROM ONLY %s WHERE key = ''mumble.public_nodes'')', config_table)
        INTO ready;
    IF ready THEN
        old_keys := old_keys || ARRAY['mumble.public_address', 'mumble.public_port'];
    ELSE
        RAISE NOTICE 'No mumble.public_nodes setting; preserve any legacy Mumble settings for review';
    END IF;

    FOREACH relation_name IN ARRAY cleanup_tables LOOP
        -- Do not process a table that appeared after preflight.
        relation_oid := (validated_tables ->> relation_name)::oid;
        IF relation_oid IS NULL THEN
            CONTINUE;
        END IF;
        qualified := format('%I.%I', target, relation_name);
        IF to_regclass(qualified)::oid IS DISTINCT FROM relation_oid THEN
            RAISE EXCEPTION '% changed during cleanup', qualified;
        END IF;

        IF relation_name = ANY(gone_tables) THEN
            EXECUTE format('SELECT count(*) FROM ONLY %s', qualified) INTO affected;
            RAISE NOTICE 'Drop table % (% rows)', qualified, affected;
            drops := array_append(drops, qualified);
        ELSIF relation_name = 'system_config' THEN
            EXECUTE format('SELECT count(*) FROM ONLY %s WHERE key = ANY($1)', qualified)
                INTO affected USING old_keys;
            RAISE NOTICE '%: remove % retired settings', qualified, affected;
            IF apply_changes THEN
                EXECUTE format('DELETE FROM ONLY %s WHERE key = ANY($1)', qualified) USING old_keys;
            END IF;
        ELSIF relation_name IN ('task_schedule', 'task_schedules', 'task_execution', 'task_executions') THEN
            EXECUTE format('SELECT count(*) FROM ONLY %s WHERE task_name = ANY($1)', qualified)
                INTO affected USING captain_tasks;
            RAISE NOTICE '%: remove % retired captain task records', qualified, affected;
            IF apply_changes THEN
                EXECUTE format('DELETE FROM ONLY %s WHERE task_name = ANY($1)', qualified) USING captain_tasks;
            END IF;
        ELSE
            -- Keep this list aligned with bootstrap.obsoleteColumnDrops.
            FOR entry IN SELECT v.column_name FROM (VALUES
                ('fleet_config_fitting', 'eft'), ('fleet_config_fitting', 'ship_name'),
                ('user', 'avatar'), ('eve_character', 'portrait_url'), ('hall_of_fame_card', 'avatar'),
                ('fuxi_hall_card', 'style_preset'), ('fuxi_hall_card', 'badge_tone'),
                ('fuxi_hall_card', 'cover_height'), ('fuxi_hall_card', 'cover_image'),
                ('fuxi_hall_card', 'title'), ('ticket', 'priority')
            ) AS v(table_name, column_name)
            WHERE v.table_name = relation_name AND EXISTS (
                SELECT FROM pg_attribute WHERE attrelid = relation_oid
                    AND attname::text = v.column_name AND attnum > 0 AND NOT attisdropped
            ) LOOP
                RAISE NOTICE 'Drop column %.%', qualified, entry.column_name;
                IF apply_changes THEN
                    EXECUTE format('ALTER TABLE ONLY %s DROP COLUMN %I RESTRICT', qualified, entry.column_name);
                END IF;
            END LOOP;
        END IF;
    END LOOP;

    -- Keep a legacy page if any card still points to it, including soft-deleted cards.
    page_filter := format('p.page_key = ANY($1) AND NOT EXISTS '
        '(SELECT FROM ONLY %s AS c WHERE c.page_key = p.page_key)', card_table);
    EXECUTE format('SELECT count(*) FROM ONLY %s AS p WHERE %s', page_table, page_filter)
        INTO affected USING hall_keys;
    RAISE NOTICE '%: remove % unreferenced legacy pages', page_table, affected;
    IF apply_changes THEN
        EXECUTE format('DELETE FROM ONLY %s AS p WHERE %s', page_table, page_filter) USING hall_keys;
    END IF;

    IF apply_changes AND cardinality(drops) > 0 THEN
        -- Drop together to allow dependencies within the allowlist, not outside it.
        EXECUTE 'DROP TABLE ' || array_to_string(drops, ', ') || ' RESTRICT';
    END IF;
END;
$cleanup$;
