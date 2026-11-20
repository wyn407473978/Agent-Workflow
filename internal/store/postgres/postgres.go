package postgres

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"

	"Agent-Workflow/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(ctx context.Context, cfg config.Postgres) (*gorm.DB, error) {
	dsn := url.URL{
		Scheme:   "postgres",
		Host:     net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		User:     url.UserPassword(cfg.User, cfg.Password),
		Path:     "/" + cfg.Database,
		RawPath:  "/" + url.PathEscape(cfg.Database),
		RawQuery: url.Values{"sslmode": {cfg.SSLMode}}.Encode(),
	}

	db, err := gorm.Open(postgres.Open(dsn.String()), &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		// Driver configuration errors can include the connection string.
		return nil, fmt.Errorf("initialize PostgreSQL driver: check connection settings")
	}
	pool, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get PostgreSQL connection pool: %w", err)
	}
	if err := pool.PingContext(ctx); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}
	return db, nil
}
