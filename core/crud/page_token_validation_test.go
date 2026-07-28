package crud_test

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
	"testing"
	"time"

	crudpb "github.com/Servora-Kit/servora/api/gen/go/servora/crud/v1"
	examplev1 "github.com/Servora-Kit/servora/api/gen/go/servora/example/v1"
	"github.com/Servora-Kit/servora/core/crud"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestComputeContextFingerprintIsLengthDelimited(t *testing.T) {
	t.Parallel()

	first := crud.ComputeContextFingerprint(crud.ContextFingerprintInput{
		ResourceType: "ab",
		Collection:   "c",
	})
	second := crud.ComputeContextFingerprint(crud.ContextFingerprintInput{
		ResourceType: "a",
		Collection:   "bc",
	})
	if first == second {
		t.Fatal("fingerprint collides across component boundaries")
	}
	third := crud.ComputeContextFingerprint(crud.ContextFingerprintInput{
		ResourceType:     "ab",
		Collection:       "c",
		ScopeFingerprint: []byte("scope-2"),
	})
	if first == third {
		t.Fatal("fingerprint ignores scope")
	}
}

func TestComputeContextFingerprintIncludesOrderProfile(t *testing.T) {
	t.Parallel()

	first := fingerprintWithProfile(t, "proto/string:utf8-binary-v1")
	second := fingerprintWithProfile(t, "proto/string:utf8-binary-v2")
	if first == second {
		t.Fatal("fingerprint ignores comparison profile")
	}
}

func TestValidatePageTokenPayloadChecksCursorContracts(t *testing.T) {
	t.Parallel()

	fingerprint := crud.ComputeContextFingerprint(crud.ContextFingerprintInput{ResourceType: "test.dev/Resource"})
	int32Order := finalOrderFor(t, mustTypedOrderBinding(t, "count", "count", false, "proto/int32:v1", crud.LogicalInt32))
	tests := []struct {
		name    string
		payload *crudpb.PageTokenPayload
		order   crud.FinalOrder
	}{
		{
			"valid int32",
			pageTokenPayload(fingerprint, &crudpb.CursorValue{Value: &crudpb.CursorValue_Int64Value{Int64Value: math.MaxInt32}}),
			int32Order,
		},
		{
			"arm mismatch",
			pageTokenPayload(fingerprint, &crudpb.CursorValue{Value: &crudpb.CursorValue_StringValue{StringValue: "1"}}),
			int32Order,
		},
		{
			"int32 overflow",
			pageTokenPayload(fingerprint, &crudpb.CursorValue{Value: &crudpb.CursorValue_Int64Value{Int64Value: math.MaxInt32 + 1}}),
			int32Order,
		},
		{
			"non-null key null",
			pageTokenPayload(fingerprint, &crudpb.CursorValue{Value: &crudpb.CursorValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}}),
			int32Order,
		},
		{
			"unset oneof",
			pageTokenPayload(fingerprint, &crudpb.CursorValue{}),
			int32Order,
		},
		{
			"typed nil oneof",
			pageTokenPayload(fingerprint, &crudpb.CursorValue{Value: (*crudpb.CursorValue_Int64Value)(nil)}),
			int32Order,
		},
		{
			"non-finite double",
			pageTokenPayload(fingerprint, &crudpb.CursorValue{Value: &crudpb.CursorValue_DoubleValue{DoubleValue: math.Inf(1)}}),
			finalOrderFor(t, mustTypedOrderBinding(t, "score", "score", false, "proto/double:v1", crud.LogicalFloat64)),
		},
		{
			"enum overflow",
			pageTokenPayload(fingerprint, &crudpb.CursorValue{Value: &crudpb.CursorValue_Int64Value{Int64Value: math.MaxInt32 + 1}}),
			finalOrderFor(t, mustTypedOrderBinding(t, "status", "status", false, "proto/enum:v1", crud.LogicalEnum)),
		},
		{
			"inexact float32",
			pageTokenPayload(fingerprint, &crudpb.CursorValue{Value: &crudpb.CursorValue_DoubleValue{DoubleValue: math.SmallestNonzeroFloat64}}),
			finalOrderFor(t, mustTypedOrderBinding(t, "score", "score", false, "proto/float:v1", crud.LogicalFloat32)),
		},
		{
			"invalid timestamp",
			pageTokenPayload(fingerprint, &crudpb.CursorValue{Value: &crudpb.CursorValue_TimestampValue{TimestampValue: &timestamppb.Timestamp{Seconds: 0, Nanos: 1_000_000_000}}}),
			finalOrderFor(t, mustTypedOrderBinding(t, "created_at", "create_time", false, "proto/timestamp:v1", crud.LogicalTimestamp)),
		},
		{
			"invalid timestamp offset",
			pageTokenPayload(fingerprint, &crudpb.CursorValue{
				Value:                  &crudpb.CursorValue_TimestampValue{TimestampValue: timestamppb.New(time.Unix(1, 0))},
				TimestampOffsetSeconds: 24 * 60 * 60,
			}),
			finalOrderFor(t, mustTypedOrderBinding(t, "created_at", "create_time", false, "proto/timestamp:v1", crud.LogicalTimestamp)),
		},
		{
			"invalid duration",
			pageTokenPayload(fingerprint, &crudpb.CursorValue{Value: &crudpb.CursorValue_DurationValue{DurationValue: &durationpb.Duration{Seconds: 1, Nanos: -1}}}),
			finalOrderFor(t, mustTypedOrderBinding(t, "elapsed", "elapsed", false, "proto/duration:v1", crud.LogicalDuration)),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := crud.ValidatePageTokenPayload(test.payload, fingerprint, test.order)
			if test.name == "valid int32" {
				if err != nil {
					t.Fatalf("ValidatePageTokenPayload: %v", err)
				}
				return
			}
			if !crudpb.IsCrudErrorReasonInvalidPageToken(err) {
				t.Fatalf("validation error = %v, want INVALID_PAGE_TOKEN", err)
			}
		})
	}
}

