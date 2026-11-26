package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/ibmdb/go_ibm_db"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/db2dialect"
)

func TestDB2Integration(t *testing.T) {
	dsn := os.Getenv("DB2_DSN")
	if dsn == "" {
		t.Skip("DB2_DSN is not set")
	}

	ctx := context.Background()
	sqldb, err := sql.Open("go_ibm_db", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer sqldb.Close()

	db := bun.NewDB(sqldb, db2dialect.NewLUW())
	defer db.Close()

	table := bun.Ident(fmt.Sprintf("bun_db2_integration_%d", time.Now().UnixNano()))
	if _, err := db.NewCreateTable().Model((*User)(nil)).ModelTableExpr("?", table).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.NewDropTable().Model((*User)(nil)).ModelTableExpr("?", table).Exec(ctx)
	})

	user := &User{Name: "Alice", Email: fmt.Sprintf("alice+%d@example.com", time.Now().UnixNano())}
	if _, err := db.NewInsert().Model(user).ModelTableExpr("?", table).Exec(ctx); err != nil {
		t.Fatal(err)
	}

	var users []User
	if err := db.NewSelect().Model(&users).ModelTableExpr("? AS ?", table, bun.Ident("user")).Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].Name != user.Name || users[0].Email != user.Email {
		t.Fatalf("unexpected users: %#v", users)
	}

	if _, err := db.NewDelete().Model((*User)(nil)).ModelTableExpr("?", table).Where("1 = 1").Exec(ctx); err != nil {
		t.Fatal(err)
	}
	users = nil
	if err := db.NewSelect().Model(&users).ModelTableExpr("? AS ?", table, bun.Ident("user")).Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(users) != 0 {
		t.Fatalf("expected no users, got %#v", users)
	}
}
