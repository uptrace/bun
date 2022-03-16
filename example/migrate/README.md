# Bun migrations example

To run migrations:

```shell
BUNDEBUG=2 go run . db migrate
```

To rollback migrations:

```shell
go run . db rollback
```

To view status of migrations:

```shell
go run . db status
```

To create a Go migration:

```shell
go run . db create_go go_migration_name
```

To create a SQL migration:

```shell
go run . db create_sql sql_migration_name
```

To get help:

```shell
go run . db

NAME:
   bun db - database commands

USAGE:
   bun db command [command options] [arguments...]

COMMANDS:
   init        create migration tables
   migrate     migrate database
   rollback    rollback the last migration group
   unlock      unlock migrations
   create_go   create a Go migration
   create_sql  create a SQL migration
   help, h     Shows a list of commands or help for one command

OPTIONS:
   --help, -h  show help (default: false)
```

See [docs](https://bun.uptrace.dev/guide/migrations.html) for details.
