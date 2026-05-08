-- name: CreateConversation :one
INSERT INTO conversations (user_id, status)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateConversationStatus :one
UPDATE conversations
SET status = $2, mod_id = COALESCE($3, mod_id), closed_at = $4, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CreateMessage :one
INSERT INTO messages (conversation_id, sender_id, content)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetMessagesByConversation :many
SELECT * FROM messages
WHERE conversation_id = $1
  AND (sqlc.narg('cursor_id')::uuid IS NULL OR id < sqlc.narg('cursor_id')::uuid)
ORDER BY created_at DESC
LIMIT sqlc.arg('limit');

-- name: GetConversationsByIDs :many
SELECT * FROM conversations WHERE id = ANY($1::uuid[]);

-- name: GetMessagesByIDs :many
SELECT * FROM messages WHERE id = ANY($1::uuid[]);

-- name: CreateChatbotHistory :one
INSERT INTO chatbot_histories (user_id, question, answer)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetChatbotHistory :many
SELECT * FROM (
    SELECT * FROM chatbot_histories
    WHERE user_id = $1
      AND (sqlc.narg('cursor_id')::uuid IS NULL OR id < sqlc.narg('cursor_id')::uuid)
    ORDER BY created_at DESC
    LIMIT sqlc.arg('limit')
) sub
ORDER BY created_at ASC;

-- name: GetChatbotHistoriesByIDs :many
SELECT * FROM chatbot_histories WHERE id = ANY($1::uuid[]);
