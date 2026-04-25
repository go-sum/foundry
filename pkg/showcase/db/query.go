package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TableInfo describes a user-defined database table.
type TableInfo struct {
	Name string
}

// ColumnInfo describes a single column in a table.
type ColumnInfo struct {
	OrdinalPos   int
	Name         string
	DataType     string
	IsNullable   bool
	DefaultValue string
	IsPrimaryKey bool
}

// IndexInfo describes a single index on a table.
type IndexInfo struct {
	Name      string
	IsUnique  bool
	IsPrimary bool
	Columns   string // comma-separated column names
}

// TableData holds a page of dynamically-scanned rows with column metadata.
type TableData struct {
	Columns []string
	Rows    [][]any
	Total   int
}

func listTables(ctx context.Context, pool *pgxpool.Pool, schema string) ([]TableInfo, error) {
	rows, err := pool.Query(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = $1
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`, schema)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var t TableInfo
		if err := rows.Scan(&t.Name); err != nil {
			return nil, fmt.Errorf("scan table: %w", err)
		}
		tables = append(tables, t)
	}
	return tables, rows.Err()
}

func validateTable(ctx context.Context, pool *pgxpool.Pool, schema, tableName string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = $1
			  AND table_name   = $2
			  AND table_type   = 'BASE TABLE'
		)
	`, schema, tableName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("validate table: %w", err)
	}
	return exists, nil
}

func tableColumns(ctx context.Context, pool *pgxpool.Pool, schema, tableName string) ([]ColumnInfo, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			c.ordinal_position,
			c.column_name,
			c.data_type,
			(c.is_nullable = 'YES')       AS is_nullable,
			COALESCE(c.column_default, '') AS default_value,
			(kcu.column_name IS NOT NULL) AS is_primary_key
		FROM information_schema.columns c
		LEFT JOIN information_schema.table_constraints tc
			ON  tc.table_schema    = c.table_schema
			AND tc.table_name      = c.table_name
			AND tc.constraint_type = 'PRIMARY KEY'
		LEFT JOIN information_schema.key_column_usage kcu
			ON  kcu.constraint_name = tc.constraint_name
			AND kcu.table_schema    = tc.table_schema
			AND kcu.column_name     = c.column_name
		WHERE c.table_schema = $1
		  AND c.table_name   = $2
		ORDER BY c.ordinal_position
	`, schema, tableName)
	if err != nil {
		return nil, fmt.Errorf("table columns: %w", err)
	}
	defer rows.Close()

	var cols []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		if err := rows.Scan(
			&col.OrdinalPos, &col.Name, &col.DataType,
			&col.IsNullable, &col.DefaultValue, &col.IsPrimaryKey,
		); err != nil {
			return nil, fmt.Errorf("scan column: %w", err)
		}
		cols = append(cols, col)
	}
	return cols, rows.Err()
}

func tableIndexes(ctx context.Context, pool *pgxpool.Pool, schema, tableName string) ([]IndexInfo, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			i.relname         AS index_name,
			ix.indisunique    AS is_unique,
			ix.indisprimary   AS is_primary,
			COALESCE(
				(SELECT string_agg(a.attname, ', ' ORDER BY array_position(ix.indkey, a.attnum))
				 FROM   pg_attribute a
				 WHERE  a.attrelid = t.oid
				   AND  a.attnum   = ANY(ix.indkey)
				   AND  a.attnum   > 0),
				''
			) AS columns
		FROM  pg_index    ix
		JOIN  pg_class    t  ON t.oid = ix.indrelid
		JOIN  pg_class    i  ON i.oid = ix.indexrelid
		JOIN  pg_namespace n ON n.oid = t.relnamespace
		WHERE t.relname  = $1
		  AND n.nspname  = $2
		ORDER BY ix.indisprimary DESC, i.relname
	`, tableName, schema)
	if err != nil {
		return nil, fmt.Errorf("table indexes: %w", err)
	}
	defer rows.Close()

	var indexes []IndexInfo
	for rows.Next() {
		var idx IndexInfo
		if err := rows.Scan(&idx.Name, &idx.IsUnique, &idx.IsPrimary, &idx.Columns); err != nil {
			return nil, fmt.Errorf("scan index: %w", err)
		}
		indexes = append(indexes, idx)
	}
	return indexes, rows.Err()
}

// queryTableData returns a paginated slice of rows from tableName.
// tableName MUST be pre-validated via validateTable before calling this.
func queryTableData(ctx context.Context, pool *pgxpool.Pool, schema, tableName string, limit, offset int) (TableData, error) {
	quoted := pgx.Identifier{schema, tableName}.Sanitize()

	var total int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+quoted).Scan(&total); err != nil {
		return TableData{}, fmt.Errorf("count rows: %w", err)
	}

	rows, err := pool.Query(ctx, "SELECT * FROM "+quoted+" LIMIT $1 OFFSET $2", limit, offset)
	if err != nil {
		return TableData{}, fmt.Errorf("select rows: %w", err)
	}
	defer rows.Close()

	fields := rows.FieldDescriptions()
	cols := make([]string, len(fields))
	for i, f := range fields {
		cols[i] = string(f.Name)
	}

	var tableRows [][]any
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return TableData{}, fmt.Errorf("scan row: %w", err)
		}
		tableRows = append(tableRows, vals)
	}
	if err := rows.Err(); err != nil {
		return TableData{}, fmt.Errorf("iter rows: %w", err)
	}

	return TableData{Columns: cols, Rows: tableRows, Total: total}, nil
}
