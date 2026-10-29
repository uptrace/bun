package dbtest_test

import (
	"database/sql"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/schema"
)

const relationAliasCountryPath = "Course.ProviderAccount.Organization.Country"

const relationAliasDeepPath = relationAliasCountryPath + ".NationalityAuthority.IssuingOrganization"

type relationAliasCertificate struct {
	bun.BaseModel `bun:"table:relation_alias_certificates,alias:certificate"`
	ID            int64 `bun:",pk"`
	CollectionID  int64
	CourseID      *int64
	Course        *relationAliasCourse `bun:"rel:belongs-to,join:course_id=id"`
}

type relationAliasCourse struct {
	bun.BaseModel     `bun:"table:relation_alias_courses"`
	ID                int64 `bun:",pk"`
	ProviderAccountID int64
	ProviderAccount   *relationAliasProviderAccount `bun:"rel:belongs-to,join:provider_account_id=id"`
}

type relationAliasProviderAccount struct {
	bun.BaseModel  `bun:"table:relation_alias_provider_accounts"`
	ID             int64 `bun:",pk"`
	OrganizationID int64
	Organization   *relationAliasOrganization `bun:"rel:belongs-to,join:organization_id=id"`
}

type relationAliasOrganization struct {
	bun.BaseModel `bun:"table:relation_alias_organizations"`
	ID            int64 `bun:",pk"`
	CountryID     int64
	Country       *relationAliasCountry `bun:"rel:belongs-to,join:country_id=id"`
}

type relationAliasCountry struct {
	bun.BaseModel          `bun:"table:relation_alias_countries"`
	ID                     int64 `bun:",pk"`
	AlternateName          string
	NationalityOnly        bool
	NationalityDisplayName string
	NationalityDisplayCode string
	NationalityAuthorityID *int64
	NationalityAuthority   *relationAliasAuthority `bun:"rel:belongs-to,join:nationality_authority_id=id"`
}

type relationAliasAuthority struct {
	bun.BaseModel         `bun:"table:relation_alias_authorities"`
	ID                    int64 `bun:",pk"`
	IssuingOrganizationID int64
	IssuingOrganization   *relationAliasIssuer `bun:"rel:belongs-to,join:issuing_organization_id=id"`
}

type relationAliasIssuer struct {
	bun.BaseModel `bun:"table:relation_alias_issuers"`
	ID            int64 `bun:",pk"`
	Name          string
}

type relationAliasUnicodeCertificate struct {
	bun.BaseModel       `bun:"table:relation_alias_unicode_certificates,alias:unicode_certificate"`
	ID                  int64 `bun:",pk"`
	OrganizationID      int64
	IssuingOrganization *relationAliasOrganization `bun:"organisation_de_formation_professionnelle_agréée,rel:belongs-to,join:organization_id=id"`
}

type relationAliasCollection struct {
	bun.BaseModel `bun:"table:relation_alias_collections"`
	ID            int64                      `bun:",pk"`
	Certificates  []relationAliasCertificate `bun:"rel:has-many,join:id=collection_id"`
}

type relationAliasSQLDialect struct {
	schema.Dialect
}

func (relationAliasSQLDialect) Init(*sql.DB) {}

type relationAliasExpression struct {
	calls int
}

func (e *relationAliasExpression) AppendQuery(_ schema.QueryGen, b []byte) ([]byte, error) {
	e.calls++
	return append(b, "'manual' AS bun_alias_0"...), nil
}

