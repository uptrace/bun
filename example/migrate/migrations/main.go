package migrations

import "github.com/uptrace/bun/migrate"

var Migrator = migrate.NewMigrator()

func init() {
	if err := Migrator.DiscoverCaller(); err != nil {
		panic(err)
	}
}
