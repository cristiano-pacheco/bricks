package migration_test

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/cristiano-pacheco/bricks/pkg/database/sqlite"
	"github.com/cristiano-pacheco/bricks/pkg/migration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type migrationFile struct {
	name string
	sql  string
}

func TestRunner_UpSQLite_EmptyFilesystem_Succeeds(t *testing.T) {
	// Arrange
	db := newSQLiteDatabase(t)
	runner := migration.NewRunner(nil)

	// Act
	err := runner.UpSQLite(db)

	// Assert
	require.NoError(t, err)
	_, exists := schemaVersion(t, db)
	assert.False(t, exists)
}

func TestRunner_UpSQLite_OrdersFilesystemContributionsDeterministically(t *testing.T) {
	// Arrange
	db := newSQLiteDatabase(t)
	first := newMigrationFileSystem(
		migrationFile{
			name: "000001_create_migration_order.up.sql",
			sql:  "CREATE TABLE migration_order (step INTEGER NOT NULL);",
		},
	)
	second := newMigrationFileSystem(
		migrationFile{
			name: "000002_insert_second.up.sql",
			sql:  "INSERT INTO migration_order (step) VALUES (2);",
		},
		migrationFile{
			name: "000003_insert_third.up.sql",
			sql:  "INSERT INTO migration_order (step) VALUES (3);",
		},
	)
	runner := migration.NewRunner([]migration.FileSystem{second, first})

	// Act
	err := runner.UpSQLite(db)

	// Assert
	require.NoError(t, err)
	rows, err := db.Query("SELECT step FROM migration_order ORDER BY step")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, rows.Close()) })
	var steps []int
	for rows.Next() {
		var step int
		require.NoError(t, rows.Scan(&step))
		steps = append(steps, step)
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, []int{2, 3}, steps)
	version, exists := schemaVersion(t, db)
	require.True(t, exists)
	assert.Equal(t, uint64(3), version)
}

func TestRunner_UpSQLite_OlderSchemaAppliesAllPendingMigrations(t *testing.T) {
	// Arrange
	db := newSQLiteDatabase(t)
	initial := newMigrationFileSystem(
		migrationFile{
			name: "000001_create_accounts.up.sql",
			sql:  "CREATE TABLE accounts (id INTEGER PRIMARY KEY);",
		},
	)
	require.NoError(t, migration.NewRunner([]migration.FileSystem{initial}).UpSQLite(db))
	full := newMigrationFileSystem(
		migrationFile{
			name: "000001_create_accounts.up.sql",
			sql:  "CREATE TABLE accounts (id INTEGER PRIMARY KEY);",
		},
		migrationFile{
			name: "000001_create_accounts.down.sql",
			sql:  "DROP TABLE accounts;",
		},
		migrationFile{
			name: "000002_add_name.up.sql",
			sql:  "ALTER TABLE accounts ADD COLUMN name TEXT NOT NULL DEFAULT '';",
		},
		migrationFile{
			name: "000002_add_name.down.sql",
			sql:  "ALTER TABLE accounts DROP COLUMN name;",
		},
		migrationFile{
			name: "000003_seed_account.up.sql",
			sql:  "INSERT INTO accounts (name) VALUES ('alice');",
		},
		migrationFile{
			name: "000003_seed_account.down.sql",
			sql:  "DELETE FROM accounts WHERE name = 'alice';",
		},
	)

	// Act
	err := migration.NewRunner([]migration.FileSystem{full}).UpSQLite(db)

	// Assert
	require.NoError(t, err)
	version, exists := schemaVersion(t, db)
	require.True(t, exists)
	assert.Equal(t, uint64(3), version)
	var name string
	require.NoError(t, db.QueryRow("SELECT name FROM accounts").Scan(&name))
	assert.Equal(t, "alice", name)
}

func TestRunner_UpSQLite_CurrentSchemaIsNoOp(t *testing.T) {
	// Arrange
	db := newSQLiteDatabase(t)
	runner := migration.NewRunner([]migration.FileSystem{newMigrationFileSystem(
		migrationFile{
			name: "000001_create_events.up.sql",
			sql:  "CREATE TABLE events (id INTEGER PRIMARY KEY);",
		},
		migrationFile{
			name: "000001_create_events.down.sql",
			sql:  "DROP TABLE events;",
		},
		migrationFile{
			name: "000002_seed_event.up.sql",
			sql:  "INSERT INTO events DEFAULT VALUES;",
		},
		migrationFile{
			name: "000002_seed_event.down.sql",
			sql:  "DELETE FROM events;",
		},
	)})
	require.NoError(t, runner.UpSQLite(db))

	// Act
	err := runner.UpSQLite(db)

	// Assert
	require.NoError(t, err)
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM events").Scan(&count))
	assert.Equal(t, 1, count)
	version, exists := schemaVersion(t, db)
	require.True(t, exists)
	assert.Equal(t, uint64(2), version)
}

