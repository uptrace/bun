package migrations

import "github.com/uptrace/bun/migrate"

var Migrator = migrate.NewMigrator(migrate.WithAutoDiscover())