func TestRelationAliasSQL(t *testing.T) {
	testEachDialect(t, func(t *testing.T, _ string, d schema.Dialect) {
		db := bun.NewDB(nil, relationAliasSQLDialect{d})
		query := db.NewSelect().Model(new(relationAliasCertificate)).Relation(relationAliasDeepPath)
		original := query.String()
		require.Equal(t, original, query.String())
		require.Equal(t, original, query.Clone().String())
		require.Equal(t, original, db.NewSelect().Model(new(relationAliasCertificate)).Relation(relationAliasDeepPath).String())

		quote := string(d.IdentQuote())
		aliases := regexp.MustCompile(` AS `+regexp.QuoteMeta(quote)+`([^`+quote+`]+)`+regexp.QuoteMeta(quote)).FindAllStringSubmatch(original, -1)
		require.NotEmpty(t, aliases)
		seen := make(map[string]bool, len(aliases))
		for _, match := range aliases {
			alias := match[1]
			require.False(t, seen[alias], "duplicate generated alias %q", alias)
			seen[alias] = true
			if d.Name() == dialect.PG {
				require.LessOrEqual(t, len(alias), 63, "PostgreSQL identifier %q", alias)
			}
		}

		country := "course__provider_account__organization__country"
		require.True(t, seen["course__id"])
		require.Len(t, country+"__alternate_name", 63)
		require.True(t, seen[country+"__alternate_name"], "a 63-byte alias must remain unchanged")
		longAliases := []string{
			country + "__nationality_only",
			country + "__nationality_display_name",
			country + "__nationality_display_code",
			country + "__nationality_authority",
			country + "__nationality_authority__issuing_organization",
		}
		for _, alias := range longAliases {
			require.Greater(t, len(alias), 63)
			if d.Name() == dialect.PG {
				require.False(t, seen[alias], "long generated alias must be shortened")
			} else {
				require.True(t, seen[alias], "non-PostgreSQL aliases must remain unchanged")
			}
		}
	})
}

