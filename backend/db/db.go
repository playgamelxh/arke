package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"arke/backend/config"
)

func Init(cfg config.Config) (*gorm.DB, error) {
	gormDSN := normalizeGORMDsn(cfg.DatabaseDSN)
	if err := waitForMySQL(gormDSN, 60*time.Second); err != nil {
		return nil, err
	}
	migrateDSN := normalizeMigrateDsn(cfg.DatabaseDSN)
	if err := runMigrations(migrateDSN, cfg.MigrationPath); err != nil {
		return nil, err
	}
	return gorm.Open(mysql.Open(gormDSN), &gorm.Config{})
}

func normalizeGORMDsn(dsn string) string {
	return strings.TrimPrefix(dsn, "mysql://")
}

func normalizeMigrateDsn(dsn string) string {
	if !strings.HasPrefix(dsn, "mysql://") {
		return "mysql://" + dsn
	}
	return dsn
}

func waitForMySQL(dsn string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		db, err := sql.Open("mysql", dsn)
		if err == nil {
			lastErr = db.Ping()
			_ = db.Close()
			if lastErr == nil {
				return nil
			}
		} else {
			lastErr = err
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("连接 MySQL 超时：%w", lastErr)
}

func runMigrations(databaseURL string, sourceURL string) error {
	log.Printf("running migrations: %s -> %s", sourceURL, databaseURL)
	m, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		return fmt.Errorf("初始化迁移失败：%w", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("执行迁移失败：%w", err)
	}
	log.Println("migrations completed")
	return nil
}