-- +goose Up
-- Old GORM naming split the WhatsApp initialism as `whats_app_account`.
-- The application and versioned schema consistently use `whatsapp_account`.
-- +goose StatementBegin
DO $$
DECLARE
    target_table text;
BEGIN
    FOREACH target_table IN ARRAY ARRAY[
        'agent_transfers', 'ai_contexts', 'bulk_message_campaigns',
        'call_permissions', 'call_transfers', 'catalogs', 'chatbot_flows',
        'chatbot_sessions', 'chatbot_settings', 'contacts', 'keyword_rules',
        'messages', 'notification_rules', 'templates', 'whatsapp_flows'
    ]
    LOOP
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = target_table AND column_name = 'whats_app_account')
           AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = target_table AND column_name = 'whatsapp_account') THEN
            EXECUTE format('ALTER TABLE %I RENAME COLUMN whats_app_account TO whatsapp_account', target_table);
        END IF;
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
DECLARE
    target_table text;
BEGIN
    FOREACH target_table IN ARRAY ARRAY[
        'agent_transfers', 'ai_contexts', 'bulk_message_campaigns',
        'call_permissions', 'call_transfers', 'catalogs', 'chatbot_flows',
        'chatbot_sessions', 'chatbot_settings', 'contacts', 'keyword_rules',
        'messages', 'notification_rules', 'templates', 'whatsapp_flows'
    ]
    LOOP
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = target_table AND column_name = 'whatsapp_account')
           AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = target_table AND column_name = 'whats_app_account') THEN
            EXECUTE format('ALTER TABLE %I RENAME COLUMN whatsapp_account TO whats_app_account', target_table);
        END IF;
    END LOOP;
END $$;
-- +goose StatementEnd
