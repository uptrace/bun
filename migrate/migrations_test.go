package migrate

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestDiscoverDuplicateMigrationID(t *testing.T) {
	fsys := fstest.MapFS{
		"20260228140000_foo.tx.up.sql":   &fstest.MapFile{Data: []byte("SELECT 1")},
		"20260228140000_bar.tx.up.sql":   &fstest.MapFile{Data: []byte("SELECT 2")},
		"20260228150000_baz.tx.up.sql":   &fstest.MapFile{Data: []byte("SELECT 3")},
		"20260228150000_baz.tx.down.sql": &fstest.MapFile{Data: []byte("SELECT 4")},
	}

	migrations := NewMigrations()
	err := migrations.Discover(fsys)
	if err == nil {
		t.Fatal("expected error for duplicate migration ID")
	}
	if got := err.Error(); !strings.Contains(got, "duplicate migration ID") || !strings.Contains(got, "20260228140000") {
		t.Fatalf("unexpected error: %s", got)
	}
}

func TestDiscoverDuplicateDownMigration(t *testing.T) {
	fsys := fstest.MapFS{
		"20260228140000_foo.tx.down.sql": &fstest.MapFile{Data: []byte("SELECT 1")},
		"20260228140000_bar.tx.down.sql": &fstest.MapFile{Data: []byte("SELECT 2")},
	}

	migrations := NewMigrations()
	err := migrations.Discover(fsys)
	if err == nil {
		t.Fatal("expected error for duplicate migration ID")
	}
	if got := err.Error(); !strings.Contains(got, "duplicate migration ID") {
		t.Fatalf("unexpected error: %s", got)
	}
}

func TestDiscoverDuplicateMixedUpDown(t *testing.T) {
	fsys := fstest.MapFS{
		"20260228140000_foo.tx.up.sql":   &fstest.MapFile{Data: []byte("SELECT 1")},
		"20260228140000_bar.tx.down.sql": &fstest.MapFile{Data: []byte("SELECT 2")},
	}

	migrations := NewMigrations()
	err := migrations.Discover(fsys)
	if err == nil {
		t.Fatal("expected error for duplicate migration ID")
	}
	if got := err.Error(); !strings.Contains(got, "duplicate migration ID") {
		t.Fatalf("unexpected error: %s", got)
	}
}

func TestDiscoverNoDuplicate(t *testing.T) {
	fsys := fstest.MapFS{
		"20260228140000_foo.tx.up.sql":   &fstest.MapFile{Data: []byte("SELECT 1")},
		"20260228140000_foo.tx.down.sql": &fstest.MapFile{Data: []byte("SELECT 2")},
		"20260228150000_bar.tx.up.sql":   &fstest.MapFile{Data: []byte("SELECT 3")},
	}

	migrations := NewMigrations()
	err := migrations.Discover(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