func TestRunner_UpSQLite_FailedMigrationRollsBackPendingSchema(t *testing.T) {
	// Arrange
	db := newSQLiteDatabase(t)
	initial := newMigrationFileSystem(
		migrationFile{
			name: "000001_create_base.up.sql",
			sql:  "CREATE TABLE base (id INTEGER PRIMARY KEY);",
		},
		migrationFile{
			name: "000001_create_base.down.sql",
			sql:  "DROP TABLE base;",
		},
	)
	require.NoError(t, migration.NewRunner([]migration.FileSystem{initial}).UpSQLite(db))
	failing := newMigrationFileSystem(
		migrationFile{
			name: "000001_create_base.up.sql",
			sql:  "CREATE TABLE base (id INTEGER PRIMARY KEY);",
		},
		migrationFile{
			name: "000001_create_base.down.sql",
			sql:  "DROP TABLE base;",
		},
		migrationFile{
			name: "000002_create_transient.up.sql",
			sql:  "CREATE TABLE transient_data (value TEXT NOT NULL);",
		},
		migrationFile{
			name: "000002_create_transient.down.sql",
			sql:  "DROP TABLE transient_data;",
		},
		migrationFile{
			name: "000003_fail.up.sql",
			sql:  "THIS IS NOT VALID SQLITE;",
		},
		migrationFile{
			name: "000003_fail.down.sql",
			sql:  "SELECT 1;",
		},
	)

	// Act
	err := migration.NewRunner([]migration.FileSystem{failing}).UpSQLite(db)

	// Assert
	require.Error(t, err)
	assert.True(t, tableExists(t, db, "base"))
	assert.False(t, tableExists(t, db, "transient_data"))
	version, exists := schemaVersion(t, db)
	require.True(t, exists)
	assert.Equal(t, uint64(1), version)

	// Act
	require.NoError(t, migration.NewRunner([]migration.FileSystem{newMigrationFileSystem(
		migrationFile{
			name: "000001_create_base.up.sql",
			sql:  "CREATE TABLE base (id INTEGER PRIMARY KEY);",
		},
		migrationFile{
			name: "000001_create_base.down.sql",
			sql:  "DROP TABLE base;",
		},
		migrationFile{
			name: "000002_create_transient.up.sql",
			sql:  "CREATE TABLE transient_data (value TEXT NOT NULL);",
		},
		migrationFile{
			name: "000002_create_transient.down.sql",
			sql:  "DROP TABLE transient_data;",
		},
	)}).UpSQLite(db))

	// Assert
	assert.True(t, tableExists(t, db, "transient_data"))
	version, exists = schemaVersion(t, db)
	require.True(t, exists)
	assert.Equal(t, uint64(2), version)
}

func TestRunner_UpSQLite_RejectsNewerSchemaBeforeRunningMigrations(t *testing.T) {
	// Arrange
	db := newSQLiteDatabase(t)
	_, err := db.Exec("CREATE TABLE schema_migrations (version uint64, dirty bool)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO schema_migrations (version, dirty) VALUES (99, false)")
	require.NoError(t, err)
	runner := migration.NewRunner([]migration.FileSystem{newMigrationFileSystem(
		migrationFile{
			name: "000001_must_not_run.up.sql",
			sql:  "CREATE TABLE must_not_run (id INTEGER);",
		},
	)})

	// Act
	err = runner.UpSQLite(db)

	// Assert
	require.ErrorIs(t, err, migration.ErrDatabaseNewer)
	assert.False(t, tableExists(t, db, "must_not_run"))
	version, exists := schemaVersion(t, db)
	require.True(t, exists)
	assert.Equal(t, uint64(99), version)
}

func TestRunner_DownSQLite_AppliesMatchingMigrationsInReverseOrder(t *testing.T) {
	// Arrange
	db := newSQLiteDatabase(t)
	runner := migration.NewRunner([]migration.FileSystem{newMigrationFileSystem(
		migrationFile{
			name: "000001_create_parent.up.sql",
			sql:  "CREATE TABLE migration_parent (id INTEGER PRIMARY KEY);",
		},
		migrationFile{
			name: "000001_create_parent.down.sql",
			sql:  "DROP TABLE migration_parent;",
		},
		migrationFile{
			name: "000002_create_child.up.sql",
			sql:  "CREATE TABLE migration_child (id INTEGER PRIMARY KEY, parent_id INTEGER REFERENCES migration_parent(id));",
		},
		migrationFile{
			name: "000002_create_child.down.sql",
			sql:  "DROP TABLE migration_child;",
		},
	)})
	require.NoError(t, runner.UpSQLite(db))

	// Act
	err := runner.DownSQLite(db)

	// Assert
	require.NoError(t, err)
	assert.False(t, tableExists(t, db, "migration_child"))
	assert.False(t, tableExists(t, db, "migration_parent"))
	_, exists := schemaVersion(t, db)
	assert.False(t, exists)
}

func TestRunner_UpSQLite_RejectsDuplicateMigrationVersions(t *testing.T) {
	// Arrange
	db := newSQLiteDatabase(t)
	runner := migration.NewRunner([]migration.FileSystem{
		newMigrationFileSystem(migrationFile{
			name: "000001_create_one.up.sql",
			sql:  "CREATE TABLE one (id INTEGER);",
		}),
		newMigrationFileSystem(migrationFile{
			name: "000001_create_other.up.sql",
			sql:  "CREATE TABLE other (id INTEGER);",
		}),
	})

	// Act
	err := runner.UpSQLite(db)

	// Assert
	require.ErrorIs(t, err, migration.ErrInvalidMigration)
	assert.False(t, tableExists(t, db, "one"))
	assert.False(t, tableExists(t, db, "other"))
}

func newSQLiteDatabase(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "migration.db")
	gormDB, err := sqlite.New(sqlite.Config{DSN: path})
	require.NoError(t, err)
	db, err := gormDB.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}

func newMigrationFileSystem(files ...migrationFile) migration.FileSystem {
	fileSystem := make(fstest.MapFS, len(files))
	for _, file := range files {
		fileSystem[file.name] = &fstest.MapFile{Data: []byte(file.sql)}
	}
	return migration.New(fileSystem)
}

func schemaVersion(t *testing.T, db *sql.DB) (uint64, bool) {
	t.Helper()
	var version uint64
	var dirty bool
	err := db.QueryRow("SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false
	}
	require.NoError(t, err)
	assert.False(t, dirty)
	return version, true
}

func tableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?",
		table,
	).Scan(&count)
	require.NoError(t, err)
	return count == 1
}
