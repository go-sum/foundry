-- Auto-generated CRUD queries for contact_submissions
-- Edit as needed, then run: db codegen

-- name: InsertContactSubmission :one
INSERT INTO contact_submissions (name, email, message, ip_address)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetContactSubmission :one
SELECT * FROM contact_submissions
WHERE id = $1;

-- name: ListContactSubmissions :many
SELECT * FROM contact_submissions
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListContactSubmissionsByEmailAndCreatedAt :many
SELECT * FROM contact_submissions
WHERE email = $1 AND created_at >= $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: DeleteContactSubmission :exec
DELETE FROM contact_submissions
WHERE id = $1;
