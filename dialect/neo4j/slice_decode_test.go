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
// Neo4j Go driver v6 returns list properties as []interface{}, not typed
// Go slices. The generated FromResponse methods need to convert these to
// typed slices ([]float64, []string, etc.) via JSON round-trip.
//
// Interface: DecodeJSONField(raw any, target any) error
//
// This function marshals 'raw' (typically []interface{}) to JSON, then
// unmarshals into 'target' (a pointer to the typed slice).

// TestDecodeJSONField_Float64Slice verifies that []interface{}{1.0, 2.0, 3.0}
// from Neo4j is decoded into []float64{1.0, 2.0, 3.0}.
// Expected: exact equality after round-trip.
func TestDecodeJSONField_Float64Slice(t *testing.T) {
	// Neo4j returns float list properties as []interface{} with float64 elements.
	raw := []interface{}{1.0, 2.0, 3.0}
	var target []float64
	if err := DecodeJSONField(raw, &target); err != nil {
		t.Fatalf("DecodeJSONField() error = %v", err)
	}
	expected := []float64{1.0, 2.0, 3.0}
	if !slices.Equal(target, expected) {
		t.Errorf("target = %v, want %v", target, expected)
	}
}

// TestDecodeJSONField_StringSlice verifies that []interface{}{"a", "b", "c"}
// from Neo4j is decoded into []string{"a", "b", "c"}.
// Expected: exact equality after round-trip.
func TestDecodeJSONField_StringSlice(t *testing.T) {
	// Neo4j returns string list properties as []interface{} with string elements.
	raw := []interface{}{"a", "b", "c"}
	var target []string
	if err := DecodeJSONField(raw, &target); err != nil {
		t.Fatalf("DecodeJSONField() error = %v", err)
	}
	expected := []string{"a", "b", "c"}
	if !slices.Equal(target, expected) {
		t.Errorf("target = %v, want %v", target, expected)
	}
}

// TestDecodeJSONField_EmptySlice verifies that an empty []interface{}{} from
// Neo4j is decoded into an empty typed slice (not nil).
// Expected: []float64{} (len 0, not nil).
func TestDecodeJSONField_EmptySlice(t *testing.T) {
	raw := []interface{}{}
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

// TestDecodeJSONField_Int64Slice verifies that []interface{}{int64(1), int64(2)}
// from Neo4j is decoded into []int{1, 2}. Neo4j returns all integers as int64.
// Expected: JSON round-trip converts int64 -> float64 (JSON number) -> int.
func TestDecodeJSONField_Int64Slice(t *testing.T) {
	raw := []interface{}{int64(1), int64(2), int64(3)}
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
	raw := make([]interface{}, 128)
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
	raw := []interface{}{"a", "b"}
	var target []string
	err := DecodeJSONField(raw, target) // pass by value, not pointer
	if err == nil {
		t.Error("DecodeJSONField(raw, nonPointer) should return error")
	}
}

// --- M3: Slice field round-trip via ReadNodeMaps + DecodeJSONField ---
//
// Tests the full decode path: ReadNodeMaps extracts property maps with
// []interface{} slice values, then DecodeJSONField converts them to typed
// Go slices. This mirrors what the generated FromResponse methods will do.

// TestSliceRoundTrip_Float64Vector verifies the full decode path for a
// TableCell node with a float64 vector: ReadNodeMaps -> DecodeJSONField.
// Expected: []interface{}{1.0, 2.0, 3.0} -> []float64{1.0, 2.0, 3.0}.
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
// Expected: []interface{}{"a", "b"} -> []string{"a", "b"}.
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
// Expected: []interface{}{} -> []float64{} (empty, not nil).
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
