## [1.0.25](https://github.com/uptrace/bun/compare/v1.0.24...v1.0.25) (2022-02-24)


### Bug Fixes

* add missing autoincrement to migration models. Fixes [#457](https://github.com/uptrace/bun/issues/457) ([268eea9](https://github.com/uptrace/bun/commit/268eea966a5b7dcb372ceff25da5831cd687afb5))
* fix soft_delete nullzero warning. Fixes [#458](https://github.com/uptrace/bun/issues/458) ([8e72048](https://github.com/uptrace/bun/commit/8e720480cd40282b8cdb0ad7c66565bd80a4a813))



## [1.0.24](https://github.com/uptrace/bun/compare/v1.0.23...v1.0.24) (2022-02-22)


### Bug Fixes

* fix missing autoincrement warning ([3bc9c72](https://github.com/uptrace/bun/commit/3bc9c721e1c1c5104c256a0c01c4525df6ecefc2))



## [1.0.23](https://github.com/uptrace/bun/compare/v1.0.22...v1.0.23) (2022-02-22)

### Deprecated

In the comming v1.1.x release, Bun will stop automatically adding `,pk,autoincrement` options on
`ID int64/int32` fields. This version (v1.0.23) only prints a warning when it encounters such
fields, but the code will continue working as before.

To fix warnings, add missing options:

```diff
type Model struct {
-	 ID int64
+	 ID int64 `bun:",pk,autoincrement"`
}
```

To silence warnings:

```go
bun.SetWarnLogger(log.New(ioutil.Discard, "", log.LstdFlags))
```

Bun will also print a warning on [soft delete](https://bun.uptrace.dev/guide/soft-deletes.html)
fields without a `,nullzero` option. You can fix the warning by adding missing `,nullzero` or
`,allowzero` options.

In v1.1.x, such options as `,nopk` and `,allowzero` will not be necessary and will be removed.

### Bug Fixes

- append slice values
  ([4a65129](https://github.com/uptrace/bun/commit/4a651294fb0f1e73079553024810c3ead9777311))
- don't automatically set pk, nullzero, and autoincrement options
  ([519a0df](https://github.com/uptrace/bun/commit/519a0df9707de01a418aba0d6b7482cfe4c9a532))

### Features

- add CreateTableQuery.DetectForeignKeys
  ([a958fcb](https://github.com/uptrace/bun/commit/a958fcbab680b0c5ad7980f369c7b73f7673db87))

## [1.0.22](https://github.com/uptrace/bun/compare/v1.0.21...v1.0.22) (2022-01-28)

### Bug Fixes

- improve scan error message
  ([54048b2](https://github.com/uptrace/bun/commit/54048b296b9648fd62107ce6fa6fd7e6e2a648c7))
- properly discover json.Marshaler on ptr field
  ([3b321b0](https://github.com/uptrace/bun/commit/3b321b08601c4b8dc6bcaa24adea20875883ac14))

### Breaking (MySQL, MariaDB)

- **insert:** get last insert id only with pk support auto increment
  ([79e7c79](https://github.com/uptrace/bun/commit/79e7c797beea54bfc9dc1cb0141a7520ff941b4d)). Make
  sure your MySQL models have `bun:",pk,autoincrement"` options if you are using autoincrements.

### Features

- refuse to start when version check does not pass
  ([ff8d767](https://github.com/uptrace/bun/commit/ff8d76794894eeaebede840e5199720f3f5cf531))
- support Column in ValuesQuery
  ([0707679](https://github.com/uptrace/bun/commit/0707679b075cac57efa8e6fe9019b57b2da4bcc7))

## [1.0.21](https://github.com/uptrace/bun/compare/v1.0.20...v1.0.21) (2022-01-06)

### Bug Fixes

- append where to index create
  ([1de6cea](https://github.com/uptrace/bun/commit/1de6ceaa8bba59b69fbe0cc6916d1b27da5586d8))
- check if slice is nil when calling BeforeAppendModel
  ([938d9da](https://github.com/uptrace/bun/commit/938d9dadb72ceeeb906064d9575278929d20cbbe))
- **dbfixture:** directly set matching types via reflect
  ([780504c](https://github.com/uptrace/bun/commit/780504cf1da687fc51a22d002ea66e2ccc41e1a3))
- properly handle driver.Valuer and type:json
  ([a17454a](https://github.com/uptrace/bun/commit/a17454ac6b95b2a2e927d0c4e4aee96494108389))
- support scanning string into uint64
  ([73cc117](https://github.com/uptrace/bun/commit/73cc117a9f7a623ced1fdaedb4546e8e7470e4d3))
- unique module name for opentelemetry example
  ([f2054fe](https://github.com/uptrace/bun/commit/f2054fe1d11cea3b21d69dab6f6d6d7d97ba06bb))

### Features

- add anonymous fields with type name
  ([508375b](https://github.com/uptrace/bun/commit/508375b8f2396cb088fd4399a9259584353eb7e5))
- add baseQuery.GetConn()
  ([81a9bee](https://github.com/uptrace/bun/commit/81a9beecb74fed7ec3574a1d42acdf10a74e0b00))
- create new queries from baseQuery
  ([ae1dd61](https://github.com/uptrace/bun/commit/ae1dd611a91c2b7c79bc2bc12e9a53e857791e71))
- support INSERT ... RETURNING for MariaDB >= 10.5.0
  ([b6531c0](https://github.com/uptrace/bun/commit/b6531c00ecbd4c7ec56b4131fab213f9313edc1b))

## [1.0.20](https://github.com/uptrace/bun/compare/v1.0.19...v1.0.20) (2021-12-19)

### Bug Fixes

- add Event.QueryTemplate and change Event.Query to be always formatted
  ([52b1ccd](https://github.com/uptrace/bun/commit/52b1ccdf3578418aa427adef9dcf942d90ae4fdd))
- change GetTableName to return formatted table name in case ModelTableExpr
  ([95144dd](https://github.com/uptrace/bun/commit/95144dde937b4ac88b36b0bd8b01372421069b44))
- change ScanAndCount to work with transactions
  ([5b3f2c0](https://github.com/uptrace/bun/commit/5b3f2c021c424da366caffd33589e8adde821403))
- **dbfixture:** directly call funcs bypassing template eval
  ([a61974b](https://github.com/uptrace/bun/commit/a61974ba2d24361c5357fb9bda1f3eceec5a45cd))
- don't append CASCADE by default in drop table/column queries
  ([26457ea](https://github.com/uptrace/bun/commit/26457ea5cb20862d232e6e5fa4dbdeac5d444bf1))
- **migrate:** mark migrations as applied on error so the migration can be rolled back
  ([8ce33fb](https://github.com/uptrace/bun/commit/8ce33fbbac8e33077c20daf19a14c5ff2291bcae))
- respect nullzero when appending struct fields. Fixes
  [#339](https://github.com/uptrace/bun/issues/339)
  ([ffd02f3](https://github.com/uptrace/bun/commit/ffd02f3170b3cccdd670a48d563cfb41094c05d6))
- reuse tx for relation join ([#366](https://github.com/uptrace/bun/issues/366))
  ([60bdb1a](https://github.com/uptrace/bun/commit/60bdb1ac84c0a699429eead3b7fdfbf14fe69ac6))

### Features

- add `Dialect()` to Transaction and IDB interface
  ([693f1e1](https://github.com/uptrace/bun/commit/693f1e135999fc31cf83b99a2530a695b20f4e1b))
- add model embedding via embed:prefix\_
  ([9a2cedc](https://github.com/uptrace/bun/commit/9a2cedc8b08fa8585d4bfced338bd0a40d736b1d))
- change the default logoutput to stderr
  ([4bf5773](https://github.com/uptrace/bun/commit/4bf577382f19c64457cbf0d64490401450954654)),
  closes [#349](https://github.com/uptrace/bun/issues/349)

## [1.0.19](https://github.com/uptrace/bun/compare/v1.0.18...v1.0.19) (2021-11-30)

### Features

- add support for column:name to specify column name
  ([e37b460](https://github.com/uptrace/bun/commit/e37b4602823babc8221970e086cfed90c6ad4cf4))

## [1.0.18](https://github.com/uptrace/bun/compare/v1.0.17...v1.0.18) (2021-11-24)

### Bug Fixes

- use correct operation for UpdateQuery
  ([687a004](https://github.com/uptrace/bun/commit/687a004ef7ec6fe1ef06c394965dd2c2d822fc82))

### Features

- add pgdriver.Notify
  ([7ee443d](https://github.com/uptrace/bun/commit/7ee443d1b869d8ddc4746850f7425d0a9ccd012b))
- CreateTableQuery.PartitionBy and CreateTableQuery.TableSpace
  ([cd3ab4d](https://github.com/uptrace/bun/commit/cd3ab4d8f3682f5a30b87c2ebc2d7e551d739078))
- **pgdriver:** add CopyFrom and CopyTo
  ([0b97703](https://github.com/uptrace/bun/commit/0b977030b5c05f509e11d13550b5f99dfd62358d))
- support InsertQuery.Ignore on PostgreSQL
  ([1aa9d14](https://github.com/uptrace/bun/commit/1aa9d149da8e46e63ff79192e394fde4d18d9b60))

## [1.0.17](https://github.com/uptrace/bun/compare/v1.0.16...v1.0.17) (2021-11-11)

### Bug Fixes

- don't call rollback when tx is already done
  ([8246c2a](https://github.com/uptrace/bun/commit/8246c2a63e2e6eba314201c6ba87f094edf098b9))
- **mysql:** escape backslash char in strings
  ([fb32029](https://github.com/uptrace/bun/commit/fb32029ea7604d066800b16df21f239b71bf121d))

## [1.0.16](https://github.com/uptrace/bun/compare/v1.0.15...v1.0.16) (2021-11-07)

### Bug Fixes

- call query hook when tx is started, committed, or rolled back
  ([30e85b5](https://github.com/uptrace/bun/commit/30e85b5366b2e51951ef17a0cf362b58f708dab1))
- **pgdialect:** auto-enable array support if the sql type is an array
  ([62c1012](https://github.com/uptrace/bun/commit/62c1012b2482e83969e5c6f5faf89e655ce78138))

### Features

- support multiple tag options join:left_col1=right_col1,join:left_col2=right_col2
  ([78cd5aa](https://github.com/uptrace/bun/commit/78cd5aa60a5c7d1323bb89081db2b2b811113052))
- **tag:** log with bad tag name
  ([4e82d75](https://github.com/uptrace/bun/commit/4e82d75be2dabdba1a510df4e1fbb86092f92f4c))

## [1.0.15](https://github.com/uptrace/bun/compare/v1.0.14...v1.0.15) (2021-10-29)

### Bug Fixes

- fixed bug creating table when model has no columns
  ([042c50b](https://github.com/uptrace/bun/commit/042c50bfe41caaa6e279e02c887c3a84a3acd84f))
- init table with dialect once
  ([9a1ce1e](https://github.com/uptrace/bun/commit/9a1ce1e492602742bb2f587e9ed24e50d7d07cad))

### Features

- accept columns in WherePK
  ([b3e7035](https://github.com/uptrace/bun/commit/b3e70356db1aa4891115a10902316090fccbc8bf))
- support ADD COLUMN IF NOT EXISTS
  ([ca7357c](https://github.com/uptrace/bun/commit/ca7357cdfe283e2f0b94eb638372e18401c486e9))

## [1.0.14](https://github.com/uptrace/bun/compare/v1.0.13...v1.0.14) (2021-10-24)

### Bug Fixes

- correct binary serialization for mysql ([#259](https://github.com/uptrace/bun/issues/259))
  ([e899f50](https://github.com/uptrace/bun/commit/e899f50b22ef6759ef8c029a6cd3f25f2bde17ef))
- correctly escape single quotes in pg arrays
  ([3010847](https://github.com/uptrace/bun/commit/3010847f5c2c50bce1969689a0b77fd8a6fb7e55))
- use BLOB sql type to encode []byte in MySQL and SQLite
  ([725ec88](https://github.com/uptrace/bun/commit/725ec8843824a7fc8f4058ead75ab0e62a78192a))

### Features

- warn when there are args but no placeholders
  ([06dde21](https://github.com/uptrace/bun/commit/06dde215c8d0bde2b2364597190729a160e536a1))

## [1.0.13](https://github.com/uptrace/bun/compare/v1.0.12...v1.0.13) (2021-10-17)

### Breaking Change

- **pgdriver:** enable TLS by default with InsecureSkipVerify=true
  ([15ec635](https://github.com/uptrace/bun/commit/15ec6356a04d5cf62d2efbeb189610532dc5eb31))

### Features

- add BeforeAppendModelHook
  ([0b55de7](https://github.com/uptrace/bun/commit/0b55de77aaffc1ed0894ef16f45df77bca7d93c1))
- **pgdriver:** add support for unix socket DSN
  ([f398cec](https://github.com/uptrace/bun/commit/f398cec1c3873efdf61ac0b94ebe06c657f0cf91))

## [1.0.12](https://github.com/uptrace/bun/compare/v1.0.11...v1.0.12) (2021-10-14)

### Bug Fixes

- add InsertQuery.ColumnExpr to specify columns
  ([60ffe29](https://github.com/uptrace/bun/commit/60ffe293b37912d95f28e69734ff51edf4b27da7))
- **bundebug:** change WithVerbose to accept a bool flag
  ([b2f8b91](https://github.com/uptrace/bun/commit/b2f8b912de1dc29f40c79066de1e9d6379db666c))
- **pgdialect:** fix bytea[] handling
  ([a5ca013](https://github.com/uptrace/bun/commit/a5ca013742c5a2e947b43d13f9c2fc0cf6a65d9c))
- **pgdriver:** rename DriverOption to Option
  ([51c1702](https://github.com/uptrace/bun/commit/51c1702431787d7369904b2624e346bf3e59c330))
- support allowzero on the soft delete field
  ([d0abec7](https://github.com/uptrace/bun/commit/d0abec71a9a546472a83bd70ed4e6a7357659a9b))

### Features

- **bundebug:** allow to configure the hook using env var, for example, BUNDEBUG={0,1,2}
  ([ce92852](https://github.com/uptrace/bun/commit/ce928524cab9a83395f3772ae9dd5d7732af281d))
- **bunotel:** report DBStats metrics
  ([b9b1575](https://github.com/uptrace/bun/commit/b9b15750f405cdbd345b776f5a56c6f742bc7361))
- **pgdriver:** add Error.StatementTimeout
  ([8a7934d](https://github.com/uptrace/bun/commit/8a7934dd788057828bb2b0983732b4394b74e960))
- **pgdriver:** allow setting Network in config
  ([b24b5d8](https://github.com/uptrace/bun/commit/b24b5d8014195a56ad7a4c634c10681038e6044d))

## [1.0.11](https://github.com/uptrace/bun/compare/v1.0.10...v1.0.11) (2021-10-05)

### Bug Fixes

- **mysqldialect:** remove duplicate AppendTime
  ([8d42090](https://github.com/uptrace/bun/commit/8d42090af34a1760004482c7fc0923b114d79937))

## [1.0.10](https://github.com/uptrace/bun/compare/v1.0.9...v1.0.10) (2021-10-05)

### Bug Fixes

- add UpdateQuery.OmitZero
  ([2294db6](https://github.com/uptrace/bun/commit/2294db61d228711435fff1075409a30086b37555))
- make ExcludeColumn work with many-to-many queries
  ([300e12b](https://github.com/uptrace/bun/commit/300e12b993554ff839ec4fa6bbea97e16aca1b55))
- **mysqldialect:** append time in local timezone
  ([e763cc8](https://github.com/uptrace/bun/commit/e763cc81eac4b11fff4e074ad3ff6cd970a71697))
- **tagparser:** improve parsing options with brackets
  ([0daa61e](https://github.com/uptrace/bun/commit/0daa61edc3c4d927ed260332b99ee09f4bb6b42f))

### Features

- add timetz parsing
  ([6e415c4](https://github.com/uptrace/bun/commit/6e415c4c5fa2c8caf4bb4aed4e5897fe5676f5a5))

## [1.0.9](https://github.com/uptrace/bun/compare/v1.0.8...v1.0.9) (2021-09-27)

### Bug Fixes

- change DBStats to use uint32 instead of uint64 to make it work on i386
  ([caca2a7](https://github.com/uptrace/bun/commit/caca2a7130288dec49fa26b49c8550140ee52f4c))

### Features

- add IQuery and QueryEvent.IQuery
  ([b762942](https://github.com/uptrace/bun/commit/b762942fa3b1d8686d0a559f93f2a6847b83d9c1))
- add QueryEvent.Model
  ([7688201](https://github.com/uptrace/bun/commit/7688201b485d14d3e393956f09a3200ea4d4e31d))
- **bunotel:** add experimental bun.query.timing metric
  ([2cdb384](https://github.com/uptrace/bun/commit/2cdb384678631ccadac0fb75f524bd5e91e96ee2))
- **pgdriver:** add Config.ConnParams to session config params
  ([408caf0](https://github.com/uptrace/bun/commit/408caf0bb579e23e26fc6149efd6851814c22517))
- **pgdriver:** allow specifying timeout in DSN
  ([7dbc71b](https://github.com/uptrace/bun/commit/7dbc71b3494caddc2e97d113f00067071b9e19da))

## [1.0.8](https://github.com/uptrace/bun/compare/v1.0.7...v1.0.8) (2021-09-18)

### Bug Fixes

- don't append soft delete where for insert queries with on conflict clause
  ([27c477c](https://github.com/uptrace/bun/commit/27c477ce071d4c49c99a2531d638ed9f20e33461))
- improve bun.NullTime to accept string
  ([73ad6f5](https://github.com/uptrace/bun/commit/73ad6f5640a0a9b09f8df2bc4ab9cb510021c50c))
- make allowzero work with auto-detected primary keys
  ([82ca87c](https://github.com/uptrace/bun/commit/82ca87c7c49797d507b31fdaacf8343716d4feff))
- support soft deletes on nil model
  ([0556e3c](https://github.com/uptrace/bun/commit/0556e3c63692a7f4e48659d52b55ffd9cca0202a))

## [1.0.7](https://github.com/uptrace/bun/compare/v1.0.6...v1.0.7) (2021-09-15)

### Bug Fixes

- don't append zero time as NULL without nullzero tag
  ([3b8d9cb](https://github.com/uptrace/bun/commit/3b8d9cb4e39eb17f79a618396bbbe0adbc66b07b))
- **pgdriver:** return PostgreSQL DATE as a string
  ([40be0e8](https://github.com/uptrace/bun/commit/40be0e8ea85f8932b7a410a6fc2dd3acd2d18ebc))
- specify table alias for soft delete where
  ([5fff1dc](https://github.com/uptrace/bun/commit/5fff1dc1dd74fa48623a24fa79e358a544dfac0b))

### Features

- add SelectQuery.Exists helper
  ([c3e59c1](https://github.com/uptrace/bun/commit/c3e59c1bc58b43c4b8e33e7d170ad33a08fbc3c7))

## [1.0.6](https://github.com/uptrace/bun/compare/v1.0.5...v1.0.6) (2021-09-11)

### Bug Fixes

- change unique tag to create a separate unique constraint
  ([8401615](https://github.com/uptrace/bun/commit/84016155a77ca77613cc054277fefadae3098757))
- improve zero checker for ptr values
  ([2b3623d](https://github.com/uptrace/bun/commit/2b3623dd665d873911fd20ca707016929921e862))

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
