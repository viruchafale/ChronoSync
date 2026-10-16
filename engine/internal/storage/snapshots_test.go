package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestUpgradeFromPhaseTwo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	dbpath := filepath.Join(path, "history.sqlite")
	f, err := os.OpenFile(dbpath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	db, err := sql.Open("sqlite", dbpath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY,checksum TEXT NOT NULL);` + migration); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte(migration))
	if _, err = db.Exec(`INSERT INTO schema_migrations VALUES(1,?)`, hex.EncodeToString(hash[:])); err != nil {
		t.Fatal(err)
	}
	db.Close()
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var n int
	if err = s.db.QueryRow(`SELECT count(*) FROM schema_migrations`).Scan(&n); err != nil || n != 3 {
		t.Fatal(n, err)
	}
	if err = s.RecoverSnapshots(context.Background()); err != nil {
		t.Fatal(err)
	}
}
