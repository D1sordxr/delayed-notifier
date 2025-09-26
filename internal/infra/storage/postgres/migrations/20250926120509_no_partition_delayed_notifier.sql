-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE channel_type AS ENUM ('email', 'telegram', 'sms');
CREATE TYPE notification_status AS ENUM ('pending', 'sent', 'failed', 'declined');

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject TEXT NOT NULL,
    message TEXT NOT NULL,
    author_id TEXT,
    email_to TEXT,
    telegram_chat_id BIGINT,
    sms_to TEXT,
    channel channel_type NOT NULL,
    status notification_status NOT NULL DEFAULT 'pending',
    attempts SMALLINT NOT NULL DEFAULT 0,
    scheduled_at TIMESTAMPTZ NOT NULL,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_recipient_channel
        CHECK (
            (channel = 'sms' AND sms_to IS NOT NULL AND email_to IS NULL AND telegram_chat_id IS NULL) OR
            (channel = 'email' AND email_to IS NOT NULL AND sms_to IS NULL AND telegram_chat_id IS NULL) OR
            (channel = 'telegram' AND telegram_chat_id IS NOT NULL AND email_to IS NULL AND sms_to IS NULL)
        )
);

CREATE INDEX idx_notifications_status_scheduled_at ON notifications (status, scheduled_at);
CREATE INDEX idx_notifications_scheduled_at ON notifications (scheduled_at);

-- +goose Down
DROP TABLE IF EXISTS notifications;
DROP TYPE IF EXISTS notification_status;
DROP TYPE IF EXISTS channel_type;
DROP EXTENSION IF EXISTS "pgcrypto";