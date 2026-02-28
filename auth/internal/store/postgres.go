package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Store handles database operations
type Store struct {
	db *sql.DB
}

// NewStore creates a new store instance
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// GetUserByEmail retrieves a user by email
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := s.db.QueryRowContext(ctx, queryGetUserByEmail, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	return &user, nil
}

// GetUserByID retrieves a user by ID
func (s *Store) GetUserByID(ctx context.Context, userID uuid.UUID) (*User, error) {
	var user User
	err := s.db.QueryRowContext(ctx, queryGetUserByID, userID).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	return &user, nil
}

// GetUserRoles retrieves roles for a user
func (s *Store) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, queryGetUserRoles, userID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return roles, nil
}

// GetUserWithRoles retrieves a user with their roles
func (s *Store) GetUserWithRoles(ctx context.Context, userID uuid.UUID) (*UserWithRoles, error) {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	roles, err := s.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &UserWithRoles{
		ID:        user.ID,
		Email:     user.Email,
		Roles:     roles,
		CreatedAt: user.CreatedAt,
	}, nil
}

// CreateUser creates a new user
func (s *Store) CreateUser(ctx context.Context, email, passwordHash string) (*User, error) {
	var user User
	err := s.db.QueryRowContext(ctx, queryCreateUser, email, passwordHash).Scan(
		&user.ID,
		&user.Email,
		&user.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create user error: %w", err)
	}
	user.PasswordHash = passwordHash
	return &user, nil
}

// GetRoleByName retrieves a role by name
func (s *Store) GetRoleByName(ctx context.Context, name string) (*Role, error) {
	var role Role
	err := s.db.QueryRowContext(ctx, queryGetRoleByName, name).Scan(&role.ID, &role.Name)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("role not found")
	}
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	return &role, nil
}

// IsEmailWhitelisted checks if an email is in the allowed_emails table
func (s *Store) IsEmailWhitelisted(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, queryIsEmailWhitelisted, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("database error: %w", err)
	}
	return exists, nil
}

// AddUserRole adds a role to a user
func (s *Store) AddUserRole(ctx context.Context, userID uuid.UUID, roleID int) error {
	_, err := s.db.ExecContext(ctx, queryAddUserRole, userID, roleID)
	if err != nil {
		return fmt.Errorf("add user role error: %w", err)
	}
	return nil
}

// RegisterUser atomically creates a user and assigns the named role in one transaction.
// If any step fails, the entire operation is rolled back — no orphan users are left
// without a role.
func (s *Store) RegisterUser(ctx context.Context, email, passwordHash, roleName string) (*User, []string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() // no-op if Commit() succeeds

	// Create user
	var user User
	err = tx.QueryRowContext(ctx, queryCreateUser, email, passwordHash).Scan(
		&user.ID,
		&user.Email,
		&user.CreatedAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create user: %w", err)
	}

	// Get role by name
	var role Role
	err = tx.QueryRowContext(ctx, queryGetRoleByName, roleName).Scan(&role.ID, &role.Name)
	if err == sql.ErrNoRows {
		return nil, nil, fmt.Errorf("role not found: %s", roleName)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("get role: %w", err)
	}

	// Assign role to user
	_, err = tx.ExecContext(ctx, queryAddUserRole, user.ID, role.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("add user role: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &user, []string{role.Name}, nil
}
