-- name: ListUsers :many
SELECT id, name, email FROM users ORDER BY name;

-- name: GetUser :one
SELECT id, name, email FROM users WHERE id = $1;

-- name: ListMemberships :many
SELECT m.hospital_id, h.name AS hospital_name, m.role
FROM memberships m
JOIN hospitals h ON h.id = m.hospital_id
WHERE m.user_id = $1
ORDER BY h.name, m.role;

-- name: ListRolesInHospital :many
SELECT role FROM memberships WHERE user_id = $1 AND hospital_id = $2;
