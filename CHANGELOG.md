# Changelog

## v0.3.0

- Added helpers `db.Select`, `db.Insert`, `db.Update`, and `db.Delete` that only support struct and
  slice-based models.
- Removed `WherePK`, because it meant too many different things depending on which query it was
  called.

  You should rewrite queries like:

  ```go
  err := db.NewSelect().Model(model).WherePK().Scan(ctx)
  res, err := db.NewInsert().Model(model).Exec(ctx)
  res, err := db.NewUpdate().Model(model).WherePK().Exec(ctx)
  res, err := db.NewDelete().Model(model).WherePK().Exec(ctx)
  ```

  to

  ```go
  err := db.Select(ctx, model)
  err := db.Insert(ctx, model)
  err := db.Update(ctx, model)
  err := db.Delete(ctx, model)
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
