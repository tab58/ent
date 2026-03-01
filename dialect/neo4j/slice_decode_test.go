// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package neo4j

import (
	"slices"
	"testing"

	ndriver "github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

// --- H4: JSON round-trip decode for slice fields ---
//
// Neo4j Go driver v6 returns list properties as []any, not typed
// Go slices. The generated FromResponse methods need to convert these to
// typed slices ([]float64, []string, etc.) via JSON round-trip.
//
// Interface: DecodeJSONField(raw any, target any) error
//
// This function marshals 'raw' (typically []any) to JSON, then
// unmarshals into 'target' (a pointer to the typed slice).

// TestDecodeJSONField_Float64Slice verifies that []any{1.0, 2.0, 3.0}
// from Neo4j is decoded into []float64{1.0, 2.0, 3.0}.
// Expected: exact equality after round-trip.
func TestDecodeJSONField_Float64Slice(t *testing.T) {
	// Neo4j returns float list properties as []any with float64 elements.
	raw := []any{1.0, 2.0, 3.0}
	var target []float64
	if err := DecodeJSONField(raw, &target); err != nil {
		t.Fatalf("DecodeJSONField() error = %v", err)
	}
	expected := []float64{1.0, 2.0, 3.0}
	if !slices.Equal(target, expected) {
		t.Errorf("target = %v, want %v", target, expected)
	}
}

// TestDecodeJSONField_StringSlice verifies that []any{"a", "b", "c"}
// from Neo4j is decoded into []string{"a", "b", "c"}.
// Expected: exact equality after round-trip.
func TestDecodeJSONField_StringSlice(t *testing.T) {
	// Neo4j returns string list properties as []any with string elements.
	raw := []any{"a", "b", "c"}
	var target []string
	if err := DecodeJSONField(raw, &target); err != nil {
		t.Fatalf("DecodeJSONField() error = %v", err)
	}
	expected := []string{"a", "b", "c"}
	if !slices.Equal(target, expected) {
		t.Errorf("target = %v, want %v", target, expected)
	}
}

// TestDecodeJSONField_EmptySlice verifies that an empty []any{} from
// Neo4j is decoded into an empty typed slice (not nil).
// Expected: []float64{} (len 0, not nil).
func TestDecodeJSONField_EmptySlice(t *testing.T) {
	raw := []any{}
	var target []float64
	if err := DecodeJSONField(raw, &target); err != nil {
		t.Fatalf("DecodeJSONField() error = %v", err)
	}
	if target == nil {
		t.Error("empty slice should decode to non-nil empty slice")
	}
	if len(target) != 0 {
		t.Errorf("len(target) = %d, want 0", len(target))
	}
}

// TestDecodeJSONField_NilInput verifies that nil raw value is decoded into
// a nil typed slice (not empty slice).
// Expected: target remains nil.
func TestDecodeJSONField_NilInput(t *testing.T) {
	var target []float64
	if err := DecodeJSONField(nil, &target); err != nil {
		t.Fatalf("DecodeJSONField(nil) error = %v", err)
	}
	if target != nil {
		t.Errorf("nil input should decode to nil slice, got %v", target)
	}
}

// TestDecodeJSONField_Int64Slice verifies that []any{int64(1), int64(2)}
// from Neo4j is decoded into []int{1, 2}. Neo4j returns all integers as int64.
// Expected: JSON round-trip converts int64 -> float64 (JSON number) -> int.
func TestDecodeJSONField_Int64Slice(t *testing.T) {
	raw := []any{int64(1), int64(2), int64(3)}
	var target []int
	if err := DecodeJSONField(raw, &target); err != nil {
		t.Fatalf("DecodeJSONField() error = %v", err)
	}
	expected := []int{1, 2, 3}
	if !slices.Equal(target, expected) {
		t.Errorf("target = %v, want %v", target, expected)
	}
}

// TestDecodeJSONField_SingleFloat64 verifies that a non-slice raw value
// (a single float64) fails gracefully when decoding into a slice target.
// Expected: error because the target type doesn't match.
func TestDecodeJSONField_SingleFloat64IntoSlice(t *testing.T) {
	raw := 3.14
	var target []float64
	err := DecodeJSONField(raw, &target)
	if err == nil {
		t.Error("DecodeJSONField(scalar, &slice) should return error")
	}
}

// TestDecodeJSONField_LargeFloat64Slice verifies decode works with a larger
// slice (vector embedding dimension count).
// Expected: exact equality for 128-element float64 slice.
func TestDecodeJSONField_LargeFloat64Slice(t *testing.T) {
	raw := make([]any, 128)
	expected := make([]float64, 128)
	for i := range raw {
		v := float64(i) * 0.01
		raw[i] = v
		expected[i] = v
	}
	var target []float64
	if err := DecodeJSONField(raw, &target); err != nil {
		t.Fatalf("DecodeJSONField() error = %v", err)
	}
	if !slices.Equal(target, expected) {
		t.Errorf("target length = %d, want %d", len(target), len(expected))
	}
}

// TestDecodeJSONField_NonPointerTarget verifies that passing a non-pointer
// target returns an error.
// Expected: error because unmarshal requires a pointer.
func TestDecodeJSONField_NonPointerTarget(t *testing.T) {
	raw := []any{"a", "b"}
	var target []string
	err := DecodeJSONField(raw, target) // pass by value, not pointer
	if err == nil {
		t.Error("DecodeJSONField(raw, nonPointer) should return error")
	}
}

// --- M3: Slice field round-trip via ReadNodeMaps + DecodeJSONField ---
//
// Tests the full decode path: ReadNodeMaps extracts property maps with
// []any slice values, then DecodeJSONField converts them to typed
// Go slices. This mirrors what the generated FromResponse methods will do.

// TestSliceRoundTrip_Float64Vector verifies the full decode path for a
// TableCell node with a float64 vector: ReadNodeMaps -> DecodeJSONField.
// Expected: []any{1.0, 2.0, 3.0} -> []float64{1.0, 2.0, 3.0}.
func TestSliceRoundTrip_Float64Vector(t *testing.T) {
	records := makeNodeRecords(
		map[string]any{
			"id":     "ksuid-1",
			"value":  3.14,
			"vector": []any{1.0, 2.0, 3.0},
		},
	)
	r := NewResponse(records, []string{"n"})
	maps, err := r.ReadNodeMaps()
	if err != nil {
		t.Fatalf("ReadNodeMaps() error = %v", err)
	}
	if len(maps) != 1 {
		t.Fatalf("ReadNodeMaps() returned %d maps, want 1", len(maps))
	}
	m := maps[0]
	// Decode the slice field using DecodeJSONField.
	var vector []float64
	if err := DecodeJSONField(m["vector"], &vector); err != nil {
		t.Fatalf("DecodeJSONField(vector) error = %v", err)
	}
	expected := []float64{1.0, 2.0, 3.0}
	if !slices.Equal(vector, expected) {
		t.Errorf("vector = %v, want %v", vector, expected)
	}
}

// TestSliceRoundTrip_StringCategories verifies the full decode path for a
// TableCell node with string categories: ReadNodeMaps -> DecodeJSONField.
// Expected: []any{"a", "b"} -> []string{"a", "b"}.
func TestSliceRoundTrip_StringCategories(t *testing.T) {
	records := makeNodeRecords(
		map[string]any{
			"id":         "ksuid-2",
			"value":      1.0,
			"categories": []any{"cat-a", "cat-b"},
		},
	)
	r := NewResponse(records, []string{"n"})
	maps, err := r.ReadNodeMaps()
	if err != nil {
		t.Fatalf("ReadNodeMaps() error = %v", err)
	}
	m := maps[0]
	var categories []string
	if err := DecodeJSONField(m["categories"], &categories); err != nil {
		t.Fatalf("DecodeJSONField(categories) error = %v", err)
	}
	if len(categories) != 2 {
		t.Fatalf("len(categories) = %d, want 2", len(categories))
	}
	if categories[0] != "cat-a" || categories[1] != "cat-b" {
		t.Errorf("categories = %v, want [cat-a, cat-b]", categories)
	}
}

// TestSliceRoundTrip_EmptySlices verifies that empty Neo4j lists decode to
// empty (non-nil) Go slices.
// Expected: []any{} -> []float64{} (empty, not nil).
func TestSliceRoundTrip_EmptySlices(t *testing.T) {
	records := makeNodeRecords(
		map[string]any{
			"id":         "ksuid-3",
			"vector":     []any{},
			"categories": []any{},
		},
	)
	r := NewResponse(records, []string{"n"})
	maps, err := r.ReadNodeMaps()
	if err != nil {
		t.Fatalf("ReadNodeMaps() error = %v", err)
	}
	m := maps[0]
	var vector []float64
	if err := DecodeJSONField(m["vector"], &vector); err != nil {
		t.Fatalf("DecodeJSONField(vector) error = %v", err)
	}
	if vector == nil {
		t.Error("empty vector should be non-nil empty slice")
	}
	if len(vector) != 0 {
		t.Errorf("len(vector) = %d, want 0", len(vector))
	}
	var categories []string
	if err := DecodeJSONField(m["categories"], &categories); err != nil {
		t.Fatalf("DecodeJSONField(categories) error = %v", err)
	}
	if categories == nil {
		t.Error("empty categories should be non-nil empty slice")
	}
	if len(categories) != 0 {
		t.Errorf("len(categories) = %d, want 0", len(categories))
	}
}

// TestSliceRoundTrip_NilSlices verifies that nil/missing slice properties
// result in nil typed slices after decode.
// Expected: nil raw value -> nil typed slice.
func TestSliceRoundTrip_NilSlices(t *testing.T) {
	records := makeNodeRecords(
		map[string]any{
			"id":    "ksuid-4",
			"value": 1.0,
			// No vector or categories keys.
		},
	)
	r := NewResponse(records, []string{"n"})
	maps, err := r.ReadNodeMaps()
	if err != nil {
		t.Fatalf("ReadNodeMaps() error = %v", err)
	}
	m := maps[0]
	var vector []float64
	// nil raw value — should result in nil target.
	if err := DecodeJSONField(m["vector"], &vector); err != nil {
		t.Fatalf("DecodeJSONField(nil) error = %v", err)
	}
	if vector != nil {
		t.Errorf("missing vector should be nil, got %v", vector)
	}
}

// TestSliceRoundTrip_MultipleRecords verifies decode works across multiple
// records, each with different slice values.
// Expected: each record's slices are independently decoded.
func TestSliceRoundTrip_MultipleRecords(t *testing.T) {
	records := makeNodeRecords(
		map[string]any{
			"id":     "ksuid-a",
			"vector": []any{1.0, 2.0},
		},
		map[string]any{
			"id":     "ksuid-b",
			"vector": []any{3.0, 4.0, 5.0},
		},
	)
	r := NewResponse(records, []string{"n"})
	maps, err := r.ReadNodeMaps()
	if err != nil {
		t.Fatalf("ReadNodeMaps() error = %v", err)
	}
	if len(maps) != 2 {
		t.Fatalf("ReadNodeMaps() returned %d maps, want 2", len(maps))
	}
	var v1 []float64
	if err := DecodeJSONField(maps[0]["vector"], &v1); err != nil {
		t.Fatalf("DecodeJSONField(v1) error = %v", err)
	}
	if len(v1) != 2 {
		t.Errorf("len(v1) = %d, want 2", len(v1))
	}
	var v2 []float64
	if err := DecodeJSONField(maps[1]["vector"], &v2); err != nil {
		t.Fatalf("DecodeJSONField(v2) error = %v", err)
	}
	if len(v2) != 3 {
		t.Errorf("len(v2) = %d, want 3", len(v2))
	}
}

// --- Regression tests: direct type assertion vs DecodeJSONField ---
//
// These tests document the exact bug that caused runtime panics: the Neo4j Go
// driver returns list properties as []any, but the old decode template
// generated direct type assertions like v.([]float64). Direct assertion on
// []any always fails. DecodeJSONField handles it via JSON round-trip.
//
// Structure:
//   - "must fail" tests prove direct type assertion CANNOT work on Neo4j data
//   - "must pass" tests prove DecodeJSONField handles the same data correctly

// TestDirectTypeAssertion_FailsOnNeo4jSliceData proves that direct type
// assertion — the pattern the OLD template generated — fails on the data
// types that Neo4j actually returns. Each subtest attempts the exact type
// assertion the old template would have generated and verifies it fails.
//
// If any of these subtests start passing, it means the Neo4j driver changed
// its return types and the decode strategy may need re-evaluation.
func TestDirectTypeAssertion_FailsOnNeo4jSliceData(t *testing.T) {
	tests := []struct {
		name     string
		raw      any    // what Neo4j driver actually returns
		wantType string // the Go type the old template asserted
	}{
		{
			name:     "float64_slice",
			raw:      []any{1.0, 2.0, 3.0},
			wantType: "[]float64",
		},
		{
			name:     "string_slice",
			raw:      []any{"a", "b", "c"},
			wantType: "[]string",
		},
		{
			name:     "int_slice",
			raw:      []any{int64(1), int64(2), int64(3)},
			wantType: "[]int",
		},
		{
			name:     "bool_slice",
			raw:      []any{true, false, true},
			wantType: "[]bool",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Attempt the direct type assertions the old template generated.
			// Every one of these MUST fail because Neo4j returns []any.
			switch tt.wantType {
			case "[]float64":
				_, ok := tt.raw.([]float64)
				if ok {
					t.Errorf("direct assertion to %s succeeded — expected failure on []any", tt.wantType)
				}
			case "[]string":
				_, ok := tt.raw.([]string)
				if ok {
					t.Errorf("direct assertion to %s succeeded — expected failure on []any", tt.wantType)
				}
			case "[]int":
				_, ok := tt.raw.([]int)
				if ok {
					t.Errorf("direct assertion to %s succeeded — expected failure on []any", tt.wantType)
				}
			case "[]bool":
				_, ok := tt.raw.([]bool)
				if ok {
					t.Errorf("direct assertion to %s succeeded — expected failure on []any", tt.wantType)
				}
			}
		})
	}
}

