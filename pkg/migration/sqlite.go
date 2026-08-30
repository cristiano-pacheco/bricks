package migration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"sort"

	"github.com/golang-migrate/migrate/v4/source"
)

const sqliteMigrationsTable = "schema_migrations"

var (
	// ErrDatabaseNewer indicates that the database contains a schema version that
	// is not available in the running binary.
	ErrDatabaseNewer = errors.New("database schema is newer than available migrations")

	// ErrDatabaseDirty indicates that the database has an unresolved dirty state.
	ErrDatabaseDirty = errors.New("database schema is dirty")

	// ErrInvalidMigration indicates that a migration contribution is malformed or
	// contains a duplicate version and direction.
	ErrInvalidMigration = errors.New("invalid migration")

	// ErrMissingDownMigration indicates that a requested development downgrade
	// has no matching down migration.
	ErrMissingDownMigration = errors.New("missing down migration")

	// ErrInvalidSchemaVersion indicates that the schema migration metadata is
	// malformed or contains more than one current version.
	ErrInvalidSchemaVersion = errors.New("invalid schema migration version")

	// ErrNilDatabase indicates that a SQLite migration was given no database.
	ErrNilDatabase = errors.New("sqlite database is nil")
)

type sqliteMigration struct {
	name      string
	version   uint
	direction source.Direction
	contents  []byte
}

type sqliteSchemaVersion struct {
	version uint
	exists  bool
	dirty   bool
}

// UpSQLite applies all pending SQLite up migrations in one transaction.
//
// The database is supplied by the caller and remains open after this method
// returns. Migration files are read before the transaction starts, and the
// transaction is committed only after every pending migration and version
// update succeeds.
func (r *Runner) UpSQLite(db *sql.DB) error {
	return r.runSQLiteMigrations(db, "up", applySQLiteUpMigrations)
}

// DownSQLite applies all matching SQLite down migrations in descending order
// in one transaction. It is intended for explicit development use; UpSQLite
// never invokes it and no automatic runtime downgrade is performed.
//
// The database is supplied by the caller and remains open after this method
// returns.
func (r *Runner) DownSQLite(db *sql.DB) error {
	return r.runSQLiteMigrations(db, "down", applySQLiteDownMigrations)
}

func (r *Runner) runSQLiteMigrations(
	db *sql.DB,
	direction string,
	apply func(*sql.Tx, []sqliteMigration) error,
) error {
	if db == nil {
		return ErrNilDatabase
	}

	migrations, loadErr := r.readSQLiteMigrations()
	if loadErr != nil {
		return fmt.Errorf("load sqlite migrations: %w", loadErr)
	}

	runErr := withSQLiteMigrationTransaction(context.Background(), db, func(tx *sql.Tx) error {
		return apply(tx, migrations)
	})
	if runErr != nil {
		return fmt.Errorf("run sqlite %s migrations: %w", direction, runErr)
	}
	return nil
}

func applySQLiteUpMigrations(tx *sql.Tx, migrations []sqliteMigration) error {
	if ensureErr := ensureSQLiteMigrationsTable(tx); ensureErr != nil {
		return ensureErr
	}

	current, versionErr := readSQLiteSchemaVersion(tx)
	if versionErr != nil {
		return versionErr
	}
	if current.dirty {
		return fmt.Errorf("%w: version %d", ErrDatabaseDirty, current.version)
	}

	latest := latestSQLiteUpVersion(migrations)
	if current.exists && current.version > latest {
		return fmt.Errorf(
			"%w: database version %d, latest available version %d",
			ErrDatabaseNewer,
			current.version,
			latest,
		)
	}

	for _, migration := range migrations {
		if migration.direction != source.Up || (current.exists && migration.version <= current.version) {
			continue
		}
		if executeErr := executeSQLiteMigration(tx, migration); executeErr != nil {
			return executeErr
		}
		if recordErr := writeSQLiteSchemaVersion(tx, migration.version); recordErr != nil {
			return fmt.Errorf("record version %d: %w", migration.version, recordErr)
		}
	}

	return nil
}

