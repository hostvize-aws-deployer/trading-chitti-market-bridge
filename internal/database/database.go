package database

import (
	"database/sql"
	"time"
	
	_ "github.com/lib/pq"
	"github.com/trading-chitti/market-bridge/internal/broker"
)

// Database handles PostgreSQL operations
type Database struct {
	conn *sql.DB
}

// NewDatabase creates a new database connection
func NewDatabase(dsn string) (*Database, error) {
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	
	if err := conn.Ping(); err != nil {
		return nil, err
	}
	
	return &Database{conn: conn}, nil
}

// Close closes the database connection
func (db *Database) Close() error {
	return db.conn.Close()
}

// GetActiveBrokerConfig returns the active broker configuration
func (db *Database) GetActiveBrokerConfig() (*broker.BrokerConfig, error) {
	query := `
		SELECT id, broker_name, display_name, enabled, api_key, api_secret, 
		       access_token, user_id, max_positions, max_risk_per_trade, 
		       created_at, updated_at
		FROM brokers.config
		WHERE enabled = true
		ORDER BY updated_at DESC
		LIMIT 1
	`
	
	config := &broker.BrokerConfig{}
	err := db.conn.QueryRow(query).Scan(
		&config.ID,
		&config.BrokerName,
		&config.DisplayName,
		&config.Enabled,
		&config.APIKey,
		&config.APISecret,
		&config.AccessToken,
		&config.UserID,
		&config.MaxPositions,
		&config.MaxRiskPerTrade,
		&config.CreatedAt,
		&config.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	
	if err != nil {
		return nil, err
	}
	
	return config, nil
}

// GetAllBrokerConfigs returns all broker configurations
func (db *Database) GetAllBrokerConfigs() ([]broker.BrokerConfig, error) {
	query := `
		SELECT id, broker_name, display_name, enabled, api_key, api_secret,
		       access_token, user_id, max_positions, max_risk_per_trade,
		       created_at, updated_at
		FROM brokers.config
		ORDER BY created_at DESC
	`
	
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	configs := []broker.BrokerConfig{}
	for rows.Next() {
		config := broker.BrokerConfig{}
		err := rows.Scan(
			&config.ID,
			&config.BrokerName,
			&config.DisplayName,
			&config.Enabled,
			&config.APIKey,
			&config.APISecret,
			&config.AccessToken,
			&config.UserID,
			&config.MaxPositions,
			&config.MaxRiskPerTrade,
			&config.CreatedAt,
			&config.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}
	
	return configs, nil
}

// SaveBrokerConfig saves a broker configuration
func (db *Database) SaveBrokerConfig(config *broker.BrokerConfig) (int, error) {
	query := `
		INSERT INTO brokers.config (
			broker_name, display_name, enabled, api_key, api_secret,
			access_token, user_id, max_positions, max_risk_per_trade
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	
	var id int
	err := db.conn.QueryRow(
		query,
		config.BrokerName,
		config.DisplayName,
		config.Enabled,
		config.APIKey,
		config.APISecret,
		config.AccessToken,
		config.UserID,
		config.MaxPositions,
		config.MaxRiskPerTrade,
	).Scan(&id)
	
	return id, err
}

// SaveAnalysis saves analysis results
func (db *Database) SaveAnalysis(symbol string, analysis interface{}) error {
	query := `
		INSERT INTO trades.analysis (
			symbol, analysis_date, period_days, analysis_json
		) VALUES ($1, $2, $3, $4)
	`
	
	_, err := db.conn.Exec(query, symbol, time.Now(), 52, analysis)
	return err
}

// SaveTrade saves a trade execution
func (db *Database) SaveTrade(trade interface{}) error {
	// TODO: Implement trade saving
	return nil
}
