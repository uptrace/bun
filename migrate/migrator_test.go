package migrate

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func TestCreateMigrationFilesContent(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		create func(m *Migrator) ([]*MigrationFile, error)
		want   []string
	}{
		{
			name: "sql",
			create: func(m *Migrator) ([]*MigrationFile, error) {
				return m.CreateSQLMigrations(ctx, "add_users")
			},
			want: []string{sqlTemplate, sqlTemplate},
		},
		{
			name: "tx sql",
			create: func(m *Migrator) ([]*MigrationFile, error) {
				return m.CreateTxSQLMigrations(ctx, "add_users")
			},
			want: []string{transactionalSQLTemplate, transactionalSQLTemplate},
		},
		{
			name: "go",
			create: func(m *Migrator) ([]*MigrationFile, error) {
				mf, err := m.CreateGoMigration(ctx, "add_users")
				return []*MigrationFile{mf}, err
			},
			want: []string{fmt.Sprintf(goTemplate, "migrations")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			migrations := NewMigrations(WithMigrationsDirectory(t.TempDir()))
			m := NewMigrator(nil, migrations)

			files, err := tt.create(m)
			if err != nil {
				t.Fatal(err)
			}
			if len(files) != len(tt.want) {
				t.Fatalf("got %d files, want %d", len(files), len(tt.want))
			}

			for i, mf := range files {
				onDisk, err := os.ReadFile(mf.Path)
				if err != nil {
					t.Fatal(err)
				}
				if mf.Content != string(onDisk) {
					t.Errorf("%s: Content does not match the file written to disk\ngot:\n%s\nwant:\n%s",
						mf.Name, mf.Content, onDisk)
				}
				if mf.Content != tt.want[i] {
					t.Errorf("%s: got Content:\n%s\nwant:\n%s", mf.Name, mf.Content, tt.want[i])
				}
			}
		})
	}
}