func applySQLiteDownMigrations(tx *sql.Tx, migrations []sqliteMigration) error {
	if ensureErr := ensureSQLiteMigrationsTable(tx); ensureErr != nil {
		return ensureErr
	}

	current, versionErr := readSQLiteSchemaVersion(tx)
	if versionErr != nil {
		return versionErr
	}
	if !current.exists {
		return nil
	}
	if current.dirty {
		return fmt.Errorf("%w: version %d", ErrDatabaseDirty, current.version)
	}

	latest := latestSQLiteUpVersion(migrations)
	if current.version > latest {
		return fmt.Errorf(
			"%w: database version %d, latest available version %d",
			ErrDatabaseNewer,
			current.version,
			latest,
		)
	}

	downMigrations, downErr := matchingSQLiteDownMigrations(migrations, current.version)
	if downErr != nil {
		return downErr
	}
	for index, migration := range downMigrations {
		if executeErr := executeSQLiteMigration(tx, migration); executeErr != nil {
			return executeErr
		}
		if index == len(downMigrations)-1 {
			if clearErr := clearSQLiteSchemaVersion(tx); clearErr != nil {
				return fmt.Errorf("clear schema version: %w", clearErr)
			}
			continue
		}
		previousVersion := downMigrations[index+1].version
		if recordErr := writeSQLiteSchemaVersion(tx, previousVersion); recordErr != nil {
			return fmt.Errorf("record version %d: %w", previousVersion, recordErr)
		}
	}

	return nil
}

func (r *Runner) readSQLiteMigrations() ([]sqliteMigration, error) {
	migrations := make([]sqliteMigration, 0)
	seen := make(map[sqliteMigrationKey]string)

	for fileSystemIndex, fileSystem := range r.filesystems {
		if fileSystem.FS == nil {
			return nil, fmt.Errorf("%w: filesystem %d is nil", ErrInvalidMigration, fileSystemIndex)
		}

		fileSystemMigrations, readErr := readSQLiteFilesystem(fileSystemIndex, fileSystem.FS)
		if readErr != nil {
			return nil, readErr
		}
		for _, migration := range fileSystemMigrations {
			key := sqliteMigrationKey{version: migration.version, direction: migration.direction}
			if previousName, exists := seen[key]; exists {
				return nil, fmt.Errorf(
					"%w: version %d %s is declared by both %q and %q",
					ErrInvalidMigration,
					migration.version,
					migration.direction,
					previousName,
					migration.name,
				)
			}
			seen[key] = migration.name
			migrations = append(migrations, migration)
		}
	}

	sort.Slice(migrations, func(i, j int) bool {
		if migrations[i].version != migrations[j].version {
			return migrations[i].version < migrations[j].version
		}
		if migrations[i].direction != migrations[j].direction {
			return migrations[i].direction < migrations[j].direction
		}
		return migrations[i].name < migrations[j].name
	})
	return migrations, nil
}

func readSQLiteFilesystem(fileSystemIndex int, fileSystem fs.FS) ([]sqliteMigration, error) {
	entries, readErr := fs.ReadDir(fileSystem, ".")
	if readErr != nil {
		return nil, fmt.Errorf("read filesystem %d: %w", fileSystemIndex, readErr)
	}

	migrations := make([]sqliteMigration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		parsed, parseErr := source.DefaultParse(name)
		if parseErr != nil {
			return nil, fmt.Errorf(
				"%w: %q does not match the migration filename convention",
				ErrInvalidMigration,
				name,
			)
		}

		contents, contentErr := fs.ReadFile(fileSystem, name)
		if contentErr != nil {
			return nil, fmt.Errorf("read migration %q: %w", name, contentErr)
		}
		migrations = append(migrations, sqliteMigration{
			name:      name,
			version:   parsed.Version,
			direction: parsed.Direction,
			contents:  contents,
		})
	}
	return migrations, nil
}

type sqliteMigrationKey struct {
	version   uint
	direction source.Direction
}

func latestSQLiteUpVersion(migrations []sqliteMigration) uint {
	var latest uint
	for _, migration := range migrations {
		if migration.direction == source.Up && migration.version > latest {
			latest = migration.version
		}
	}
	return latest
}