func TestValidatePageTokenPayloadPreservesTimestampOffset(t *testing.T) {
	t.Parallel()

	fingerprint := crud.ComputeContextFingerprint(crud.ContextFingerprintInput{ResourceType: "test.dev/Resource"})
	order := finalOrderFor(t, mustTypedOrderBinding(t, "created_at", "create_time", false, "proto/timestamp:v1", crud.LogicalTimestamp))
	original := time.Date(2026, time.July, 22, 12, 0, 0, 123, time.FixedZone("UTC+8", 8*60*60))
	values, err := crud.ValidatePageTokenPayload(pageTokenPayload(fingerprint, &crudpb.CursorValue{
		Value:                  &crudpb.CursorValue_TimestampValue{TimestampValue: timestamppb.New(original)},
		TimestampOffsetSeconds: 8 * 60 * 60,
	}), fingerprint, order)
	if err != nil {
		t.Fatalf("ValidatePageTokenPayload: %v", err)
	}
	got, ok := values[0].TimestampValue()
	if !ok || !got.Equal(original) {
		t.Fatalf("TimestampValue = (%v, %t), want instant %v", got, ok, original)
	}
	_, offset := got.Zone()
	if offset != 8*60*60 {
		t.Fatalf("TimestampValue offset = %d, want %d", offset, 8*60*60)
	}
}

