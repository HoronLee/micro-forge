package crud

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	crudpb "github.com/Servora-Kit/servora/api/gen/go/servora/crud/v1"
	examplev1 "github.com/Servora-Kit/servora/api/gen/go/servora/example/v1"
	corecrud "github.com/Servora-Kit/servora/core/crud"
	kratoserrors "github.com/go-kratos/kratos/v3/errors"
)

func TestListFieldsCompilesStringMatchMatrix(t *testing.T) {
	t.Parallel()

	fields := mustListFields(t,
		Bind(examplev1.UserFields.Email, "email").Filter(),
		JSONPath(examplev1.UserFields.DisplayName, "profile", "contact", "display_name").Filter(),
		Bind(examplev1.UserFields.CreateTime, "created_at").Order(),
	)
	sources := []struct {
		name  string
		field string
	}{
		{name: "column", field: "email"},
		{name: "json_path", field: "display_name"},
	}
	modes := []struct {
		name    string
		pattern string
	}{
		{name: "exact", pattern: "foo"},
		{name: "exact_empty", pattern: ""},
		{name: "prefix", pattern: "foo*"},
		{name: "suffix", pattern: "*foo"},
		{name: "contains", pattern: "*foo*"},
		{name: "match_all", pattern: "*"},
	}
	for _, dialectName := range []string{dialect.SQLite, dialect.Postgres, dialect.MySQL} {
		for _, source := range sources {
			for _, mode := range modes {
				name := dialectName + "/" + source.name + "/" + mode.name
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					statement, arguments := compileFilterStatement(
						t,
						fields,
						dialectName,
						fmt.Sprintf(`%s = %q`, source.field, mode.pattern),
					)
					assertStringSourceSQL(t, statement, dialectName, source.name)
					assertStringModeSQL(t, statement, arguments, dialectName, mode.name)
					if strings.Contains(statement, "foo") {
						t.Fatalf("query interpolates client literal: %s", statement)
					}
				})
			}
		}
	}
}

func TestListFieldsCompilesNegatedStringMatch(t *testing.T) {
	t.Parallel()

	fields := mustListFields(t,
		Bind(examplev1.UserFields.Email, "email").Filter(),
		JSONPath(examplev1.UserFields.DisplayName, "profile", "contact", "display_name").Filter(),
		Bind(examplev1.UserFields.CreateTime, "created_at").Order(),
	)
	for _, dialectName := range []string{dialect.SQLite, dialect.Postgres, dialect.MySQL} {
		for _, field := range []string{"email", "display_name"} {
			for _, pattern := range []string{"foo*", "*"} {
				name := dialectName + "/" + field + "/" + pattern
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					statement, _ := compileFilterStatement(t, fields, dialectName, fmt.Sprintf(`%s != %q`, field, pattern))
					if !strings.Contains(statement, "NOT (") {
						t.Fatalf("query does not negate the complete match predicate: %s", statement)
					}
					if pattern == "*" && strings.Contains(statement, " IS NULL") {
						t.Fatalf("negated match-all collapses NULL three-value logic: %s", statement)
					}
				})
			}
		}
	}
}

func TestListFieldsCompilesNullableFilters(t *testing.T) {
	t.Parallel()

	fields := mustListFields(t,
		Bind(examplev1.UserFields.Nickname, "nullable_text").Filter().Nullable(),
		JSONPath(examplev1.UserFields.Email, "profile", "contact", "email").Filter().Nullable(),
		Bind(examplev1.UserFields.CreateTime, "created_at").Order(),
	)
	for _, dialectName := range []string{dialect.SQLite, dialect.Postgres, dialect.MySQL} {
		for _, field := range []string{"nickname", "email"} {
			for _, operator := range []string{"=", "!="} {
				name := dialectName + "/" + field + "/" + operator
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					statement, arguments := compileFilterStatement(t, fields, dialectName, fmt.Sprintf(`%s %s null`, field, operator))
					if len(arguments) != 0 {
						t.Fatalf("NULL query arguments = %v, want none", arguments)
					}
					if field == "nickname" && !strings.Contains(statement, "NULL") {
						t.Fatalf("column NULL query omits NULL predicate: %s", statement)
					}
					if field == "email" {
						if !strings.Contains(statement, "JSON") && !strings.Contains(statement, "->") {
							t.Fatalf("JSON NULL query omits JSON expression: %s", statement)
						}
						nullPredicate := " IS NULL"
						if operator == "!=" {
							nullPredicate = " IS NOT NULL"
						}
						if !strings.Contains(statement, nullPredicate) {
							t.Fatalf("JSON NULL query omits %q: %s", nullPredicate, statement)
						}
						if strings.Contains(statement, "JSON_CONTAINS") {
							t.Fatalf("JSON NULL query checks only a JSON null literal: %s", statement)
						}
					}
				})
			}
		}
	}
}

