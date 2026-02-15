//go:build integration

package itestkit_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/cristiano-pacheco/bricks/pkg/itestkit"
	"github.com/stretchr/testify/suite"
)

type ITestKitIntegrationSuite struct {
	suite.Suite
}

func TestITestKitIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ITestKitIntegrationSuite))
}

func (s *ITestKitIntegrationSuite) TestRunMigrations() {
	// Arrange
	kit := s.setupTestKit()
	s.Require().NoError(kit.StartPostgres())
	s.T().Cleanup(kit.StopPostgres)

	// Act
	s.Require().NoError(kit.RunMigrations())

	// Assert
	type existsResult struct {
		Exists bool
	}
	var result existsResult
	s.Require().NoError(
		kit.DB().Raw("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'users') AS exists").Scan(&result).Error,
	)
	s.True(result.Exists)
}

func (s *ITestKitIntegrationSuite) TestPostgresInsertAndSelect() {
	// Arrange
	kit := s.setupTestKit()
	s.Require().NoError(kit.StartPostgres())
	s.T().Cleanup(kit.StopPostgres)
	s.Require().NoError(kit.RunMigrations())
	insertedName := "alice"

	// Act
	s.Require().NoError(kit.DB().Exec("INSERT INTO users (name) VALUES (?)", insertedName).Error)

	type userRow struct {
		ID   int
		Name string
	}
	var user userRow
	s.Require().NoError(
		kit.DB().Raw("SELECT id, name FROM users WHERE name = ? LIMIT 1", insertedName).Scan(&user).Error,
	)

	// Assert
	s.Greater(user.ID, 0)
	s.Equal(insertedName, user.Name)
}

func (s *ITestKitIntegrationSuite) TestRedisSetAndGet() {
	// Arrange
	kit := s.setupTestKit()
	s.Require().NoError(kit.StartRedis())
	s.T().Cleanup(kit.StopRedis)
	ctx := context.Background()
	key := "itestkit:health"
	expected := "ok"

	// Act
	s.Require().NoError(kit.Redis().Set(ctx, key, expected, 0).Err())
	value, err := kit.Redis().Get(ctx, key).Result()

	// Assert
	s.Require().NoError(err)
	s.Equal(expected, value)
}

func (s *ITestKitIntegrationSuite) TestPostgresStartAndStopLifecycle() {
	// Arrange
	kit := s.setupTestKit()
	s.Require().NoError(kit.StartPostgres())
	s.Require().NoError(kit.RunMigrations())

	type healthResult struct {
		Ready int
	}

	// Act
	var health healthResult
	s.Require().NoError(kit.DB().Raw("SELECT 1 AS ready").Scan(&health).Error)
	kit.StopPostgres()

	// Assert
	s.Equal(1, health.Ready)
	s.Require().NotNil(kit.DB())
	sqlDB, err := kit.DB().DB()
	s.Require().NoError(err)
	s.Error(sqlDB.PingContext(context.Background()))
}

func (s *ITestKitIntegrationSuite) TestRedisStartAndStopLifecycle() {
	// Arrange
	kit := s.setupTestKit()
	s.Require().NoError(kit.StartRedis())
	ctx := context.Background()

	// Act
	pong, err := kit.Redis().Ping(ctx).Result()
	kit.StopRedis()
	errAfterStop := kit.Redis().Ping(ctx).Err()

	// Assert
	s.Require().NoError(err)
	s.Equal("PONG", pong)
	s.Error(errAfterStop)
}

func (s *ITestKitIntegrationSuite) setupTestKit() *itestkit.ITestKit {
	s.ensureDocker()
	migrationsDir := s.migrationsPath()

	return itestkit.New(itestkit.Config{
		PostgresImage:  itestkit.DefaultConfig().PostgresImage,
		RedisImage:     itestkit.DefaultConfig().RedisImage,
		MigrationsPath: "file://" + migrationsDir,
		Database:       "itest_integration",
		User:           "itest",
		Password:       "itest",
	})
}

func (s *ITestKitIntegrationSuite) ensureDocker() {
	_, err := exec.LookPath("docker")
	s.Require().NoError(err, "docker CLI must be installed for integration tests")
	s.Require().NoError(exec.Command("docker", "info").Run(), "docker daemon must be available for integration tests")
}

func (s *ITestKitIntegrationSuite) migrationsPath() string {
	_, filename, _, ok := runtime.Caller(0)
	s.Require().True(ok)
	testDir := filepath.Dir(filename)
	migrationsDir := filepath.Join(testDir, "migrations")
	_, err := os.Stat(filepath.Join(migrationsDir, "000001_create_users.up.sql"))
	s.Require().NoError(err)

	return migrationsDir
}
