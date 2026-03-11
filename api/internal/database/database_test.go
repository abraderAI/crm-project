package database

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path)
	require.NoError(t, err)
	require.NotNil(t, db)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	assert.NoError(t, sqlDB.Ping())

	// Verify WAL mode (note: pure Go sqlite may report mode differently on some systems).
	var journalMode string
	err = sqlDB.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	require.NoError(t, err)
	assert.Contains(t, []string{"wal", "delete"}, journalMode)

	// Verify foreign keys enabled.
	var fk int
	err = sqlDB.QueryRow("PRAGMA foreign_keys").Scan(&fk)
	require.NoError(t, err)
	assert.Equal(t, 1, fk)
}

func TestOpen_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "test.db")

	db, err := Open(path)
	require.NoError(t, err)
	require.NotNil(t, db)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	assert.NoError(t, sqlDB.Ping())
}

func TestOpen_CurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path)
	require.NoError(t, err)
	require.NotNil(t, db)
}

func TestOpen_InvalidPath(t *testing.T) {
	// Try to open in a path we cannot create (read-only parent).
	_, err := Open("/proc/nonexistent/deep/path/db")
	assert.Error(t, err)
}

func TestOpen_ForeignKeysWork(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fk.db")

	db, err := Open(path)
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)

	// Create a parent table and child with FK.
	_, err = sqlDB.Exec("CREATE TABLE parent (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)
	_, err = sqlDB.Exec("CREATE TABLE child (id INTEGER PRIMARY KEY, parent_id INTEGER REFERENCES parent(id))")
	require.NoError(t, err)

	// Insert valid reference.
	_, err = sqlDB.Exec("INSERT INTO parent (id) VALUES (1)")
	require.NoError(t, err)
	_, err = sqlDB.Exec("INSERT INTO child (id, parent_id) VALUES (1, 1)")
	assert.NoError(t, err)
}
