package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/trading-chitti/market-bridge/internal/auth"
)

// CreateUser creates a new user account
func (db *Database) CreateUser(email, passwordHash, fullName string) (*auth.User, error) {
	query := `
		INSERT INTO auth.users (email, password_hash, full_name)
		VALUES ($1, $2, $3)
		RETURNING user_id, email, password_hash, full_name, created_at, updated_at,
		          last_login_at, is_active, email_verified
	`

	var user auth.User
	err := db.conn.QueryRow(query, email, passwordHash, fullName).Scan(
		&user.UserID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&user.IsActive,
		&user.EmailVerified,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (db *Database) GetUserByEmail(email string) (*auth.User, error) {
	query := `
		SELECT user_id, email, password_hash, full_name, created_at, updated_at,
		       last_login_at, is_active, email_verified
		FROM auth.users
		WHERE email = $1
	`

	var user auth.User
	err := db.conn.QueryRow(query, email).Scan(
		&user.UserID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&user.IsActive,
		&user.EmailVerified,
	)

	if err == sql.ErrNoRows {
		return nil, auth.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetUserByID retrieves a user by ID
func (db *Database) GetUserByID(userID string) (*auth.User, error) {
	query := `
		SELECT user_id, email, password_hash, full_name, created_at, updated_at,
		       last_login_at, is_active, email_verified
		FROM auth.users
		WHERE user_id = $1
	`

	var user auth.User
	err := db.conn.QueryRow(query, userID).Scan(
		&user.UserID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&user.IsActive,
		&user.EmailVerified,
	)

	if err == sql.ErrNoRows {
		return nil, auth.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// UpdateLastLogin updates the user's last login timestamp
func (db *Database) UpdateLastLogin(userID string) error {
	query := `UPDATE auth.users SET last_login_at = NOW() WHERE user_id = $1`
	_, err := db.conn.Exec(query, userID)
	return err
}

// CreateSession creates a new session for a user
func (db *Database) CreateSession(session *auth.Session) error {
	query := `
		INSERT INTO auth.sessions (
			session_id, user_id, token_hash, refresh_token_hash,
			expires_at, ip_address, user_agent
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := db.conn.Exec(
		query,
		session.SessionID,
		session.UserID,
		session.TokenHash,
		session.RefreshTokenHash,
		session.ExpiresAt,
		session.IPAddress,
		session.UserAgent,
	)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetSessionByToken retrieves a session by token hash
func (db *Database) GetSessionByToken(tokenHash string) (*auth.Session, error) {
	query := `
		SELECT session_id, user_id, token_hash, refresh_token_hash,
		       expires_at, created_at, last_used_at, ip_address,
		       user_agent, is_revoked
		FROM auth.sessions
		WHERE token_hash = $1
	`

	var session auth.Session
	err := db.conn.QueryRow(query, tokenHash).Scan(
		&session.SessionID,
		&session.UserID,
		&session.TokenHash,
		&session.RefreshTokenHash,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.LastUsedAt,
		&session.IPAddress,
		&session.UserAgent,
		&session.IsRevoked,
	)

	if err == sql.ErrNoRows {
		return nil, auth.ErrInvalidToken
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Check if session is expired or revoked
	if session.IsRevoked {
		return nil, auth.ErrSessionRevoked
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, auth.ErrInvalidToken
	}

	return &session, nil
}

// UpdateSessionLastUsed updates the last used timestamp
func (db *Database) UpdateSessionLastUsed(sessionID string) error {
	query := `UPDATE auth.sessions SET last_used_at = NOW() WHERE session_id = $1`
	_, err := db.conn.Exec(query, sessionID)
	return err
}

// RevokeSession marks a session as revoked
func (db *Database) RevokeSession(sessionID string) error {
	query := `UPDATE auth.sessions SET is_revoked = TRUE WHERE session_id = $1`
	_, err := db.conn.Exec(query, sessionID)
	return err
}

// RevokeAllUserSessions revokes all sessions for a user
func (db *Database) RevokeAllUserSessions(userID string) error {
	query := `UPDATE auth.sessions SET is_revoked = TRUE WHERE user_id = $1`
	_, err := db.conn.Exec(query, userID)
	return err
}

// CleanupExpiredSessions removes expired sessions
func (db *Database) CleanupExpiredSessions(ctx context.Context) error {
	query := `DELETE FROM auth.sessions WHERE expires_at < NOW() OR is_revoked = TRUE`
	_, err := db.conn.ExecContext(ctx, query)
	return err
}

// CreateAuditLog creates an audit log entry
func (db *Database) CreateAuditLog(userID, action, resourceType, resourceID, ipAddress, userAgent string, details map[string]interface{}) error {
	query := `
		INSERT INTO auth.audit_log (user_id, action, resource_type, resource_id, ip_address, user_agent, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	var detailsJSON sql.NullString
	if details != nil {
		// Convert details to JSON (simplified - in production use json.Marshal)
		detailsJSON.Valid = true
	}

	_, err := db.conn.Exec(query, userID, action, resourceType, resourceID, ipAddress, userAgent, detailsJSON)
	return err
}

// GetUserBrokerConfigs retrieves all broker configurations for a user
func (db *Database) GetUserBrokerConfigs(userID string) ([]*BrokerConfig, error) {
	query := `
		SELECT config_id, user_id, broker_name, api_key, api_secret, access_token,
		       refresh_token, token_expires_at, last_token_refresh, is_active,
		       account_name, is_default, created_at, updated_at
		FROM brokers.config
		WHERE user_id = $1
		ORDER BY is_default DESC, created_at DESC
	`

	rows, err := db.conn.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query broker configs: %w", err)
	}
	defer rows.Close()

	var configs []*BrokerConfig
	for rows.Next() {
		var config BrokerConfig
		var userIDNullable sql.NullString
		var accountNameNullable sql.NullString
		var isDefaultNullable sql.NullBool

		err := rows.Scan(
			&config.ConfigID,
			&userIDNullable,
			&config.BrokerName,
			&config.APIKey,
			&config.APISecret,
			&config.AccessToken,
			&config.RefreshToken,
			&config.TokenExpiresAt,
			&config.LastTokenRefresh,
			&config.IsActive,
			&accountNameNullable,
			&isDefaultNullable,
			&config.CreatedAt,
			&config.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan broker config: %w", err)
		}

		if userIDNullable.Valid {
			config.UserID = userIDNullable.String
		}
		if accountNameNullable.Valid {
			config.AccountName = accountNameNullable.String
		}
		if isDefaultNullable.Valid {
			config.IsDefault = isDefaultNullable.Bool
		}

		configs = append(configs, &config)
	}

	return configs, nil
}

// CreateUserBrokerConfig creates a new broker configuration for a user
func (db *Database) CreateUserBrokerConfig(userID, brokerName, apiKey, apiSecret, accountName string, isDefault bool) (*BrokerConfig, error) {
	query := `
		INSERT INTO brokers.config (user_id, broker_name, api_key, api_secret, account_name, is_default, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, TRUE)
		RETURNING config_id, user_id, broker_name, api_key, api_secret, access_token,
		          refresh_token, token_expires_at, last_token_refresh, is_active,
		          account_name, is_default, created_at, updated_at
	`

	var config BrokerConfig
	var userIDNullable sql.NullString
	var accountNameNullable sql.NullString
	var isDefaultNullable sql.NullBool

	err := db.conn.QueryRow(query, userID, brokerName, apiKey, apiSecret, accountName, isDefault).Scan(
		&config.ConfigID,
		&userIDNullable,
		&config.BrokerName,
		&config.APIKey,
		&config.APISecret,
		&config.AccessToken,
		&config.RefreshToken,
		&config.TokenExpiresAt,
		&config.LastTokenRefresh,
		&config.IsActive,
		&accountNameNullable,
		&isDefaultNullable,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create broker config: %w", err)
	}

	if userIDNullable.Valid {
		config.UserID = userIDNullable.String
	}
	if accountNameNullable.Valid {
		config.AccountName = accountNameNullable.String
	}
	if isDefaultNullable.Valid {
		config.IsDefault = isDefaultNullable.Bool
	}

	return &config, nil
}
