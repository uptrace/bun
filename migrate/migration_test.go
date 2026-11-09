package migrate

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/uptrace/bun"
)

type recordQueryHook struct{ queries []string }

func (h *recordQueryHook) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	return ctx
}

func (h *recordQueryHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	h.queries = append(h.queries, event.Query)
}

func TestSQLMigrationKeepsBlankLinesInsideQuery(t *testing.T) {
	// A non-transactional migration, so the failing commit of this driver is never reached.
	migrator := newTestMigrator(t, "bun-migrate-failcommit")
	hook := new(recordQueryHook)
	migrator.db = migrator.db.WithQueryHook(hook)

	const name = "20260101000000_test.up.sql"
	fsys := fstest.MapFS{
		name: &fstest.MapFile{Data: []byte("\n" +
			"--bun:split\n" +
			"\n" +
			"INSERT INTO notes VALUES ('line1\n" +
			"\n" +
			"line3');\n" +
			"\n" +
			"--bun:split\n" +
			"\n" +
			"SELECT 1;\n")},
	}

	fn := newSQLMigrationFunc(fsys, name)
	if err := fn(context.Background(), migrator, &Migration{}); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"INSERT INTO notes VALUES ('line1\n\nline3');\n\n",
		"SELECT 1;\n",
	}
	if len(hook.queries) != len(want) {
		t.Fatalf("got queries %q, want %q", hook.queries, want)
	}
	for i := range want {
		if hook.queries[i] != want[i] {
			t.Fatalf("got queries %q, want %q", hook.queries, want)
		}
	}
}
