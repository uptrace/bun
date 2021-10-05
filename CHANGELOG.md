## [1.0.10](https://github.com/uptrace/bun/compare/v1.0.9...v1.0.10) (2021-10-05)


### Bug Fixes

* add UpdateQuery.OmitZero ([2294db6](https://github.com/uptrace/bun/commit/2294db61d228711435fff1075409a30086b37555))
* make ExcludeColumn work with many-to-many queries ([300e12b](https://github.com/uptrace/bun/commit/300e12b993554ff839ec4fa6bbea97e16aca1b55))
* **mysqldialect:** append time in local timezone ([e763cc8](https://github.com/uptrace/bun/commit/e763cc81eac4b11fff4e074ad3ff6cd970a71697))
* **tagparser:** improve parsing options with brackets ([0daa61e](https://github.com/uptrace/bun/commit/0daa61edc3c4d927ed260332b99ee09f4bb6b42f))


### Features

* add timetz parsing ([6e415c4](https://github.com/uptrace/bun/commit/6e415c4c5fa2c8caf4bb4aed4e5897fe5676f5a5))



## [1.0.9](https://github.com/uptrace/bun/compare/v1.0.8...v1.0.9) (2021-09-27)


### Bug Fixes

* change DBStats to use uint32 instead of uint64 to make it work on i386 ([caca2a7](https://github.com/uptrace/bun/commit/caca2a7130288dec49fa26b49c8550140ee52f4c))


### Features

* add IQuery and QueryEvent.IQuery ([b762942](https://github.com/uptrace/bun/commit/b762942fa3b1d8686d0a559f93f2a6847b83d9c1))
* add QueryEvent.Model ([7688201](https://github.com/uptrace/bun/commit/7688201b485d14d3e393956f09a3200ea4d4e31d))
* **bunotel:** add experimental bun.query.timing metric ([2cdb384](https://github.com/uptrace/bun/commit/2cdb384678631ccadac0fb75f524bd5e91e96ee2))
* **pgdriver:** add Config.ConnParams to session config params ([408caf0](https://github.com/uptrace/bun/commit/408caf0bb579e23e26fc6149efd6851814c22517))
* **pgdriver:** allow specifying timeout in DSN ([7dbc71b](https://github.com/uptrace/bun/commit/7dbc71b3494caddc2e97d113f00067071b9e19da))



## [1.0.8](https://github.com/uptrace/bun/compare/v1.0.7...v1.0.8) (2021-09-18)


### Bug Fixes

* don't append soft delete where for insert queries with on conflict clause ([27c477c](https://github.com/uptrace/bun/commit/27c477ce071d4c49c99a2531d638ed9f20e33461))
* improve bun.NullTime to accept string ([73ad6f5](https://github.com/uptrace/bun/commit/73ad6f5640a0a9b09f8df2bc4ab9cb510021c50c))
* make allowzero work with auto-detected primary keys ([82ca87c](https://github.com/uptrace/bun/commit/82ca87c7c49797d507b31fdaacf8343716d4feff))
* support soft deletes on nil model ([0556e3c](https://github.com/uptrace/bun/commit/0556e3c63692a7f4e48659d52b55ffd9cca0202a))



## [1.0.7](https://github.com/uptrace/bun/compare/v1.0.6...v1.0.7) (2021-09-15)


### Bug Fixes

* don't append zero time as NULL without nullzero tag ([3b8d9cb](https://github.com/uptrace/bun/commit/3b8d9cb4e39eb17f79a618396bbbe0adbc66b07b))
* **pgdriver:** return PostgreSQL DATE as a string ([40be0e8](https://github.com/uptrace/bun/commit/40be0e8ea85f8932b7a410a6fc2dd3acd2d18ebc))
* specify table alias for soft delete where ([5fff1dc](https://github.com/uptrace/bun/commit/5fff1dc1dd74fa48623a24fa79e358a544dfac0b))


### Features

* add SelectQuery.Exists helper ([c3e59c1](https://github.com/uptrace/bun/commit/c3e59c1bc58b43c4b8e33e7d170ad33a08fbc3c7))



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
