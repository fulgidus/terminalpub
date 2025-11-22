package db

import (
	"context"
	"fmt"
	"time"

	"github.com/fulgidus/terminalpub/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// DB holds database connections
type DB struct {
	Postgres *pgxpool.Pool
	Redis    *redis.Client
}

// Connect establishes connections to PostgreSQL and Redis
func Connect(cfg *config.Config) (*DB, error) {
	db := &DB{}

	// Connect to PostgreSQL
	pgConfig := cfg.Database.Postgres
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s pool_max_conns=%d",
		pgConfig.Host,
		pgConfig.Port,
		pgConfig.User,
		pgConfig.Password,
		pgConfig.Database,
		pgConfig.SSLMode,
		pgConfig.MaxConnections,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	db.Postgres = pool

	// Connect to Redis
	redisConfig := cfg.Database.Redis
	db.Redis = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisConfig.Host, redisConfig.Port),
		Password: redisConfig.Password,
		DB:       redisConfig.DB,
	})

	// Test Redis connection
	if err := db.Redis.Ping(ctx).Err(); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return db, nil
}

// Close closes all database connections
func (db *DB) Close() {
	if db.Postgres != nil {
		db.Postgres.Close()
	}
	if db.Redis != nil {
		db.Redis.Close()
	}
}

// Health checks database connections
func (db *DB) Health(ctx context.Context) error {
	// Check PostgreSQL
	if err := db.Postgres.Ping(ctx); err != nil {
		return fmt.Errorf("postgres unhealthy: %w", err)
	}

	// Check Redis
	if err := db.Redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis unhealthy: %w", err)
	}

	return nil
}
