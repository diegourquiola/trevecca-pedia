package store

const (
	queryGetUserByEmail = `
		SELECT id, email, password_hash, created_at 
		FROM users 
		WHERE email = $1
	`

	queryGetUserRoles = `
		SELECT r.name 
		FROM roles r
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY r.name
	`

	queryGetUserByID = `
		SELECT id, email, password_hash, created_at 
		FROM users 
		WHERE id = $1
	`

	queryCreateUser = `
		INSERT INTO users (email, password_hash) 
		VALUES ($1, $2) 
		RETURNING id, email, created_at
	`

	queryGetRoleByName = `
		SELECT id, name 
		FROM roles 
		WHERE name = $1
	`

	queryAddUserRole = `
		INSERT INTO user_roles (user_id, role_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`

	queryIsEmailWhitelisted = `
		SELECT EXISTS (
			SELECT 1 FROM allowed_emails WHERE email = $1
		)
	`
)
