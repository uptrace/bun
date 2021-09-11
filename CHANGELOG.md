## [1.0.6](https://github.com/uptrace/bun/compare/v1.0.5...v1.0.6) (2021-09-11)


### Bug Fixes

* change unique tag to create a separate unique constraint ([8401615](https://github.com/uptrace/bun/commit/84016155a77ca77613cc054277fefadae3098757))
* improve zero checker for ptr values ([2b3623d](https://github.com/uptrace/bun/commit/2b3623dd665d873911fd20ca707016929921e862))



## v1.0.5 - Sep 09 2021

- chore: tweak bundebug colors
- fix: check if table is present when appending columns
- fix: copy []byte when scanning

## v1.0.4 - Sep 08 2021

- Added support for MariaDB.
- Restored default `SET` for `ON CONFLICT DO UPDATE` queries.

## v1.0.3 - Sep 06 2021

- Fixed bulk soft deletes.
- pgdialect: fixed scanning into an array pointer.

## v1.0.2 - Sep 04 2021

- Changed to completely ignore fields marked with `bun:"-"`. If you want to be able to scan into
  such columns, use `bun:",scanonly"`.
- pgdriver: fixed SASL authentication handling.

## v1.0.1 - Sep 02 2021

- pgdriver: added erroneous zero writes retry.
- Improved column handling in Relation callback.

## v1.0.0 - Sep 01 2021

- First stable release.

## v0.4.1 - Aug 18 2021

- Fixed migrate package to properly rollback migrations.
- Added `allowzero` tag option that undoes `nullzero` option.

## v0.4.0 - Aug 11 2021

- Changed `WhereGroup` function to accept `*SelectQuery`.
- Fixed query hooks for count queries.

## v0.3.4 - Jul 19 2021

- Renamed `migrate.CreateGo` to `CreateGoMigration`.
- Added `migrate.WithPackageName` to customize the Go package name in generated migrations.
- Renamed `migrate.CreateSQL` to `CreateSQLMigrations` and changed `CreateSQLMigrations` to create
  both up and down migration files.

## v0.3.1 - Jul 12 2021

- Renamed `alias` field struct tag to `alt` so it is not confused with column alias.
- Reworked migrate package API. See
  [migrate](https://github.com/uptrace/bun/tree/master/example/migrate) example for details.

## v0.3.0 - Jul 09 2021

- Changed migrate package to return structured data instead of logging the progress. See
  [migrate](https://github.com/uptrace/bun/tree/master/example/migrate) example for details.

## v0.2.14 - Jul 01 2021

- Added [sqliteshim](https://pkg.go.dev/github.com/uptrace/bun/driver/sqliteshim) by
  [Ivan Trubach](https://github.com/tie).
- Added support for MySQL 5.7 in addition to MySQL 8.

## v0.2.12 - Jun 29 2021

- Fixed scanners for net.IP and net.IPNet.

## v0.2.10 - Jun 29 2021

- Fixed pgdriver to format passed query args.

## v0.2.9 - Jun 27 2021

- Added support for prepared statements in pgdriver.

## v0.2.7 - Jun 26 2021

- Added `UpdateQuery.Bulk` helper to generate bulk-update queries.

  Before:

  ```go
  models := []Model{
  	{42, "hello"},
  	{43, "world"},
  }
  return db.NewUpdate().
  	With("_data", db.NewValues(&models)).
  	Model(&models).
  	Table("_data").
  	Set("model.str = _data.str").
  	Where("model.id = _data.id")
  ```

  Now:

  ```go
  db.NewUpdate().
  	Model(&models).
  	Bulk()
  ```

## v0.2.5 - Jun 25 2021

- Changed time.Time to always append zero time as `NULL`.
- Added `db.RunInTx` helper.

## v0.2.4 - Jun 21 2021

- Added SSL support to pgdriver.

## v0.2.3 - Jun 20 2021

- Replaced `ForceDelete(ctx)` with `ForceDelete().Exec(ctx)` for soft deletes.

## v0.2.1 - Jun 17 2021

- Renamed `DBI` to `IConn`. `IConn` is a common interface for `*sql.DB`, `*sql.Conn`, and `*sql.Tx`.
- Added `IDB`. `IDB` is a common interface for `*bun.DB`, `bun.Conn`, and `bun.Tx`.

## v0.2.0 - Jun 16 2021

- Changed [model hooks](https://bun.uptrace.dev/guide/hooks.html#model-hooks). See
  [model-hooks](example/model-hooks) example.
- Renamed `has-one` to `belongs-to`. Renamed `belongs-to` to `has-one`. Previously Bun used
  incorrect names for these relations.