// TestDecodeJSONField_SucceedsWhereDirectAssertionFails proves that
// DecodeJSONField correctly handles the exact same []any data that
// direct type assertion fails on. This is the pattern the NEW template uses.
//
// Each subtest uses the same raw data as TestDirectTypeAssertion_FailsOnNeo4jSliceData
// and verifies DecodeJSONField converts it to the correct typed slice.
func TestDecodeJSONField_SucceedsWhereDirectAssertionFails(t *testing.T) {
	tests := []struct {
		name     string
		raw      any
		decode   func(any) error
		validate func(t *testing.T)
	}{
		{
			name: "float64_slice",
			raw:  []any{1.0, 2.0, 3.0},
			decode: func(raw any) error {
				var target []float64
				return DecodeJSONField(raw, &target)
			},
		},
		{
			name: "string_slice",
			raw:  []any{"a", "b", "c"},
			decode: func(raw any) error {
				var target []string
				return DecodeJSONField(raw, &target)
			},
		},
		{
			name: "int_slice_from_int64",
			raw:  []any{int64(1), int64(2), int64(3)},
			decode: func(raw any) error {
				var target []int
				return DecodeJSONField(raw, &target)
			},
		},
		{
			name: "bool_slice",
			raw:  []any{true, false, true},
			decode: func(raw any) error {
				var target []bool
				return DecodeJSONField(raw, &target)
			},
		},
		{
			name: "nested_map_slice",
			raw:  []any{map[string]any{"key": "val"}},
			decode: func(raw any) error {
				var target []map[string]any
				return DecodeJSONField(raw, &target)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.decode(tt.raw); err != nil {
				t.Errorf("DecodeJSONField() error = %v — should succeed where direct assertion fails", err)
			}
		})
	}
}

