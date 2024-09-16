/*
package migrations is used internally to specify DB schema updates that need to run on process start

All migrations should be appended using the init() function in a file dedicated to the migration.
Be aware that multiple init() funcs are executed in lexical file name order, so when adding multiple
changes in a single PR be sure to name the files in a way that applies them in the right order if the order matters.
*/
package migrations

import (
	"context"
	"log"
	"sync"

	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/tern/v2/migrate"
)

var (
	migrator *migrate.Migrator
	once     sync.Once
)

func Run() {
	log.Println("executing migrations...")
	err := getMigrator().Migrate(context.Background())
	if err != nil {
		log.Fatal(err)
		return
	}
	migrator = nil
}

func getMigrator() *migrate.Migrator {
	if migrator == nil {
		once.Do(func() {
			conn, err := db.Conn(context.Background())
			if err != nil {
				log.Fatal(err)
				return
			}

			migrator, err = migrate.NewMigrator(context.Background(), conn, "dbversions")
			if err != nil {
				log.Fatal(err)
				return
			}
		})
	}
	return migrator
}