func TestListFieldsEscapesBackendPatternMetacharacters(t *testing.T) {
	t.Parallel()

	fields := mustListFields(t,
		Bind(examplev1.UserFields.Email, "email").Filter(),
		JSONPath(examplev1.UserFields.DisplayName, "profile", "contact", "display_name").Filter(),
		Bind(examplev1.UserFields.CreateTime, "created_at").Order(),
	)
	for _, dialectName := range []string{dialect.Postgres, dialect.MySQL} {
		for _, field := range []string{"email", "display_name"} {
			t.Run(dialectName+"/"+field, func(t *testing.T) {
				t.Parallel()
				statement, arguments := compileFilterStatement(t, fields, dialectName, fmt.Sprintf(`%s = "a!b%%c_d*"`, field))
				if !strings.Contains(statement, "ESCAPE '!'") {
					t.Fatalf("query omits explicit LIKE escape: %s", statement)
				}
				if got, want := fmt.Sprint(arguments), `[a!!b!%c!_d%]`; got != want {
					t.Fatalf("arguments = %s, want %s", got, want)
				}
			})
		}
	}
}

func TestQueryConverterClassifiesStringMatchFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		converter QueryConverter
		want      crudpb.CrudErrorReason
	}{
		{
			name: "client literal rejected",
			converter: func(corecrud.FilterValue) (any, error) {
				return nil, errors.New("unsupported literal")
			},
			want: crudpb.CrudErrorReason_CRUD_ERROR_REASON_INVALID_FILTER,
		},
		{
			name: "wildcard converted to non-string",
			converter: func(corecrud.FilterValue) (any, error) {
				return int64(7), nil
			},
			want: crudpb.CrudErrorReason_CRUD_ERROR_REASON_INTERNAL,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			fields := mustListFields(t,
				Bind(examplev1.UserFields.Email, "email").Filter().WithQueryConverter(test.converter),
				Bind(examplev1.UserFields.CreateTime, "created_at").Order(),
			)
			query := prepareListFilter(t, `email = "foo*"`)
			selector := sql.Dialect(dialect.SQLite).Select("*").From(sql.Table("users"))
			_, err := fields.filterPredicate(selector, query.Filter())
			assertCRUDReason(t, err, test.want)
		})
	}
}

func TestCustomPredicateCanRestrictStringMatchKinds(t *testing.T) {
	t.Parallel()

	fields := mustListFields(t,
		Custom(examplev1.UserFields.Email, "email_prefix", func(
			_ corecrud.FilterOperator,
			value corecrud.FilterValue,
		) (SelectorPredicate, error) {
			match, ok := value.StringMatch()
			if !ok {
				return nil, errors.New("expected string match")
			}
			if match.Kind() != corecrud.StringMatchExact && match.Kind() != corecrud.StringMatchPrefix {
				return nil, fmt.Errorf("match kind %d is not supported", match.Kind())
			}
			return func(selector *sql.Selector) *sql.Predicate {
				return sql.EQ(selector.C("email"), match.Literal())
			}, nil
		}).Filter(),
		Bind(examplev1.UserFields.CreateTime, "created_at").Order(),
	)
	selector := sql.Dialect(dialect.SQLite).Select("*").From(sql.Table("users"))
	query := prepareListFilter(t, `email = "*foo*"`)
	_, err := fields.filterPredicate(selector, query.Filter())
	assertCRUDReason(t, err, crudpb.CrudErrorReason_CRUD_ERROR_REASON_INVALID_FILTER)
}

func TestStringMatchLowererRejectsInvalidKindAsInternal(t *testing.T) {
	t.Parallel()

	_, err := stringMatchComparison(nil, corecrud.FilterOperatorEqual, corecrud.StringMatchInvalid, "")
	assertCRUDReason(t, err, crudpb.CrudErrorReason_CRUD_ERROR_REASON_INTERNAL)
}

