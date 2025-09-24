-- name: CreateNotification :one
-- Запрос создает новое уведомление в базе данных;
-- Возвращает созданную запись целиком
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
-- Запрос получает одно уведомление по его UUID;
-- Используется для проверки статуса или деталей уведомления
SELECT * FROM notifications
WHERE id = $1;

-- name: UpdateNotificationStatus :one
-- Запрос обновляет статус, счетчик попыток и время отправки уведомления;
-- Используется воркером после попытки отправки
UPDATE notifications
SET
    status = $2,    -- Новый статус: sent, failed, etc.
    attempts = $3,  -- Увеличиваем счетчик попыток
    sent_at = $4    -- Время фактической отправки (если успешно)
WHERE
    id = $1
    RETURNING *;

-- name: CancelNotification :one
-- Запрос выполняет "мягкое удаление" путем изменения статуса на 'declined'
-- Мы никогда не удаляем данные полностью, только меняем их состояние
-- Это обеспечивает аудит и историчность данных
UPDATE notifications
SET
    status = 'declined'  -- Меняем статус на отмененный
WHERE
    id = $1
    RETURNING *;

-- name: GetPendingNotificationsForUpdate :many
-- Блокирует строки для обновления в транзакции
SELECT * FROM notifications
WHERE status = 'pending'
ORDER BY scheduled_at ASC
FOR UPDATE SKIP LOCKED
LIMIT $1;

-- name: SetNotificationStatusSentMany :exec
UPDATE notifications
SET status = 'sent'
WHERE id = ANY(@ids::uuid[]);

-- name: SetNotificationStatusFailedMany :exec
UPDATE notifications
SET status = 'failed'
WHERE id = ANY(@ids::uuid[]);
