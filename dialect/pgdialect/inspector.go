package pgdialect

import (
	"context"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate/sqlschema"
)

func (d *Dialect) Inspector(db *bun.DB, excludeTables ...string) sqlschema.Inspector {
	return newInspector(db, excludeTables...)
}

type Inspector struct {
	db            *bun.DB
	excludeTables []string
}

var _ sqlschema.Inspector = (*Inspector)(nil)

func newInspector(db *bun.DB, excludeTables ...string) *Inspector {
	return &Inspector{db: db, excludeTables: excludeTables}
}

func (in *Inspector) Inspect(ctx context.Context) (sqlschema.State, error) {
	var state sqlschema.State

	exclude := in.excludeTables
	if len(exclude) == 0 {
		// Avoid getting NOT IN (NULL) if bun.In() is called with an empty slice.
		exclude = []string{""}
	}

	var tables []*InformationSchemaTable
	if err := in.db.NewRaw(sqlInspectTables, bun.In(exclude)).Scan(ctx, &tables); err != nil {
		return state, err
	}

	var fks []*ForeignKey
	if err := in.db.NewRaw(sqlInspectForeignKeys, bun.In(exclude), bun.In(exclude)).Scan(ctx, &fks); err != nil {
		return state, err
	}
	state.FKs = make(map[sqlschema.FK]string, len(fks))

	for _, table := range tables {
		var columns []*InformationSchemaColumn
		if err := in.db.NewRaw(sqlInspectColumnsQuery, table.Schema, table.Name).Scan(ctx, &columns); err != nil {
			return state, err
		}

		colDefs := make(map[string]sqlschema.Column)
		uniqueGroups := make(map[string][]string)

		for _, c := range columns {
			def := c.Default
			if c.IsSerial || c.IsIdentity {
				def = ""
			} else if !c.IsDefaultLiteral {
				def = strings.ToLower(def)
			}

			colDefs[c.Name] = sqlschema.Column{
				SQLType:         c.DataType,
				VarcharLen:      c.VarcharLen,
				DefaultValue:    def,
				IsPK:            c.IsPK,
				IsNullable:      c.IsNullable,
				IsAutoIncrement: c.IsSerial,
				IsIdentity:      c.IsIdentity,
			}

			for _, group := range c.UniqueGroups {
				uniqueGroups[group] = append(uniqueGroups[group], c.Name)
			}
		}

		var unique []sqlschema.Unique
		for name, columns := range uniqueGroups {
			unique = append(unique, sqlschema.Unique{
				Name:    name,
				Columns: sqlschema.NewComposite(columns...),
			})
		}

		state.Tables = append(state.Tables, sqlschema.Table{
			Schema:           table.Schema,
			Name:             table.Name,
			Columns:          colDefs,
			UniqueContraints: unique,
		})
	}

	for _, fk := range fks {
		state.FKs[sqlschema.FK{
			From: sqlschema.C(fk.SourceSchema, fk.SourceTable, fk.SourceColumns...),
			To:   sqlschema.C(fk.TargetSchema, fk.TargetTable, fk.TargetColumns...),
		}] = fk.ConstraintName
	}
	return state, nil
}

type InformationSchemaTable struct {
	Schema string `bun:"table_schema,pk"`
	Name   string `bun:"table_name,pk"`

	Columns []*InformationSchemaColumn `bun:"rel:has-many,join:table_schema=table_schema,join:table_name=table_name"`
}

type InformationSchemaColumn struct {
	Schema           string   `bun:"table_schema"`
	Table            string   `bun:"table_name"`
	Name             string   `bun:"column_name"`
	DataType         string   `bun:"data_type"`
	VarcharLen       int      `bun:"varchar_len"`
	IsArray          bool     `bun:"is_array"`
	ArrayDims        int      `bun:"array_dims"`
	Default          string   `bun:"default"`
	IsDefaultLiteral bool     `bun:"default_is_literal_expr"`
	IsPK             bool     `bun:"is_pk"`
	IsIdentity       bool     `bun:"is_identity"`
	IndentityType    string   `bun:"identity_type"`
	IsSerial         bool     `bun:"is_serial"`
	IsNullable       bool     `bun:"is_nullable"`
	UniqueGroups     []string `bun:"unique_groups,array"`
}

