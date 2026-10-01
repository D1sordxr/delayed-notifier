-- +goose Up
-- queued_at marks a pending notification as handed over to the broker;
-- a notification that stays pending long after it was queued is re-claimed.
ALTER TABLE notifications ADD COLUMN queued_at TIMESTAMPTZ;

CREATE INDEX idx_notifications_pending_scheduled_at
    ON notifications (scheduled_at)
    WHERE status = 'pending';

-- +goose Down
DROP INDEX IF EXISTS idx_notifications_pending_scheduled_at;
ALTER TABLE notifications DROP COLUMN IF EXISTS queued_at;
