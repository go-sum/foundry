package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/go-sum/db/ddl"
)

// Introspect queries the PostgreSQL catalog to build a Schema representing
// the current live state of the public schema. Covers extensions, tables,
// columns, indexes, and functions. Triggers are excluded (rarely altered).
func Introspect(ctx context.Context, db *sql.DB) (*ddl.Schema, error) {
	s := &ddl.Schema{}

	if err := introspectExtensions(ctx, db, s); err != nil {
		return nil, err
	}
	if err := introspectTables(ctx, db, s); err != nil {
		return nil, err
	}
	if err := introspectIndexes(ctx, db, s); err != nil {
		return nil, err
	}
	if err := introspectFunctions(ctx, db, s); err != nil {
		return nil, err
	}
	return s, nil
}

// IntrospectDSN opens a temporary connection to dsn, introspects the public
// schema, and returns the result. The connection is closed before returning.
func IntrospectDSN(ctx context.Context, dsn string) (*ddl.Schema, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("introspect: open db: %w", err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close() //nolint:errcheck
	return Introspect(ctx, db)
}

func introspectExtensions(ctx context.Context, db *sql.DB, s *ddl.Schema) error {
	rows, err := db.QueryContext(ctx, `
		SELECT extname
		FROM pg_extension
		WHERE extname <> 'plpgsql'
		ORDER BY extname
	`)
	if err != nil {
		return fmt.Errorf("introspect extensions: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("introspect extensions: scan: %w", err)
		}
		s.Extensions = append(s.Extensions, name)
	}
	return rows.Err()
}

func introspectTables(ctx context.Context, db *sql.DB, s *ddl.Schema) error {
	rows, err := db.QueryContext(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		  AND table_type   = 'BASE TABLE'
		  AND table_name   NOT LIKE '\_%' ESCAPE '\'
		ORDER BY table_name
	`)
	if err != nil {
		return fmt.Errorf("introspect tables: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("introspect tables: scan: %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close() //nolint:errcheck

	for _, tableName := range names {
		t, err := introspectColumns(ctx, db, tableName)
		if err != nil {
			return err
		}
		s.Tables = append(s.Tables, t)
	}
	return nil
}

func introspectColumns(ctx context.Context, db *sql.DB, tableName string) (ddl.Table, error) {
	t := ddl.Table{Name: tableName}

	rows, err := db.QueryContext(ctx, `
		SELECT
			c.column_name,
			CASE WHEN c.domain_name IS NOT NULL THEN c.domain_name ELSE c.udt_name END AS col_type,
			(c.is_nullable = 'YES') AS is_nullable,
			COALESCE(c.column_default, '') AS col_default,
			EXISTS (
				SELECT 1
				FROM information_schema.key_column_usage kcu
				JOIN information_schema.table_constraints tc
					ON  tc.constraint_name = kcu.constraint_name
					AND tc.table_schema    = kcu.table_schema
					AND tc.constraint_type = 'PRIMARY KEY'
				WHERE kcu.table_schema = 'public'
				  AND kcu.table_name   = $1
				  AND kcu.column_name  = c.column_name
			) AS is_pk
		FROM information_schema.columns c
		WHERE c.table_schema = 'public'
		  AND c.table_name   = $1
		ORDER BY c.ordinal_position
	`, tableName)
	if err != nil {
		return t, fmt.Errorf("introspect columns %s: %w", tableName, err)
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var col ddl.Column
		if err := rows.Scan(&col.Name, &col.Type, &col.Nullable, &col.Default, &col.IsPK); err != nil {
			return t, fmt.Errorf("introspect columns %s: scan: %w", tableName, err)
		}
		col.Raw = col.Name + " " + col.Type
		t.Columns = append(t.Columns, col)
	}
	return t, rows.Err()
}

func introspectIndexes(ctx context.Context, db *sql.DB, s *ddl.Schema) error {
	rows, err := db.QueryContext(ctx, `
		SELECT indexname, tablename, indexdef
		FROM pg_indexes
		WHERE schemaname = 'public'
		  AND indexname  NOT LIKE '%_pkey'
		ORDER BY indexname
	`)
	if err != nil {
		return fmt.Errorf("introspect indexes: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var name, table, def string
		if err := rows.Scan(&name, &table, &def); err != nil {
			return fmt.Errorf("introspect indexes: scan: %w", err)
		}
		isUnique := strings.Contains(strings.ToUpper(def), "CREATE UNIQUE INDEX")
		def = strings.ReplaceAll(def, " ON public.", " ON ")
		s.Indexes = append(s.Indexes, ddl.Index{
			Name:     name,
			Table:    table,
			IsUnique: isUnique,
			Raw:      def,
		})
	}
	return rows.Err()
}

func introspectFunctions(ctx context.Context, db *sql.DB, s *ddl.Schema) error {
	rows, err := db.QueryContext(ctx, `
		SELECT p.proname, pg_get_functiondef(p.oid)
		FROM pg_proc p
		JOIN pg_namespace n ON n.oid = p.pronamespace
		WHERE n.nspname = 'public'
		  AND p.prokind = 'f'
		ORDER BY p.proname
	`)
	if err != nil {
		return fmt.Errorf("introspect functions: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var name, body string
		if err := rows.Scan(&name, &body); err != nil {
			return fmt.Errorf("introspect functions: scan: %w", err)
		}
		s.Functions = append(s.Functions, ddl.Function{Name: name, Body: body})
	}
	return rows.Err()
}