type ForeignKey struct {
	ConstraintName string   `bun:"constraint_name"`
	SourceSchema   string   `bun:"schema_name"`
	SourceTable    string   `bun:"table_name"`
	SourceColumns  []string `bun:"columns,array"`
	TargetSchema   string   `bun:"target_schema"`
	TargetTable    string   `bun:"target_table"`
	TargetColumns  []string `bun:"target_columns,array"`
}

const (
	// sqlInspectTables retrieves all user-defined tables across all schemas.
	// It excludes relations from Postgres's reserved "pg_" schemas and views from the "information_schema".
	// Pass bun.In([]string{...}) to exclude tables from this inspection or bun.In([]string{''}) to include all results.
	sqlInspectTables = `
SELECT "table_schema", "table_name"
FROM information_schema.tables
WHERE table_type = 'BASE TABLE'
	AND "table_schema" <> 'information_schema'
	AND "table_schema" NOT LIKE 'pg_%'
	AND "table_name" NOT IN (?)
ORDER BY "table_schema", "table_name"
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
	"c".column_default ~ '^''.*''::.*$' OR "c".column_default ~ '^[0-9\.]+$' AS default_is_literal_expr,
	'p' = ANY("c".constraint_type) AS is_pk,
	"c".is_identity = 'YES' AS is_identity,
	"c".column_default = format('nextval(''%s_%s_seq''::regclass)', "c".table_name, "c".column_name) AS is_serial,
	COALESCE("c".identity_type, '') AS identity_type,
	"c".is_nullable = 'YES' AS is_nullable,
	"c"."unique_groups" AS unique_groups
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
		att."unique_groups",
		att."constraint_type"
	FROM information_schema.columns "c"
		LEFT JOIN (
			SELECT
				s.nspname AS "table_schema",
				"t".relname AS "table_name",
				"c".attname AS "column_name",
				"c".attndims AS array_dims,
				"c".attidentity AS identity_type,
				ARRAY_AGG(con.conname) FILTER (WHERE con.contype = 'u') AS "unique_groups",
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
ORDER BY "table_schema", "table_name", "column_name"
`

	// sqlInspectForeignKeys get FK definitions for user-defined tables.
	// Pass bun.In([]string{...}) to exclude tables from this inspection or bun.In([]string{''}) to include all results.
	sqlInspectForeignKeys = `
WITH
	"schemas" AS (
		SELECT oid, nspname
		FROM pg_namespace
	),
	"tables" AS (
		SELECT oid, relnamespace, relname, relkind
		FROM pg_class
	),
	"columns" AS (
		SELECT attrelid, attname, attnum
		FROM pg_attribute
		WHERE attisdropped = false
	)
SELECT DISTINCT
	co.conname AS "constraint_name",
	ss.nspname AS schema_name,
	s.relname AS "table_name",
	ARRAY_AGG(sc.attname) AS "columns",
	ts.nspname AS target_schema,
	"t".relname AS target_table,
	ARRAY_AGG(tc.attname) AS target_columns
FROM pg_constraint co
	LEFT JOIN "tables" s ON s.oid = co.conrelid
	LEFT JOIN "schemas" ss ON ss.oid = s.relnamespace
	LEFT JOIN "columns" sc ON sc.attrelid = s.oid AND sc.attnum = ANY(co.conkey)
	LEFT JOIN "tables" t ON t.oid = co.confrelid
	LEFT JOIN "schemas" ts ON ts.oid = "t".relnamespace
	LEFT JOIN "columns" tc ON tc.attrelid = "t".oid AND tc.attnum = ANY(co.confkey)
WHERE co.contype = 'f'
	AND co.conrelid IN (SELECT oid FROM pg_class WHERE relkind = 'r')
	AND ARRAY_POSITION(co.conkey, sc.attnum) = ARRAY_POSITION(co.confkey, tc.attnum)
	AND s.relname NOT IN (?) AND "t".relname NOT IN (?)
GROUP BY "constraint_name", "schema_name", "table_name", target_schema, target_table
`
)
