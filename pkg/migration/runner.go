package migration

import (
	"errors"
	"fmt"
	"io/fs"
	"testing/fstest"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // postgres driver for migrate
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

type Runner struct {
	filesystems []FileSystem
}

func NewRunner(filesystems []FileSystem) *Runner {
	return &Runner{filesystems: filesystems}
}

// Up runs all pending migrations. Returns nil on ErrNoChange.
func (r *Runner) Up(dsn string) error {
	merged, err := r.merge()
	if err != nil {
		return fmt.Errorf("merge migration filesystems: %w", err)
	}
	src, err := iofs.New(merged, "migrations")
	if err != nil {
		return fmt.Errorf("create iofs source: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, dsn)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}
	defer m.Close()
	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

// merge combines all FileSystems into a single FS rooted at "migrations/".
// Files are keyed "migrations/<filename>" so iofs.New(merged, "migrations") works.
// Lexicographic sort of timestamps ensures correct global ordering.
func (r *Runner) merge() (fs.FS, error) {
	merged := make(fstest.MapFS)
	for _, fileSystem := range r.filesystems {
		entries, err := fs.ReadDir(fileSystem.FS, ".")
		if err != nil {
			return nil, fmt.Errorf("read migration dir: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			var data []byte
			data, err = fs.ReadFile(fileSystem.FS, entry.Name())
			if err != nil {
				return nil, fmt.Errorf("read %q: %w", entry.Name(), err)
			}
			merged["migrations/"+entry.Name()] = &fstest.MapFile{Data: data}
		}
	}
	return merged, nil
}