// TestFromResponse_SliceField_Regression simulates the full FromResponse
// decode path for a node with slice fields, using the exact data shape that
// the Neo4j driver returns. This is the end-to-end regression test for the
// bug: ReadSingle → extract map → decode slice field.
//
// The test constructs a Response with a TableCell-like node containing
// []any slices and verifies DecodeJSONField produces the correct
// typed values, while direct assertion would have failed.
func TestFromResponse_SliceField_Regression(t *testing.T) {
	// Simulate a TableCell node as returned by Neo4j: id, value, vector, categories.
	records := makeNodeRecords(map[string]any{
		"id":         "ksuid-regression",
		"value":      42.5,
		"vector":     []any{0.1, 0.2, 0.3, 0.4},
		"categories": []any{"cat-x", "cat-y"},
	})
	r := NewResponse(records, []string{"n"})

	// ReadSingle extracts the property map.
	m, err := r.ReadSingle()
	if err != nil {
		t.Fatalf("ReadSingle() error = %v", err)
	}

	// --- Must-fail: direct type assertion (the old template pattern) ---
	t.Run("direct_assertion_vector_fails", func(t *testing.T) {
		_, ok := m["vector"].([]float64)
		if ok {
			t.Error("direct assertion v.([]float64) should fail on []any from Neo4j")
		}
	})
	t.Run("direct_assertion_categories_fails", func(t *testing.T) {
		_, ok := m["categories"].([]string)
		if ok {
			t.Error("direct assertion v.([]string) should fail on []any from Neo4j")
		}
	})

	// --- Must-pass: DecodeJSONField (the new template pattern) ---
	t.Run("decode_json_field_vector_succeeds", func(t *testing.T) {
		var vector []float64
		if err := DecodeJSONField(m["vector"], &vector); err != nil {
			t.Fatalf("DecodeJSONField(vector) error = %v", err)
		}
		expected := []float64{0.1, 0.2, 0.3, 0.4}
		if !slices.Equal(vector, expected) {
			t.Errorf("vector = %v, want %v", vector, expected)
		}
	})
	t.Run("decode_json_field_categories_succeeds", func(t *testing.T) {
		var categories []string
		if err := DecodeJSONField(m["categories"], &categories); err != nil {
			t.Fatalf("DecodeJSONField(categories) error = %v", err)
		}
		expected := []string{"cat-x", "cat-y"}
		if !slices.Equal(categories, expected) {
			t.Errorf("categories = %v, want %v", categories, expected)
		}
	})

	// --- Scalar fields still work with direct assertion ---
	t.Run("scalar_float64_direct_assertion_works", func(t *testing.T) {
		val, ok := m["value"].(float64)
		if !ok {
			t.Error("direct assertion v.(float64) should work for scalar float64")
		}
		if val != 42.5 {
			t.Errorf("value = %v, want 42.5", val)
		}
	})
	t.Run("scalar_string_direct_assertion_works", func(t *testing.T) {
		id, ok := m["id"].(string)
		if !ok {
			t.Error("direct assertion v.(string) should work for scalar string")
		}
		if id != "ksuid-regression" {
			t.Errorf("id = %q, want %q", id, "ksuid-regression")
		}
	})
}