func TestRelationAliasScan(t *testing.T) {
	testEachDB(t, func(t *testing.T, dbName string, db *bun.DB) {
		if dbName != pgName && dbName != pgxName && dbName != sqliteName {
			t.Skip("relation alias roundtrip requires PostgreSQL or SQLite")
		}
		want := seedRelationAliasModels(t, db)

		t.Run("slice", func(t *testing.T) {
			var got []relationAliasCertificate
			err := db.NewSelect().Model(&got).Relation(relationAliasDeepPath).OrderExpr("certificate.id").Scan(ctx)
			require.NoError(t, err)
			require.Equal(t, want, got)
		})

		t.Run("scan_and_count", func(t *testing.T) {
			var got []relationAliasCertificate
			query := db.NewSelect().Conn(db).Model(&got).Relation(relationAliasDeepPath).OrderExpr("certificate.id").Limit(1)
			original := query.String()
			count, err := query.ScanAndCount(ctx)
			require.NoError(t, err)
			require.Equal(t, int64(len(want)), count)
			require.Equal(t, want[:1], got)
			require.Equal(t, original, query.String())
		})

		t.Run("unicode_relation_names", func(t *testing.T) {
			mustResetModel(t, ctx, db, (*relationAliasUnicodeCertificate)(nil))
			expected := relationAliasUnicodeCertificate{
				ID:                  1,
				OrganizationID:      31,
				IssuingOrganization: want[0].Course.ProviderAccount.Organization,
			}
			_, err := db.NewInsert().Model(&expected).Exec(ctx)
			require.NoError(t, err)

			var got relationAliasUnicodeCertificate
			err = db.NewSelect().Model(&got).
				Relation("IssuingOrganization.Country.NationalityAuthority.IssuingOrganization").
				Where("unicode_certificate.id = ?", expected.ID).
				Scan(ctx)
			require.NoError(t, err)
			require.Equal(t, expected, got)
		})

		t.Run("repeat_and_clone", func(t *testing.T) {
			got := new(relationAliasCertificate)
			query := db.NewSelect().Model(got).Relation(relationAliasDeepPath).Where("certificate.id = ?", 1)
			original := query.String()
			for range 2 {
				*got = relationAliasCertificate{}
				require.NoError(t, query.Scan(ctx))
				require.Equal(t, want[0], *got)
				require.Equal(t, original, query.String())
			}

			var dest relationAliasCertificate
			require.NoError(t, query.Clone().Scan(ctx, &dest))
			require.Equal(t, want[0], dest)
			require.Equal(t, original, query.String())
		})

		t.Run("destination_override", func(t *testing.T) {
			query := db.NewSelect().Model(new(relationAliasCertificate)).Relation(relationAliasDeepPath).OrderExpr("certificate.id")
			var got []relationAliasCertificate
			require.NoError(t, query.Scan(ctx, &got))
			require.Equal(t, want, got)
		})

		t.Run("exec_destination", func(t *testing.T) {
			var got relationAliasCertificate
			_, err := db.NewSelect().Model(new(relationAliasCertificate)).
				Relation(relationAliasDeepPath).
				Where("certificate.id = ?", 1).
				Exec(ctx, &got)
			require.NoError(t, err)
			require.Equal(t, want[0], got)
		})

		t.Run("has_many_deep_relations", func(t *testing.T) {
			mustResetModel(t, ctx, db, (*relationAliasCollection)(nil))
			_, err := db.NewInsert().Model(&relationAliasCollection{ID: 1}).Exec(ctx)
			require.NoError(t, err)

			var got relationAliasCollection
			err = db.NewSelect().Model(&got).
				Relation("Certificates", func(q *bun.SelectQuery) *bun.SelectQuery {
					return q.OrderExpr("certificate.id")
				}).
				Relation("Certificates." + relationAliasDeepPath).
				Scan(ctx)
			require.NoError(t, err)
			require.Equal(t, relationAliasCollection{ID: 1, Certificates: want}, got)
		})

		t.Run("map_destination", func(t *testing.T) {
			query := db.NewSelect().Model(new(relationAliasCertificate)).
				Relation(relationAliasDeepPath).
				Where("certificate.id = ?", 1)
			var got map[string]any
			require.NoError(t, query.Scan(ctx, &got))
			country := "course__provider_account__organization__country"
			require.Equal(t, "Danish", got[country+"__nationality_display_name"])
			require.Equal(t, "DK", got[country+"__nationality_display_code"])
			require.Equal(t, "Northern Maritime Authority", got[country+"__nationality_authority__issuing_organization__name"])
		})

		t.Run("explicit_column_aliases", func(t *testing.T) {
			for _, tt := range []struct {
				name  string
				expr  string
				args  []any
				alias string
			}{
				{name: "quoted", expr: `'manual' AS "bun_alias_0"`, alias: "bun_alias_0"},
				{name: "placeholder", expr: "? AS ?", args: []any{"manual", bun.Ident("bun_alias_0")}, alias: "bun_alias_0"},
				{name: "uppercase", expr: "'manual' AS BUN_ALIAS_0", alias: "bun_alias_0"},
				{name: "implicit", expr: `'manual' "bun_alias_0"`, alias: "bun_alias_0"},
				{name: "unicode_escape", expr: `'manual' AS U&"bun_\0061lias_0"`, alias: "bun_alias_0"},
				{name: "custom_unicode_escape", expr: `'manual' AS U&"bun_!0061lias_0" UESCAPE '!'`, alias: "bun_alias_0"},
				{name: "multiple", expr: "'manual' AS bun_alias_0, 'secondary' AS bun_alias_1", alias: "bun_alias_0"},
			} {
				t.Run(tt.name, func(t *testing.T) {
					if dbName == sqliteName && (tt.name == "unicode_escape" || tt.name == "custom_unicode_escape") {
						t.Skip("Unicode identifier escapes require PostgreSQL")
					}
					var got map[string]any
					err := db.NewSelect().Model(new(relationAliasCertificate)).
						ColumnExpr("certificate.*").
						ColumnExpr(tt.expr, tt.args...).
						Relation(relationAliasDeepPath).
						Where("certificate.id = ?", 1).
						Scan(ctx, &got)
					require.NoError(t, err)
					alias := tt.alias
					if tt.name == "uppercase" && dbName == sqliteName {
						alias = "BUN_ALIAS_0"
					}
					require.Equal(t, "manual", got[alias])
					if tt.name == "multiple" {
						require.Equal(t, "secondary", got["bun_alias_1"])
					}
					country := "course__provider_account__organization__country"
					require.Equal(t, "Danish", got[country+"__nationality_display_name"])
					require.Equal(t, "DK", got[country+"__nationality_display_code"])
					require.Equal(t, "Northern Maritime Authority", got[country+"__nationality_authority__issuing_organization__name"])
				})
			}
		})

		t.Run("column_appender_once", func(t *testing.T) {
			expression := new(relationAliasExpression)
			var got map[string]any
			err := db.NewSelect().Model(new(relationAliasCertificate)).
				ColumnExpr("certificate.*").ColumnExpr("?", expression).
				Relation(relationAliasDeepPath).
				Where("certificate.id = ?", 1).
				Scan(ctx, &got)
			require.NoError(t, err)
			require.Equal(t, 1, expression.calls)
			require.Equal(t, "manual", got["bun_alias_0"])
			require.Equal(t, "Danish", got["course__provider_account__organization__country__nationality_display_name"])
		})

		t.Run("rows_scan_rows", func(t *testing.T) {
			rows, err := db.NewSelect().Model(new(relationAliasCertificate)).
				Relation(relationAliasDeepPath).OrderExpr("certificate.id").Rows(ctx)
			require.NoError(t, err)
			defer rows.Close()
			var got []relationAliasCertificate
			require.NoError(t, db.ScanRows(ctx, rows, &got))
			require.Equal(t, want, got)
		})

		t.Run("rows_scan_maps_other_db", func(t *testing.T) {
			rows, err := db.NewSelect().Model(new(relationAliasCertificate)).
				Relation(relationAliasDeepPath).OrderExpr("certificate.id").Rows(ctx)
			require.NoError(t, err)
			defer rows.Close()
			otherDB := bun.NewDB(db.DB, db.Dialect())
			var got []map[string]any
			require.NoError(t, otherDB.ScanRows(ctx, rows, &got))
			require.Len(t, got, len(want))
			country := "course__provider_account__organization__country"
			require.Equal(t, "Danish", got[0][country+"__nationality_display_name"])
			require.Equal(t, "DK", got[0][country+"__nationality_display_code"])
			require.Equal(t, "Northern Maritime Authority", got[0][country+"__nationality_authority__issuing_organization__name"])
			require.Equal(t, "Swedish", got[2][country+"__nationality_display_name"])
		})

		t.Run("rows_scan_row", func(t *testing.T) {
			rows, err := db.NewSelect().Model(new(relationAliasCertificate)).
				Relation(relationAliasDeepPath).OrderExpr("certificate.id").Rows(ctx)
			require.NoError(t, err)
			defer rows.Close()
			var got []relationAliasCertificate
			for rows.Next() {
				var certificate relationAliasCertificate
				require.NoError(t, db.ScanRow(ctx, rows, &certificate))
				got = append(got, certificate)
			}
			require.NoError(t, rows.Err())
			require.Equal(t, want, got)
		})

		t.Run("rows_independent_mappings", func(t *testing.T) {
			first, err := db.NewSelect().Model(new(relationAliasCertificate)).
				Relation(relationAliasCountryPath, func(q *bun.SelectQuery) *bun.SelectQuery {
					return q.Column("nationality_only")
				}).Where("certificate.id = ?", 1).Rows(ctx)
			require.NoError(t, err)
			defer first.Close()
			second, err := db.NewSelect().Model(new(relationAliasCertificate)).
				Relation(relationAliasCountryPath, func(q *bun.SelectQuery) *bun.SelectQuery {
					return q.Column("nationality_display_name")
				}).Where("certificate.id = ?", 1).Rows(ctx)
			require.NoError(t, err)
			defer second.Close()
			var firstValues, secondValues []map[string]any
			require.NoError(t, db.ScanRows(ctx, first, &firstValues))
			require.NoError(t, db.ScanRows(ctx, second, &secondValues))
			require.Len(t, firstValues, 1)
			require.Len(t, secondValues, 1)
			country := "course__provider_account__organization__country"
			require.Contains(t, firstValues[0], country+"__nationality_only")
			require.NotContains(t, firstValues[0], country+"__nationality_display_name")
			require.Equal(t, "Danish", secondValues[0][country+"__nationality_display_name"])
			require.NotContains(t, secondValues[0], country+"__nationality_only")
		})

		t.Run("handwritten_short_join_alias", func(t *testing.T) {
			var got relationAliasCertificate
			err := db.NewSelect().Model(&got).
				Relation(relationAliasCountryPath).
				Where("course__provider_account__organization__country.nationality_only = ?", true).
				Scan(ctx)
			require.NoError(t, err)
			require.Equal(t, int64(1), got.ID)
			require.True(t, got.Course.ProviderAccount.Organization.Country.NationalityOnly)
		})

		t.Run("explicit_relation_columns", func(t *testing.T) {
			var got relationAliasCertificate
			err := db.NewSelect().Model(&got).
				Relation(relationAliasCountryPath, func(q *bun.SelectQuery) *bun.SelectQuery {
					return q.Column("id", "alternate_name", "nationality_only", "nationality_display_name", "nationality_display_code")
				}).
				Where("certificate.id = ?", 1).
				Scan(ctx)
			require.NoError(t, err)
			country := got.Course.ProviderAccount.Organization.Country
			expected := *want[0].Course.ProviderAccount.Organization.Country
			expected.NationalityAuthorityID = nil
			expected.NationalityAuthority = nil
			require.Equal(t, expected, *country)
		})
	})
}