func matchingSQLiteDownMigrations(
	migrations []sqliteMigration,
	currentVersion uint,
) ([]sqliteMigration, error) {
	upVersions := make(map[uint]struct{})
	downMigrationsByVersion := make(map[uint]sqliteMigration)
	for _, migration := range migrations {
		switch migration.direction {
		case source.Up:
			if migration.version <= currentVersion {
				upVersions[migration.version] = struct{}{}
			}
		case source.Down:
			downMigrationsByVersion[migration.version] = migration
		}
	}

	versions := make([]uint, 0, len(upVersions))
	for version := range upVersions {
		if _, exists := downMigrationsByVersion[version]; !exists {
			return nil, fmt.Errorf("%w: version %d", ErrMissingDownMigration, version)
		}
		versions = append(versions, version)
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i] > versions[j] })

	result := make([]sqliteMigration, 0, len(versions))
	for _, version := range versions {
		result = append(result, downMigrationsByVersion[version])
	}
	return result, nil
}

func withSQLiteMigrationTransaction(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) (err error) {
	tx, beginErr := db.BeginTx(ctx, nil)
	if beginErr != nil {
		return fmt.Errorf("begin transaction: %w", beginErr)
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			_ = tx.Rollback()
			panic(recovered)
		}
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				err = errors.Join(err, fmt.Errorf("rollback transaction: %w", rollbackErr))
			}
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			err = fmt.Errorf("commit transaction: %w", commitErr)
		}
	}()

	err = fn(tx)
	return err
}

func ensureSQLiteMigrationsTable(tx *sql.Tx) error {
	const query = `
CREATE TABLE IF NOT EXISTS schema_migrations (version uint64, dirty bool);
CREATE UNIQUE INDEX IF NOT EXISTS schema_migrations_version_unique
    ON schema_migrations (version);`
	if _, execErr := tx.ExecContext(context.Background(), query); execErr != nil {
		return fmt.Errorf("create schema migrations table: %w", execErr)
	}
	return nil
}

func readSQLiteSchemaVersion(tx *sql.Tx) (sqliteSchemaVersion, error) {
	rows, queryErr := tx.QueryContext(context.Background(), "SELECT version, dirty FROM "+sqliteMigrationsTable)
	if queryErr != nil {
		return sqliteSchemaVersion{}, fmt.Errorf("read schema version: %w", queryErr)
	}
	defer rows.Close()

	if !rows.Next() {
		if rowsErr := rows.Err(); rowsErr != nil {
			return sqliteSchemaVersion{}, fmt.Errorf("read schema version: %w", rowsErr)
		}
		return sqliteSchemaVersion{}, nil
	}

	var version uint64
	var dirty bool
	if scanErr := rows.Scan(&version, &dirty); scanErr != nil {
		return sqliteSchemaVersion{}, fmt.Errorf("scan schema version: %w", scanErr)
	}
	if uint64(uint(version)) != version {
		return sqliteSchemaVersion{}, fmt.Errorf("%w: version %d overflows uint", ErrInvalidSchemaVersion, version)
	}
	if rows.Next() {
		return sqliteSchemaVersion{}, fmt.Errorf("%w: more than one current version", ErrInvalidSchemaVersion)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return sqliteSchemaVersion{}, fmt.Errorf("read schema version: %w", rowsErr)
	}

	return sqliteSchemaVersion{version: uint(version), exists: true, dirty: dirty}, nil
}

func executeSQLiteMigration(tx *sql.Tx, migration sqliteMigration) error {
	if _, execErr := tx.ExecContext(context.Background(), string(migration.contents)); execErr != nil {
		return fmt.Errorf(
			"run sqlite %s migration %d (%s): %w",
			migration.direction,
			migration.version,
			migration.name,
			execErr,
		)
	}
	return nil
}

func writeSQLiteSchemaVersion(tx *sql.Tx, version uint) error {
	if _, deleteErr := tx.ExecContext(context.Background(), "DELETE FROM "+sqliteMigrationsTable); deleteErr != nil {
		return fmt.Errorf("delete previous schema version: %w", deleteErr)
	}
	if _, insertErr := tx.ExecContext(
		context.Background(),
		"INSERT INTO "+sqliteMigrationsTable+" (version, dirty) VALUES (?, ?)",
		version,
		false,
	); insertErr != nil {
		return fmt.Errorf("insert schema version: %w", insertErr)
	}
	return nil
}

func clearSQLiteSchemaVersion(tx *sql.Tx) error {
	if _, deleteErr := tx.ExecContext(context.Background(), "DELETE FROM "+sqliteMigrationsTable); deleteErr != nil {
		return fmt.Errorf("delete schema version: %w", deleteErr)
	}
	return nil
}
