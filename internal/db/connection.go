package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds database configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// Connection wraps the database connection pool
type Connection struct {
	Pool *pgxpool.Pool
}

// NewConnection creates a new database connection
func NewConnection(ctx context.Context, config Config) (*Connection, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		var ltreeOID uint32
		err := conn.QueryRow(ctx, "select oid from pg_type where typname = 'ltree'").Scan(&ltreeOID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// Extension not installed yet; skip registration so connection still succeeds.
				return nil
			}
			return fmt.Errorf("failed to look up ltree type: %w", err)
		}

		ltreeType := &pgtype.Type{Name: "ltree", OID: ltreeOID, Codec: pgtype.LtreeCodec{}}
		conn.TypeMap().RegisterType(ltreeType)

		var ltreeArrayOID uint32
		err = conn.QueryRow(ctx, "select oid from pg_type where typname = '_ltree'").Scan(&ltreeArrayOID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return fmt.Errorf("failed to look up _ltree type: %w", err)
		}

		conn.TypeMap().RegisterType(&pgtype.Type{
			Name:  "_ltree",
			OID:   ltreeArrayOID,
			Codec: &pgtype.ArrayCodec{ElementType: ltreeType},
		})

		return nil
	}

	// Configure pool settings - more conservative to avoid connection issues
	poolConfig.MaxConns = 5
	poolConfig.MinConns = 1
	poolConfig.MaxConnLifetime = time.Minute * 30
	poolConfig.MaxConnIdleTime = time.Minute * 5
	poolConfig.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Connection{Pool: pool}, nil
}

// Close closes the database connection pool
func (c *Connection) Close() {
	if c.Pool != nil {
		c.Pool.Close()
	}
}

// WithTx executes a function within a database transaction
func (c *Connection) WithTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := c.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			if err := tx.Rollback(ctx); err != nil {
				log.Printf("Failed to rollback transaction: %v", err)
			}
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("transaction error: %v, rollback error: %v", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DefaultConfig returns a default database configuration
func DefaultConfig() Config {
	return Config{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "admin",
		DBName:   "engineering_api",
		SSLMode:  "disable",
	}
}