func seedRelationAliasModels(t *testing.T, db *bun.DB) []relationAliasCertificate {
	t.Helper()
	mustResetModel(t, ctx, db,
		(*relationAliasIssuer)(nil),
		(*relationAliasAuthority)(nil),
		(*relationAliasCountry)(nil),
		(*relationAliasOrganization)(nil),
		(*relationAliasProviderAccount)(nil),
		(*relationAliasCourse)(nil),
		(*relationAliasCertificate)(nil),
	)

	issuer := &relationAliasIssuer{ID: 61, Name: "Northern Maritime Authority"}
	authority := &relationAliasAuthority{ID: 51, IssuingOrganizationID: issuer.ID, IssuingOrganization: issuer}
	countries := []relationAliasCountry{
		{ID: 41, AlternateName: "Denmark", NationalityOnly: true, NationalityDisplayName: "Danish", NationalityDisplayCode: "DK", NationalityAuthorityID: &authority.ID, NationalityAuthority: authority},
		{ID: 42, AlternateName: "Sweden", NationalityDisplayName: "Swedish", NationalityDisplayCode: "SE"},
	}
	organizations := []relationAliasOrganization{
		{ID: 31, CountryID: countries[0].ID, Country: &countries[0]},
		{ID: 32, CountryID: countries[1].ID, Country: &countries[1]},
	}
	accounts := []relationAliasProviderAccount{
		{ID: 21, OrganizationID: organizations[0].ID, Organization: &organizations[0]},
		{ID: 22, OrganizationID: organizations[1].ID, Organization: &organizations[1]},
	}
	courses := []relationAliasCourse{
		{ID: 11, ProviderAccountID: accounts[0].ID, ProviderAccount: &accounts[0]},
		{ID: 12, ProviderAccountID: accounts[1].ID, ProviderAccount: &accounts[1]},
	}
	certificates := []relationAliasCertificate{
		{ID: 1, CollectionID: 1, CourseID: &courses[0].ID, Course: &courses[0]},
		{ID: 2, CollectionID: 1},
		{ID: 3, CollectionID: 1, CourseID: &courses[1].ID, Course: &courses[1]},
	}
	for _, model := range []any{issuer, authority, &countries, &organizations, &accounts, &courses, &certificates} {
		_, err := db.NewInsert().Model(model).Exec(ctx)
		require.NoError(t, err)
	}
	return certificates
}