func TestValidatePageTokenPayloadAllowsNullableValue(t *testing.T) {
	t.Parallel()

	fingerprint := crud.ComputeContextFingerprint(crud.ContextFingerprintInput{ResourceType: "test.dev/Resource"})
	nickname := mustTypedOrderBinding(t, "nickname", "nickname", true, "proto/string:utf8-binary-v1", crud.LogicalString)
	id := mustTypedOrderBinding(t, "id", "", false, "proto/uint64:v1", crud.LogicalUint64)
	assembler, err := crud.NewOrderAssembler(
		testOrderResolver{},
		[]crud.ConfiguredOrderTerm{
			crud.NewConfiguredOrderTerm(nickname, crud.OrderAscending),
			crud.NewConfiguredOrderTerm(id, crud.OrderAscending),
		},
		[]crud.ConfiguredOrderTerm{crud.NewConfiguredOrderTerm(id, crud.OrderAscending)},
	)
	if err != nil {
		t.Fatalf("NewOrderAssembler: %v", err)
	}
	order, err := assembler.Resolve(crud.OrderExpression{})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	payload := pageTokenPayload(
		fingerprint,
		&crudpb.CursorValue{Value: &crudpb.CursorValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}},
		&crudpb.CursorValue{Value: &crudpb.CursorValue_Uint64Value{Uint64Value: 7}},
	)
	values, err := crud.ValidatePageTokenPayload(payload, fingerprint, order)
	if err != nil {
		t.Fatalf("ValidatePageTokenPayload: %v", err)
	}
	if !values[0].IsNull() {
		t.Fatal("nullable cursor did not preserve null state")
	}
}

func TestValidatePageTokenPayloadRejectsContextAndCountMismatch(t *testing.T) {
	t.Parallel()

	fingerprint := crud.ComputeContextFingerprint(crud.ContextFingerprintInput{ResourceType: "test.dev/Resource"})
	order := finalOrderFor(t, mustTypedOrderBinding(t, "id", "", false, "proto/uint64:v1", crud.LogicalUint64))
	wrongFingerprint := fingerprint
	wrongFingerprint[0]++
	for _, payload := range []*crudpb.PageTokenPayload{
		pageTokenPayload(wrongFingerprint, &crudpb.CursorValue{Value: &crudpb.CursorValue_Uint64Value{Uint64Value: 1}}),
		pageTokenPayload(fingerprint),
		{Version: crud.CurrentPageTokenVersion, ContextFingerprint: []byte("short")},
	} {
		if _, err := crud.ValidatePageTokenPayload(payload, fingerprint, order); !crudpb.IsCrudErrorReasonInvalidPageToken(err) {
			t.Fatalf("validation error = %v, want INVALID_PAGE_TOKEN", err)
		}
	}
}

func fingerprintWithProfile(t *testing.T, profile string) [32]byte {
	t.Helper()
	order := finalOrderFor(t, mustTypedOrderBinding(t, "name", "display_name", false, profile, crud.LogicalString))
	return crud.ComputeContextFingerprint(crud.ContextFingerprintInput{
		ResourceType: "test.dev/Resource",
		Order:        order,
	})
}

func finalOrderFor(t *testing.T, binding crud.OrderBinding) crud.FinalOrder {
	t.Helper()
	term := crud.NewConfiguredOrderTerm(binding, crud.OrderAscending)
	assembler, err := crud.NewOrderAssembler(testOrderResolver{}, []crud.ConfiguredOrderTerm{term}, []crud.ConfiguredOrderTerm{term})
	if err != nil {
		t.Fatalf("NewOrderAssembler: %v", err)
	}
	order, err := assembler.Resolve(crud.OrderExpression{})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	return order
}

func mustTypedOrderBinding(
	t *testing.T,
	key, path string,
	nullable bool,
	profile string,
	logicalType crud.LogicalType,
) crud.OrderBinding {
	t.Helper()
	binding, err := crud.NewTypedOrderBinding(key, path, nullable, profile, logicalType)
	if err != nil {
		t.Fatalf("NewTypedOrderBinding: %v", err)
	}
	return binding
}

func pageTokenPayload(fingerprint [32]byte, cursor ...*crudpb.CursorValue) *crudpb.PageTokenPayload {
	return &crudpb.PageTokenPayload{
		Version:            crud.CurrentPageTokenVersion,
		ContextFingerprint: append([]byte(nil), fingerprint[:]...),
		Cursor:             cursor,
	}
}