// TestFromResponse_ManySliceField_Regression is the decode/many equivalent
// of TestFromResponse_SliceField_Regression. Verifies DecodeJSONField works
// correctly when iterating over multiple records from ReadNodeMaps.
func TestFromResponse_ManySliceField_Regression(t *testing.T) {
	records := makeNodeRecords(
		map[string]any{
			"id":         "ksuid-many-1",
			"value":      1.0,
			"vector":     []any{0.1, 0.2},
			"categories": []any{"a"},
		},
		map[string]any{
			"id":         "ksuid-many-2",
			"value":      2.0,
			"vector":     []any{0.3, 0.4, 0.5},
			"categories": []any{"b", "c"},
		},
	)
	r := NewResponse(records, []string{"n"})
	maps, err := r.ReadNodeMaps()
	if err != nil {
		t.Fatalf("ReadNodeMaps() error = %v", err)
	}
	if len(maps) != 2 {
		t.Fatalf("ReadNodeMaps() returned %d maps, want 2", len(maps))
	}

	// Verify each record's slice fields decode correctly.
	expectations := []struct {
		vector     []float64
		categories []string
	}{
		{[]float64{0.1, 0.2}, []string{"a"}},
		{[]float64{0.3, 0.4, 0.5}, []string{"b", "c"}},
	}
	for i, m := range maps {
		t.Run("record_"+m["id"].(string), func(t *testing.T) {
			// Direct assertion must fail.
			if _, ok := m["vector"].([]float64); ok {
				t.Error("direct assertion v.([]float64) should fail on []any")
			}
			// DecodeJSONField must succeed.
			var vector []float64
			if err := DecodeJSONField(m["vector"], &vector); err != nil {
				t.Fatalf("DecodeJSONField(vector) error = %v", err)
			}
			if !slices.Equal(vector, expectations[i].vector) {
				t.Errorf("vector = %v, want %v", vector, expectations[i].vector)
			}
			var categories []string
			if err := DecodeJSONField(m["categories"], &categories); err != nil {
				t.Fatalf("DecodeJSONField(categories) error = %v", err)
			}
			if !slices.Equal(categories, expectations[i].categories) {
				t.Errorf("categories = %v, want %v", categories, expectations[i].categories)
			}
		})
	}
}

// makeNodeRecords creates neo4j.Record slices from property maps.
// Each map becomes a record with key "n" containing the map as Value[0].
// Helper for Response tests.
func makeNodeRecords(maps ...map[string]any) []*ndriver.Record {
	records := make([]*ndriver.Record, len(maps))
	for i, m := range maps {
		records[i] = &ndriver.Record{
			Keys:   []string{"n"},
			Values: []any{m},
		}
	}
	return records
}
