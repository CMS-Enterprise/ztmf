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

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
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

	cfg := config.GetInstance()

	// Only populate ephemeral local/test databases, never a deployed environment.
	// Gate on ENVIRONMENT (which defaults to "production") rather than the database
	// host. The dev api container reaches Postgres over the "postgre" compose service
	// name (compose-dev.yml overrides DB_ENDPOINT), while dev.compose.env advertises
	// "localhost" only for host-side tooling, so the old DB_ENDPOINT == "localhost"
	// check never matched inside the container and silently skipped seeding on a fresh
	// volume. ENVIRONMENT is host- and platform-agnostic: "local" for dev, "test" for
	// the Emberfall E2E stack. Deployed envs default to "production" and never set
	// DB_POPULATE, so the PopulateSql != nil clause is the primary safety gate.
	if cfg.Db.PopulateSql != nil && cfg.IsLocalOrTest() {
		err := populate(*cfg.Db.PopulateSql)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func getMigrator() *migrate.Migrator {
	if migrator == nil {
		once.Do(func() {
			conn, err := db.MigrationConn(context.Background())
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
