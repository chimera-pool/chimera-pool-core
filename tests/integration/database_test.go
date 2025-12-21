//go:build integration
// +build integration

package integration

import (
	"database/sql"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/lib/pq"
)

func TestDatabaseIntegration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping database integration test")
	}

	// Connect to database
	db, err := sql.Open("postgres", databaseURL)
	require.NoError(t, err)
	defer db.Close()

	// Test connection
	err = db.Ping()
	require.NoError(t, err, "Should be able to connect to database")

	// Test basic operations
	t.Run("CreateTable", func(t *testing.T) {
		_, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS test_table (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100) NOT NULL,
				created_at TIMESTAMP DEFAULT NOW()
			)
		`)
		assert.NoError(t, err)
	})

	t.Run("InsertAndSelect", func(t *testing.T) {
		// Insert test data
		var id int
		err := db.QueryRow(
			"INSERT INTO test_table (name) VALUES ($1) RETURNING id",
			"integration_test",
		).Scan(&id)
		assert.NoError(t, err)
		assert.Greater(t, id, 0)

		// Select test data
		var name string
		err = db.QueryRow(
			"SELECT name FROM test_table WHERE id = $1",
			id,
		).Scan(&name)
		assert.NoError(t, err)
		assert.Equal(t, "integration_test", name)
	})

	t.Run("Cleanup", func(t *testing.T) {
		_, err := db.Exec("DROP TABLE IF EXISTS test_table")
		assert.NoError(t, err)
	})
}

func main() {
	// This allows the test to be run as a standalone program
	// for integration testing in Docker
	testing.Main(func(pat, str string) (bool, error) { return true, nil },
		[]testing.InternalTest{
			{
				Name: "TestDatabaseIntegration",
				F:    TestDatabaseIntegration,
			},
		},
		[]testing.InternalBenchmark{},
		[]testing.InternalExample{},
	)
}
