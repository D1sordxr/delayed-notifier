-- +goose Up

-- pgcrypto используется для функции gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Использование ENUM предотвращает вставку недопустимых значений в эти столбцы
CREATE TYPE channel_type AS ENUM ('email', 'telegram', 'sms');
CREATE TYPE notification_status AS ENUM ('pending', 'sent', 'failed', 'declined');

-- Таблица партиционируется по `scheduled_at` для эффективных запросов по времени
-- Фактические данные будут храниться в дочерних таблицах-партициях
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject TEXT NOT NULL,
    message TEXT NOT NULL,
    author_id TEXT,

    -- Поля для получателя
    email_to TEXT,
    telegram_chat_id BIGINT,
    sms_to TEXT,

    -- Основные поля уведомления
    channel channel_type NOT NULL,
    status notification_status NOT NULL DEFAULT 'pending',
    attempts SMALLINT NOT NULL DEFAULT 0,

    -- Временные метки
    scheduled_at TIMESTAMPTZ NOT NULL,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

-- Проверочное ограничение: данные получателя должны соответствовать выбранному каналу
    CONSTRAINT chk_recipient_channel
        CHECK (
            (channel = 'sms' AND sms_to IS NOT NULL AND email_to IS NULL AND telegram_chat_id IS NULL) OR
            (channel = 'email' AND email_to IS NOT NULL AND sms_to IS NULL AND telegram_chat_id IS NULL) OR
            (channel = 'telegram' AND telegram_chat_id IS NOT NULL AND email_to IS NULL AND sms_to IS NULL)
        )
) PARTITION BY RANGE (scheduled_at);

-- Это делает таблицу готовой к использованию сразу после миграции
-- Последующие партиции будут создаваться автоматически с помощью pg_partman или cron-задачи
CREATE TABLE notifications_2025_09 PARTITION OF notifications
    FOR VALUES FROM ('2025-09-01 00:00:00Z') TO ('2025-09-30 23:59:59Z');

-- Этот индекс критически важен для быстрого поиска заданий для обработки воркером
CREATE INDEX idx_notifications_status_scheduled_at ON notifications (status, scheduled_at);

CREATE OR REPLACE FUNCTION trigger_set_timestamp()
    RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON notifications
    FOR EACH ROW
    EXECUTE PROCEDURE trigger_set_timestamp();

-- +goose Down
-- Откат миграции в порядке, обратном созданию
DROP TABLE IF EXISTS notifications;
DROP TRIGGER IF EXISTS set_timestamp ON notifications;
DROP FUNCTION IF EXISTS trigger_set_timestamp();
DROP TYPE IF EXISTS notification_status;
DROP TYPE IF EXISTS channel_type;
DROP EXTENSION IF EXISTS "pgcrypto";