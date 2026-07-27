/*
package migrations is used internally to specify DB schema updates that need to run on process start

All migrations should be registered by calling appendMigration from an init() function in a file
dedicated to the migration. Registration is pure (append to a slice, no I/O): the database
connection is only opened by Run(), so importing this package — e.g. from its test binary —
never requires a reachable database.
Be aware that multiple init() funcs are executed in lexical file name order, so when adding multiple
changes in a single PR be sure to name the files in a way that applies them in the right order if the order matters.
*/
package migrations

import (
	"context"
	"log"

	"github.com/CMS-Enterprise/ztmf/backend/internal/config"
	"github.com/CMS-Enterprise/ztmf/backend/internal/db"
	"github.com/jackc/tern/v2/migrate"
)

type migration struct {
	name    string
	upSQL   string
	downSQL string
}

// registry holds migrations in registration (= lexical file name) order.
var registry []migration

func appendMigration(name, upSQL, downSQL string) {
	registry = append(registry, migration{name, upSQL, downSQL})
}

func Run() {
	log.Println("executing migrations...")

	ctx := context.Background()

	conn, err := db.MigrationConn(ctx)
	if err != nil {
		log.Fatal(err)
		return
	}

	migrator, err := migrate.NewMigrator(ctx, conn, "dbversions")
	if err != nil {
		log.Fatal(err)
		return
	}

	for _, m := range registry {
		migrator.AppendMigration(m.name, m.upSQL, m.downSQL)
	}

	err = migrator.Migrate(ctx)
	if err != nil {
		log.Fatal(err)
		return
	}

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