func compileFilterStatement(
	t *testing.T,
	fields *ListFields[struct{}],
	dialectName string,
	filter string,
) (string, []any) {
	t.Helper()

	query := prepareListFilter(t, filter)
	selector := sql.Dialect(dialectName).Select("*").From(sql.Table("users"))
	predicate, err := fields.filterPredicate(selector, query.Filter())
	if err != nil {
		t.Fatalf("filterPredicate(%q): %v", filter, err)
	}
	selector.Where(predicate)
	return selector.Query()
}

func assertStringSourceSQL(t *testing.T, statement, dialectName, source string) {
	t.Helper()

	if source == "json_path" {
		fragment := "JSON_EXTRACT"
		switch dialectName {
		case dialect.Postgres:
			fragment = "->>"
		case dialect.MySQL:
			fragment = "JSON_UNQUOTE(JSON_EXTRACT"
		}
		if !strings.Contains(statement, fragment) {
			t.Fatalf("JSONPath query omits %q: %s", fragment, statement)
		}
	}
	if source == "json_path" && dialectName == dialect.MySQL &&
		!strings.Contains(statement, "CASE WHEN JSON_TYPE(") {
		t.Fatalf("MySQL JSONPath query does not normalize JSON null to SQL NULL: %s", statement)
	}
	switch dialectName {
	case dialect.Postgres:
		if !strings.Contains(statement, `COLLATE "C"`) {
			t.Fatalf("PostgreSQL query omits binary collation: %s", statement)
		}
	case dialect.MySQL:
		if !strings.Contains(statement, "BINARY ") {
			t.Fatalf("MySQL query omits binary expression: %s", statement)
		}
	case dialect.SQLite:
		if !strings.Contains(statement, "COLLATE BINARY") {
			t.Fatalf("SQLite query omits binary collation: %s", statement)
		}
	}
}

func assertStringModeSQL(t *testing.T, statement string, arguments []any, dialectName, mode string) {
	t.Helper()

	if mode == "match_all" {
		if !strings.Contains(statement, "IS NOT NULL") || len(arguments) != 0 {
			t.Fatalf("match-all query = %s args=%v, want IS NOT NULL with no args", statement, arguments)
		}
		return
	}
	if mode == "exact" || mode == "exact_empty" {
		if strings.Contains(statement, " LIKE ") || strings.Contains(statement, "instr(") || strings.Contains(statement, "substr(") {
			t.Fatalf("Exact query uses wildcard lowering: %s", statement)
		}
		want := "foo"
		if mode == "exact_empty" {
			want = ""
		}
		if len(arguments) != 1 || arguments[0] != want {
			t.Fatalf("Exact arguments = %v, want [%q]", arguments, want)
		}
		return
	}

	if dialectName == dialect.SQLite {
		fragment := "instr("
		wantArgs := "[foo]"
		if mode == "suffix" {
			fragment = "substr("
			wantArgs = "[foo foo]"
		}
		if !strings.Contains(statement, fragment) {
			t.Fatalf("SQLite %s query omits %q: %s", mode, fragment, statement)
		}
		if got := fmt.Sprint(arguments); got != wantArgs {
			t.Fatalf("SQLite %s arguments = %s, want %s", mode, got, wantArgs)
		}
		return
	}

	if !strings.Contains(statement, " LIKE ") || !strings.Contains(statement, "ESCAPE '!'") {
		t.Fatalf("%s %s query omits escaped LIKE: %s", dialectName, mode, statement)
	}
	patterns := map[string]string{
		"prefix":   "[foo%]",
		"suffix":   "[%foo]",
		"contains": "[%foo%]",
	}
	if got, want := fmt.Sprint(arguments), patterns[mode]; got != want {
		t.Fatalf("%s %s arguments = %s, want %s", dialectName, mode, got, want)
	}
}

func assertCRUDReason(t *testing.T, err error, reason crudpb.CrudErrorReason) {
	t.Helper()

	frameworkError, ok := err.(*kratoserrors.Error)
	if !ok {
		t.Fatalf("error = %v (%T), want *errors.Error", err, err)
	}
	if got, want := frameworkError.GetReason(), reason.String(); got != want {
		t.Fatalf("reason = %q, want %q", got, want)
	}
}
