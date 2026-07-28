package crud_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	examplev1 "github.com/Servora-Kit/servora/api/gen/go/servora/example/v1"
	"github.com/Servora-Kit/servora/core/crud"
)

type stringMatchConformance struct {
	Valid []struct {
		Name            string `json:"name"`
		WireFilter      string `json:"wireFilter"`
		Kind            string `json:"kind"`
		Literal         string `json:"literal"`
		CanonicalFilter string `json:"canonicalFilter"`
	} `json:"valid"`
	Invalid []struct {
		Name          string `json:"name"`
		WireFilter    string `json:"wireFilter"`
		ErrorContains string `json:"errorContains"`
	} `json:"invalid"`
}

func TestStringMatchConformanceVectors(t *testing.T) {
	t.Parallel()

	vectors := loadStringMatchConformance(t)
	preparer, err := crud.NewListPreparer()
	if err != nil {
		t.Fatalf("NewListPreparer: %v", err)
	}
	plan := crud.MustBuildResourcePlan[*examplev1.User](examplev1.UserCRUDDescriptor())

	for _, vector := range vectors.Valid {
		t.Run(vector.Name, func(t *testing.T) {
			query, err := preparer.PrepareList(plan, crud.ListInput{Filter: vector.WireFilter})
			if err != nil {
				t.Fatalf("PrepareList(%q): %v", vector.WireFilter, err)
			}
			match, ok := query.Filter().Root().Value().StringMatch()
			if !ok {
				t.Fatal("StringMatch() = (_, false), want typed string match")
			}
			if got, want := match.Kind(), conformanceStringMatchKind(t, vector.Kind); got != want {
				t.Fatalf("StringMatch.Kind() = %v, want %v", got, want)
			}
			if got, want := match.Literal(), vector.Literal; got != want {
				t.Fatalf("StringMatch.Literal() = %q, want %q", got, want)
			}
			if got, want := query.Filter().String(), vector.CanonicalFilter; got != want {
				t.Fatalf("Filter.String() = %q, want %q", got, want)
			}

			roundTrip, err := preparer.PrepareList(plan, crud.ListInput{Filter: query.Filter().String()})
			if err != nil {
				t.Fatalf("PrepareList(canonical): %v", err)
			}
			if got, want := roundTrip.Filter().String(), vector.CanonicalFilter; got != want {
				t.Fatalf("round-trip Filter.String() = %q, want %q", got, want)
			}
		})
	}

	for _, vector := range vectors.Invalid {
		t.Run(vector.Name, func(t *testing.T) {
			_, err := preparer.PrepareList(plan, crud.ListInput{Filter: vector.WireFilter})
			if err == nil {
				t.Fatalf("PrepareList(%q) unexpectedly succeeded", vector.WireFilter)
			}
			if !strings.Contains(err.Error(), vector.ErrorContains) {
				t.Fatalf("PrepareList(%q) error = %q, want substring %q", vector.WireFilter, err, vector.ErrorContains)
			}
		})
	}
}

func TestStringMatchZeroValueIsInvalid(t *testing.T) {
	t.Parallel()

	var value crud.FilterValue
	match, ok := value.StringMatch()
	if ok {
		t.Fatal("zero FilterValue StringMatch() reported a string match")
	}
	if got, want := match.Kind(), crud.StringMatchInvalid; got != want {
		t.Fatalf("zero StringMatch.Kind() = %v, want %v", got, want)
	}
	if got := match.Literal(); got != "" {
		t.Fatalf("zero StringMatch.Literal() = %q, want empty", got)
	}
}

func loadStringMatchConformance(t *testing.T) stringMatchConformance {
	t.Helper()

	contents, err := os.ReadFile("../../conformance/crud/string_matches.json")
	if err != nil {
		t.Fatalf("read string match vectors: %v", err)
	}
	var vectors stringMatchConformance
	if err := json.Unmarshal(contents, &vectors); err != nil {
		t.Fatalf("decode string match vectors: %v", err)
	}
	return vectors
}

func conformanceStringMatchKind(t *testing.T, value string) crud.StringMatchKind {
	t.Helper()

	switch value {
	case "Exact":
		return crud.StringMatchExact
	case "Prefix":
		return crud.StringMatchPrefix
	case "Suffix":
		return crud.StringMatchSuffix
	case "Contains":
		return crud.StringMatchContains
	default:
		t.Fatalf("unsupported conformance string match kind %q", value)
		return crud.StringMatchInvalid
	}
}
