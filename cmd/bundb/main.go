/*
TODO:

- Add a mechanism to detect potentially duplicate migration files. That is,
once we've collected migrations in a bytes.Buffer, check if 'migrations/' package
has another migration files that:
 1. is identical in content
 2. belongs to the migration that has not been applied yet

If we find such migration, prompt the user for confirmation, unless -force flag is set.
Ideally, we should be able to ignore "transactional" for this purpose,
i.e. same_thing.up.tx.sql should overwrite same_thing.up.sql.

- Store configured options to env variables? E.g. after 'bundb init --create-directory=db-migrations/'
set BUNDB_MIGRATIONS=db-migrations, so that subsequent commands can be run without additional parameters.
Although... this way we are moving towards a .bundb.config or something.
*/
package main

import (
	"log"
	"os"

	"github.com/uptrace/bun/extra/buncli"
)

func main() {
	log.SetPrefix("bundb: ")
	if err := buncli.NewStandalone("bundb").Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
