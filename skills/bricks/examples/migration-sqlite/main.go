package main

import (
	"database/sql"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"

	_ "github.com/mattn/go-sqlite3" // Register the caller's SQLite driver.

	"github.com/cristiano-pacheco/bricks/pkg/migration"
)

//go:embed migrations/*.sql
var embedded embed.FS

func run(dsn string, down bool) (err error) {
	root, err := fs.Sub(embedded, "migrations")

	if err != nil {
		return err
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, db.Close())
	}()

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		return err
	}

	runner := migration.NewRunner([]migration.FileSystem{migration.New(root)})
	if down {
		return runner.DownSQLite(db)
	}

	return runner.UpSQLite(db)
}

func main() {
	dsn := flag.String("dsn", "file:migration-example.db?_foreign_keys=on&_busy_timeout=5000", "SQLite DSN")
	down := flag.Bool("down", false, "explicitly undo all migrations on a disposable development database")

	flag.Parse()

	if err := run(*dsn, *down); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
