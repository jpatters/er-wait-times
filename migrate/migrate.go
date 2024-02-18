package migrate

import (
	"context"
	"database/sql"
	"log/slog"
	"path"

	"embed"

	"github.com/jmoiron/sqlx"
)

//go:embed all:migrations
var Migrations embed.FS

func initMigrationTable(tx *sqlx.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS _migrations (
			id BIGSERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			ran_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`)

	if err != nil {
		slog.Error("Error creating migrations table", "error", err)
		return err
	}

	return nil
}

func Migrate(db *sqlx.DB) error {
	slog.Info("Running migrations")

	tx, err := db.BeginTxx(context.Background(), &sql.TxOptions{})
	if err != nil {
		slog.Error("Error beginning transaction", "error", err)
		return err
	}

	err = initMigrationTable(tx)
	if err != nil {
		slog.Error("Error initializing migration table", "error", err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		slog.Error("Error committing transaction", "error", err)
		return err
	}

	migrations, err := Migrations.ReadDir("migrations")
	if err != nil {
		slog.Error("Error reading migrations", "error", err)
		return err
	}

	for _, migration := range migrations {
		rows, err := db.Queryx("SELECT * FROM _migrations WHERE name = $1", migration.Name())
		if err != nil {
			slog.Error("Error checking if migration has been run", "name", migration.Name(), "error", err)
			return err
		}

		hasAlreadyBeenRun := rows.Next()
		rows.Close()

		if hasAlreadyBeenRun {
			slog.Info("Skipping migration", "name", migration.Name())
			continue
		}

		migrationSQL, err := Migrations.ReadFile(path.Join("migrations", migration.Name()))
		if err != nil {
			slog.Error("Error reading migration", "name", migration.Name(), "error", err)
			return err
		}

		tx, err = db.BeginTxx(context.Background(), &sql.TxOptions{})
		if err != nil {
			slog.Error("Error getting database connection", "error", err)
			return err
		}

		slog.Info("Running migration", "name", migration.Name())

		_, err = tx.Exec(string(migrationSQL))
		if err != nil {
			slog.Error("Error running migration", "name", migration.Name(), "error", err)
			return err
		}

		_, err = tx.Exec("INSERT INTO _migrations (name) VALUES ($1)", migration.Name())

		if err != nil {
			slog.Error("Error recording migration", "name", migration.Name(), "error", err)
			return err
		}

		err = tx.Commit()

		if err != nil {
			return err
		}
	}

	return nil
}