func TestComputeContextFingerprintAlwaysIncludesFilterProfile(t *testing.T) {
	t.Parallel()

	preparer, err := crud.NewListPreparer()
	if err != nil {
		t.Fatalf("NewListPreparer: %v", err)
	}
	plan := crud.MustBuildResourcePlan[*examplev1.User](examplev1.UserCRUDDescriptor())
	filters := []struct {
		name   string
		filter string
	}{
		{name: "empty"},
		{name: "exact", filter: `display_name = "foo"`},
		{name: "wildcard", filter: `display_name = "*foo*"`},
	}
	for _, test := range filters {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			query, err := preparer.PrepareList(plan, crud.ListInput{Filter: test.filter})
			if err != nil {
				t.Fatalf("PrepareList: %v", err)
			}
			input := crud.ContextFingerprintInput{
				ResourceType:     "test.dev/Resource",
				Collection:       "tenants/acme/users",
				Filter:           query.Filter(),
				ScopeFingerprint: []byte("scope"),
			}
			if got, legacy := crud.ComputeContextFingerprint(input), legacyContextFingerprint(input); got == legacy {
				t.Fatal("fingerprint matches the pre-filter-profile format")
			}
		})
	}
}

func TestFilterProfileInvalidatesPreUpgradeToken(t *testing.T) {
	t.Parallel()

	input := crud.ContextFingerprintInput{ResourceType: "test.dev/Resource"}
	current := crud.ComputeContextFingerprint(input)
	legacy := legacyContextFingerprint(input)
	payload := pageTokenPayload(legacy)
	if _, err := crud.ValidatePageTokenPayload(payload, current, crud.FinalOrder{}); !crudpb.IsCrudErrorReasonInvalidPageToken(err) {
		t.Fatalf("ValidatePageTokenPayload error = %v, want INVALID_PAGE_TOKEN", err)
	}
}

func TestEquivalentWildcardFiltersShareFingerprint(t *testing.T) {
	t.Parallel()

	preparer, err := crud.NewListPreparer()
	if err != nil {
		t.Fatalf("NewListPreparer: %v", err)
	}
	plan := crud.MustBuildResourcePlan[*examplev1.User](examplev1.UserCRUDDescriptor())
	first, err := preparer.PrepareList(plan, crud.ListInput{Filter: `display_name = "**foo**"`})
	if err != nil {
		t.Fatalf("PrepareList(first): %v", err)
	}
	second, err := preparer.PrepareList(plan, crud.ListInput{Filter: `display_name = "*foo*"`})
	if err != nil {
		t.Fatalf("PrepareList(second): %v", err)
	}
	firstFingerprint := crud.ComputeContextFingerprint(crud.ContextFingerprintInput{ResourceType: "test.dev/Resource", Filter: first.Filter()})
	secondFingerprint := crud.ComputeContextFingerprint(crud.ContextFingerprintInput{ResourceType: "test.dev/Resource", Filter: second.Filter()})
	if firstFingerprint != secondFingerprint {
		t.Fatal("equivalent wildcard filters produced distinct fingerprints")
	}
}

func legacyContextFingerprint(input crud.ContextFingerprintInput) [sha256.Size]byte {
	hasher := sha256.New()
	writeLegacyFingerprintPart(hasher, []byte("servora.crud.query-context.v1"))
	writeLegacyFingerprintPart(hasher, []byte(input.ResourceType))
	writeLegacyFingerprintPart(hasher, []byte(input.Collection))
	writeLegacyFingerprintPart(hasher, []byte(input.Filter.String()))
	var orderCount [8]byte
	writeLegacyFingerprintPart(hasher, orderCount[:])
	writeLegacyFingerprintPart(hasher, input.ScopeFingerprint)
	var result [sha256.Size]byte
	copy(result[:], hasher.Sum(nil))
	return result
}

func writeLegacyFingerprintPart(hasher interface{ Write([]byte) (int, error) }, value []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = hasher.Write(length[:])
	_, _ = hasher.Write(value)
}
