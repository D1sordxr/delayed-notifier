-- name: CreateNotification :one
-- Создает новое уведомление; возвращает созданную запись целиком
INSERT INTO notifications (
    subject,           -- Тема уведомления
    message,           -- Текст сообщения
    author_id,         -- ID автора (может быть NULL)
    email_to,          -- Email получателя (для email канала)
    telegram_chat_id,  -- ID чата Telegram (для telegram канала)
    sms_to,            -- Телефон получателя
    channel,           -- Канал отправки: email, telegram, sms
    status,            -- Статус уведомления
    attempts,          -- Количество попыток отправки
    scheduled_at       -- Время планируемой отправки
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10
)
RETURNING *;

-- name: GetNotificationByID :one
-- Получает одно уведомление по его UUID
SELECT * FROM notifications
WHERE id = $1;

-- name: CancelNotification :one
-- "Мягкое удаление": отменить можно только еще не отправленное уведомление.
-- Если уведомление прямо сейчас отправляется, запрос дождется конца отправки
-- и не найдет строку, так как статус уже не 'pending'
UPDATE notifications
SET
    status = 'declined',
    updated_at = clock_timestamp()
WHERE id = $1
    AND status = 'pending'
RETURNING *;

-- name: ClaimDueNotifications :many
-- Забирает пачку уведомлений, время отправки которых наступает до due_before,
-- и которые еще не переданы в брокер (или переданы давно и застряли).
-- SKIP LOCKED позволяет нескольким воркерам работать параллельно без дублей
SELECT * FROM notifications
WHERE status = 'pending'
    AND scheduled_at <= @due_before
    AND (queued_at IS NULL OR queued_at < @stale_before::timestamptz)
ORDER BY scheduled_at
LIMIT @batch_size
FOR UPDATE SKIP LOCKED;

-- name: MarkNotificationsQueued :exec
-- Отмечает уведомления как переданные в брокер.
-- updated_at не меняется: видимое состояние уведомления осталось прежним
UPDATE notifications
SET queued_at = clock_timestamp()
WHERE id = ANY(@ids::uuid[]);

-- name: LockPendingNotification :one
-- Блокирует уведомление на время отправки. Если строку уже отправляет другой
-- обработчик, запрос ждет его и затем не находит строку (статус сменился)
SELECT * FROM notifications
WHERE id = $1
    AND status = 'pending'
FOR UPDATE;

-- name: MarkNotificationSent :one
UPDATE notifications
SET
    status = 'sent',
    attempts = attempts + 1,
    sent_at = clock_timestamp(),
    updated_at = clock_timestamp()
WHERE id = $1
RETURNING *;

-- name: MarkNotificationFailed :one
UPDATE notifications
SET
    status = 'failed',
    attempts = attempts + 1,
    updated_at = clock_timestamp()
WHERE id = $1
RETURNING *;

-- name: MarkNotificationRetry :one
-- Неудачная попытка, после которой будет повтор через retry-очередь
UPDATE notifications
SET
    attempts = attempts + 1,
    queued_at = clock_timestamp(),
    updated_at = clock_timestamp()
WHERE id = $1
RETURNING *;
