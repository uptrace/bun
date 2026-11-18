package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/ibmdb/go_ibm_db"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/db2dialect"
	"github.com/uptrace/bun/extra/bundebug"
)

type User struct {
	bun.BaseModel `bun:"table:bun_db2_demo_users"`

	ID        int64     `bun:"id,pk,autoincrement"`
	Name      string    `bun:"name,notnull"`
	Email     string    `bun:"email,notnull,unique"`
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

func main() {
	ctx := context.Background()

	// Connection string for IBM DB2, see github.com/ibmdb/go_ibm_db for details.
	dsn := os.Getenv("DB2_DSN")
	if dsn == "" {
		dsn = "HOSTNAME=localhost;DATABASE=testdb;PORT=50000;UID=db2inst1;PWD=password"
	}

	sqldb, err := sql.Open("go_ibm_db", dsn)
	if err != nil {
		panic(err)
	}
	defer sqldb.Close()

	db := bun.NewDB(sqldb, db2dialect.New())
	defer db.Close()

	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
		bundebug.FromEnv("BUNDEBUG"),
	))
	tableName := fmt.Sprintf("bun_db2_demo_users_%d", time.Now().Unix())
	table := bun.Ident(tableName)

	if _, err := db.NewCreateTable().Model((*User)(nil)).ModelTableExpr("?", table).Exec(ctx); err != nil {
		panic(err)
	}
	fmt.Printf("created table %s\n", strings.ToUpper(tableName))
	printUsers(ctx, db, table, "after create")

	user := &User{
		Name:  "Alice",
		Email: fmt.Sprintf("alice+%d@example.com", time.Now().UnixNano()),
	}
	if _, err := db.NewInsert().Model(user).ModelTableExpr("?", table).Exec(ctx); err != nil {
		panic(err)
	}
	fmt.Println("inserted demo user")
	printUsers(ctx, db, table, "after insert")

	if _, err := db.NewDelete().Model((*User)(nil)).ModelTableExpr("?", table).Where("1 = 1").Exec(ctx); err != nil {
		panic(err)
	}
	fmt.Println("deleted demo users")
	printUsers(ctx, db, table, "after delete")

	if _, err := db.NewDropTable().Model((*User)(nil)).ModelTableExpr("?", table).Exec(ctx); err != nil {
		panic(err)
	}
	fmt.Printf("dropped table %s\n", strings.ToUpper(tableName))
}

func printUsers(ctx context.Context, db *bun.DB, table bun.Ident, label string) {
	var users []User
	if err := db.NewSelect().
		Model(&users).
		ModelTableExpr("? AS ?", table, bun.Ident("user")).
		OrderExpr(`"user"."id" ASC`).
		Limit(10).
		Scan(ctx); err != nil {
		panic(err)
	}
	fmt.Printf("%s: %v\n", label, users)
}
