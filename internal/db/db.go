package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/prakashniraula/portfolio-in-go/internal/config"
	_ "modernc.org/sqlite"
)

// Open opens a database connection.
// DB_DRIVER=sqlite (default) uses a local file.
// DB_DRIVER=libsql is reserved for Turso — set DATABASE_URL when ready.
func Open(cfg config.Config) (*sql.DB, error) {
	switch strings.ToLower(cfg.DBDriver) {
	case "sqlite", "":
		if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
			return nil, fmt.Errorf("create data dir: %w", err)
		}
		dsn := cfg.DBPath + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
		db, err := sql.Open("sqlite", dsn)
		if err != nil {
			return nil, err
		}
		db.SetMaxOpenConns(4)
		db.SetMaxIdleConns(4)
		if err := db.Ping(); err != nil {
			_ = db.Close()
			return nil, err
		}
		return db, nil
	case "libsql", "turso":
		// Turso / libSQL: swap in github.com/tursodatabase/libsql-client-go/libsql
		// and open with cfg.DatabaseURL when deploying remotely.
		if cfg.DatabaseURL == "" {
			return nil, fmt.Errorf("DB_DRIVER=%s requires DATABASE_URL", cfg.DBDriver)
		}
		return nil, fmt.Errorf("libsql/turso driver not compiled in yet; use DB_DRIVER=sqlite with a persistent volume, or add the libsql driver")
	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER: %s", cfg.DBDriver)
	}
}
