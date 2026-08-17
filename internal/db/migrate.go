package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// FindSQLDir locates the schema/seed directory relative to the process cwd.
func FindSQLDir() string {
	candidates := []string{"sql", "./sql"}
	if wd, err := os.Getwd(); err == nil {
		dir := wd
		for i := 0; i < 6; i++ {
			candidates = append(candidates, filepath.Join(dir, "sql"))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "001_init.sql")); err == nil {
			return c
		}
	}
	return "sql"
}

// ApplySchemaAndSeed runs sql/001_init.sql then sql/002_seed.sql if not yet applied.
func ApplySchemaAndSeed(ctx context.Context, pool *Pool, sqlDir string) error {
	if sqlDir == "" {
		sqlDir = FindSQLDir()
	}
	files := []struct {
		id   int
		name string
	}{
		{1, "001_init.sql"},
		{2, "002_seed.sql"},
		{3, "003_skills.sql"},
	}
	for _, f := range files {
		applied, err := migrationApplied(ctx, pool, f.id)
		if err != nil {
			// schema_migrations may not exist yet
			applied = false
		}
		if applied {
			continue
		}
		path := filepath.Join(sqlDir, f.name)
		body, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		if err := execSQL(ctx, pool, string(body)); err != nil {
			return fmt.Errorf("apply %s: %w", f.name, err)
		}
		if _, err := pool.p.Exec(ctx, `INSERT INTO schema_migrations (id) VALUES ($1) ON CONFLICT DO NOTHING`, f.id); err != nil {
			return err
		}
		log.Printf("applied %s", f.name)
	}
	return nil
}

func migrationApplied(ctx context.Context, pool *Pool, id int) (bool, error) {
	var n int
	err := pool.p.QueryRow(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE id=$1`, id).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func execSQL(ctx context.Context, pool *Pool, script string) error {
	// pgx cannot run multiple statements in one simple Exec with all drivers;
	// split on ';' at statement boundaries (good enough for our migration files).
	stmts := splitSQL(script)
	for _, s := range stmts {
		if _, err := pool.p.Exec(ctx, s); err != nil {
			return fmt.Errorf("%w\nstatement: %s", err, truncate(s, 120))
		}
	}
	return nil
}

func splitSQL(script string) []string {
	var out []string
	var b strings.Builder
	for _, line := range strings.Split(script, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "--") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
		if strings.HasSuffix(trim, ";") {
			stmt := strings.TrimSpace(b.String())
			stmt = strings.TrimSuffix(stmt, ";")
			if stmt != "" {
				out = append(out, stmt)
			}
			b.Reset()
		}
	}
	if rest := strings.TrimSpace(b.String()); rest != "" {
		out = append(out, rest)
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
