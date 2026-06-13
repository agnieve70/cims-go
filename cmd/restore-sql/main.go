package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"cims-go/internal/config"
	appdb "cims-go/internal/db"

	"github.com/jackc/pgx/v5"
)

func main() {
	var sqlPath string
	var psqlPath string
	var skipEmpty bool
	var yes bool

	flag.StringVar(&sqlPath, "file", "", "path to the PostgreSQL .sql backup file to load")
	flag.StringVar(&psqlPath, "psql", "psql", "path to psql.exe or psql")
	flag.BoolVar(&skipEmpty, "skip-empty", false, "skip emptying existing public tables before loading")
	flag.BoolVar(&yes, "yes", false, "confirm this destructive restore")
	flag.Parse()

	if strings.TrimSpace(sqlPath) == "" {
		log.Fatal("missing -file path to a PostgreSQL .sql backup")
	}
	if !yes {
		log.Fatal("refusing to restore without -yes; this command empties/replaces database tables")
	}

	absSQLPath, err := filepath.Abs(sqlPath)
	must(err)
	if _, err := os.Stat(absSQLPath); err != nil {
		log.Fatalf("backup file is not readable: %v", err)
	}

	cfg, err := config.Load()
	must(err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if !skipEmpty {
		must(emptyPublicTables(ctx, cfg))
	}

	must(runPSQL(ctx, psqlPath, cfg.DatabaseURL, absSQLPath))
	fmt.Println("Restore completed successfully.")
}

func emptyPublicTables(ctx context.Context, cfg config.Config) error {
	pool, err := appdb.OpenPool(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		return err
	}
	defer pool.Close()

	rows, err := pool.Query(ctx, `
		select table_name
		from information_schema.tables
		where table_schema = 'public'
		  and table_type = 'BASE TABLE'
		order by table_name`)
	if err != nil {
		return fmt.Errorf("list public tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return fmt.Errorf("scan table: %w", err)
		}
		tables = append(tables, pgx.Identifier{"public", table}.Sanitize())
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read public tables: %w", err)
	}
	if len(tables) == 0 {
		fmt.Println("No public tables found to empty.")
		return nil
	}

	sql := "truncate table " + strings.Join(tables, ", ") + " restart identity cascade"
	if _, err := pool.Exec(ctx, sql); err != nil {
		return fmt.Errorf("empty public tables: %w", err)
	}
	fmt.Printf("Emptied %d public tables.\n", len(tables))
	return nil
}

func runPSQL(ctx context.Context, psqlPath, databaseURL, sqlPath string) error {
	cmd := exec.CommandContext(ctx, psqlPath, databaseURL, "-v", "ON_ERROR_STOP=1", "-f", sqlPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run psql restore: %w", err)
	}
	return nil
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
