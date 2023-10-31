package pgdialect

import (
	"context"
	"fmt"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqltype"
	"github.com/uptrace/bun/schema"
)

func (d *Dialect) Inspector(db *bun.DB, excludeTables ...string) schema.Inspector {
	return newInspector(db, excludeTables...)
}

type Inspector struct {
	db            *bun.DB
	excludeTables []string
}

var _ schema.Inspector = (*Inspector)(nil)

func newInspector(db *bun.DB, excludeTables ...string) *Inspector {
	return &Inspector{db: db, excludeTables: excludeTables}
}

func (in *Inspector) Inspect(ctx context.Context) (schema.State, error) {
	var state schema.State

	exclude := in.excludeTables
	if len(exclude) == 0 {
		// Avoid getting NOT IN (NULL) if bun.In() is called with an empty slice.
		exclude = []string{""}
	}

	var tables []*InformationSchemaTable
	if err := in.db.NewRaw(sqlInspectTables, bun.In(exclude)).Scan(ctx, &tables); err != nil {
		return state, err
	}

	for _, table := range tables {
		var columns []*InformationSchemaColumn
		if err := in.db.NewRaw(sqlInspectColumnsQuery, table.Schema, table.Name).Scan(ctx, &columns); err != nil {
			return state, err
		}
		colDefs := make(map[string]schema.ColumnDef)
		for _, c := range columns {
			dataType := fromDatabaseType(c.DataType)
			if strings.EqualFold(dataType, sqltype.VarChar) && c.VarcharLen > 0 {
				dataType = fmt.Sprintf("%s(%d)", dataType, c.VarcharLen)
			}

			def := c.Default
			if c.IsSerial || c.IsIdentity {
				def = ""
			}

			colDefs[c.Name] = schema.ColumnDef{
				SQLType:         strings.ToLower(dataType),
				IsPK:            c.IsPK,
				IsNullable:      c.IsNullable,
				IsAutoIncrement: c.IsSerial,
				IsIdentity:      c.IsIdentity,
				DefaultValue:    def,
			}
		}

		state.Tables = append(state.Tables, schema.TableDef{
			Schema:  table.Schema,
			Name:    table.Name,
			Columns: colDefs,
		})
	}
	return state, nil
}

type InformationSchemaTable struct {
	bun.BaseModel

	Schema string `bun:"table_schema,pk"`
	Name   string `bun:"table_name,pk"`

	Columns []*InformationSchemaColumn `bun:"rel:has-many,join:table_schema=table_schema,join:table_name=table_name"`
}

type InformationSchemaColumn struct {
	bun.BaseModel

	Schema        string   `bun:"table_schema"`
	Table         string   `bun:"table_name"`
	Name          string   `bun:"column_name"`
	DataType      string   `bun:"data_type"`
	VarcharLen    int      `bun:"varchar_len"`
	IsArray       bool     `bun:"is_array"`
	ArrayDims     int      `bun:"array_dims"`
	Default       string   `bun:"default"`
	IsPK          bool     `bun:"is_pk"`
	IsIdentity    bool     `bun:"is_identity"`
	IndentityType string   `bun:"identity_type"`
	IsSerial      bool     `bun:"is_serial"`
	IsNullable    bool     `bun:"is_nullable"`
	IsUnique      bool     `bun:"is_unique"`
	UniqueGroup   []string `bun:"unique_group,array"`
}

const (
	// sqlInspectTables retrieves all user-defined tables across all schemas.
	// It excludes relations from Postgres's reserved "pg_" schemas and views from the "information_schema".
	sqlInspectTables = `
SELECT table_schema, table_name
FROM information_schema.tables
WHERE table_type = 'BASE TABLE'
	AND table_schema <> 'information_schema'
	AND table_schema NOT LIKE 'pg_%'
	AND table_name NOT IN (?)
	`

	// sqlInspectColumnsQuery retrieves column definitions for the specified table.
	// Unlike sqlInspectTables and sqlInspectSchema, it should be passed to bun.NewRaw
	// with additional args for table_schema and table_name.
	sqlInspectColumnsQuery = `
SELECT
	"c".table_schema,
	"c".table_name,
	"c".column_name,
	"c".data_type,
	"c".character_maximum_length::integer AS varchar_len,
	"c".data_type = 'ARRAY' AS is_array,
	COALESCE("c".array_dims, 0) AS array_dims,
	CASE
		WHEN "c".column_default ~ '^''.*''::.*$' THEN substring("c".column_default FROM '^''(.*)''::.*$')
		ELSE "c".column_default
	END AS "default",
	'p' = ANY("c".constraint_type) AS is_pk,
	"c".is_identity = 'YES' AS is_identity,
	"c".column_default = format('nextval(''%s_%s_seq''::regclass)', "c".table_name, "c".column_name) AS is_serial,
	COALESCE("c".identity_type, '') AS identity_type,
	"c".is_nullable = 'YES' AS is_nullable,
	'u' = ANY("c".constraint_type) AS is_unique,
	"c"."constraint_name" AS unique_group
FROM (
	SELECT
		"table_schema",
		"table_name",
		"column_name",
		"c".data_type,
		"c".character_maximum_length,
		"c".column_default,
		"c".is_identity,
		"c".is_nullable,
		att.array_dims,
		att.identity_type,
		att."constraint_name",
		att."constraint_type"
	FROM information_schema.columns "c"
		LEFT JOIN (
			SELECT
				s.nspname AS "table_schema",
				"t".relname AS "table_name",
				"c".attname AS "column_name",
				"c".attndims AS array_dims,
				"c".attidentity AS identity_type,
				ARRAY_AGG(con.conname) AS "constraint_name",
				ARRAY_AGG(con.contype) AS "constraint_type"
			FROM ( 
				SELECT 
					conname,
					contype,
					connamespace,
					conrelid,
					conrelid AS attrelid,
					UNNEST(conkey) AS attnum
				FROM pg_constraint
			) con
				LEFT JOIN pg_attribute "c" USING (attrelid, attnum)
				LEFT JOIN pg_namespace s ON s.oid = con.connamespace
				LEFT JOIN pg_class "t" ON "t".oid = con.conrelid
			GROUP BY 1, 2, 3, 4, 5
		) att USING ("table_schema", "table_name", "column_name")
	) "c"
WHERE "table_schema" = ? AND "table_name" = ?
	`

	// sqlInspectSchema retrieves column type definitions for all user-defined tables.
	// Other relations, such as views and indices, as well as Posgres's internal relations are excluded.
	sqlInspectSchema = `
SELECT
	"t"."table_schema",
	"t".table_name,
	"c".column_name,
	"c".data_type,
	"c".character_maximum_length::integer AS varchar_len,
	"c".data_type = 'ARRAY' AS is_array,
	COALESCE("c".array_dims, 0) AS array_dims,
	CASE
		WHEN "c".column_default ~ '^''.*''::.*$' THEN substring("c".column_default FROM '^''(.*)''::.*$')
		ELSE "c".column_default
	END AS "default",
	"c".constraint_type = 'p' AS is_pk,
	"c".is_identity = 'YES' AS is_identity,
	"c".column_default = format('nextval(''%s_%s_seq''::regclass)', "t".table_name, "c".column_name) AS is_serial,
	COALESCE("c".identity_type, '') AS identity_type,
	"c".is_nullable = 'YES' AS is_nullable,
	"c".constraint_type = 'u' AS is_unique,
	"c"."constraint_name" AS unique_group
FROM information_schema.tables "t"
	LEFT JOIN (
		SELECT
			"table_schema",
			"table_name",
			"column_name",
			"c".data_type,
			"c".character_maximum_length,
			"c".column_default,
			"c".is_identity,
			"c".is_nullable,
			att.array_dims,
			att.identity_type,
			att."constraint_name",
			att."constraint_type"
		FROM information_schema.columns "c"
			LEFT JOIN (
				SELECT
					s.nspname AS table_schema,
					"t".relname AS "table_name",
					"c".attname AS "column_name",
					"c".attndims AS array_dims,
					"c".attidentity AS identity_type,
					con.conname AS "constraint_name",
					con.contype AS "constraint_type"
				FROM ( 
					SELECT 
						conname,
						contype,
						connamespace,
						conrelid,
						conrelid AS attrelid,
						UNNEST(conkey) AS attnum
					FROM pg_constraint
				) con
					LEFT JOIN pg_attribute "c" USING (attrelid, attnum)
					LEFT JOIN pg_namespace s ON s.oid = con.connamespace
					LEFT JOIN pg_class "t" ON "t".oid = con.conrelid
			) att USING (table_schema, "table_name", "column_name")
	) "c" USING (table_schema, "table_name")
WHERE table_type = 'BASE TABLE'
	AND table_schema <> 'information_schema'
	AND table_schema NOT LIKE 'pg_%'
	`
)
