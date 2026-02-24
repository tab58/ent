// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package neo4j

import (
	"fmt"
	"testing"

	"entgo.io/ent/dialect/neo4j/cypher"
	ndriver "github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

// --- L2: Edge cases for delete and empty results ---
//
// These tests verify correct behavior for edge cases in delete operations,
// empty result sets, and edge traversals returning no results.
//
// Interface: Response.ReadNodeMaps, Response.ReadInt for edge case inputs.

// TestEdgeCase_DetachDeleteWithEdges verifies that the Cypher pattern for
// DETACH DELETE includes the DETACH keyword, which removes the node AND all
// its relationships in a single operation.
// Expected: MATCH (n:Business) WHERE n.id = $p0 DETACH DELETE n RETURN count(n)
func TestEdgeCase_DetachDeleteWithEdges(t *testing.T) {
	b := cypher.New()
	idP := b.AddParam("ksuid-detach-1")
	b.Match("(n:Business)")
	b.Where("n.id = " + idP)
	b.DetachDelete("n")
	b.Return("count(n)")

	query, _ := b.Query()
	wantQuery := "MATCH (n:Business) WHERE n.id = $p0 DETACH DELETE n RETURN count(n)"
	if query != wantQuery {
		t.Errorf("query = %q\nwant  = %q", query, wantQuery)
	}
}

// TestEdgeCase_DetachDeleteWithMultipleEdges verifies the DETACH DELETE
// pattern for a node connected to multiple relationship types.
// Expected: DETACH DELETE removes node regardless of number of edges.
func TestEdgeCase_DetachDeleteWithMultipleEdges(t *testing.T) {
	// A Business with documents, categories, and years edges should
	// still use the same DETACH DELETE pattern.
	labels := []string{"Business", "Document", "Category"}
	for _, label := range labels {
		t.Run(label, func(t *testing.T) {
			b := cypher.New()
			idP := b.AddParam("ksuid-multi-edge")
			b.Match(fmt.Sprintf("(n:%s)", label))
			b.Where("n.id = " + idP)
			b.DetachDelete("n")
			b.Return("count(n)")

			query, _ := b.Query()
			wantQuery := fmt.Sprintf(
				"MATCH (n:%s) WHERE n.id = $p0 DETACH DELETE n RETURN count(n)", label)
			if query != wantQuery {
				t.Errorf("query = %q\nwant  = %q", query, wantQuery)
			}
		})
	}
}

// TestEdgeCase_QueryNoResults_ReadNodeMaps verifies that querying for entities
// that don't exist returns an empty slice (not nil, not error).
// Expected: ReadNodeMaps returns []map[string]any{} with len 0 and nil error.
func TestEdgeCase_QueryNoResults_ReadNodeMaps(t *testing.T) {
	// Simulate empty query result (no matching nodes).
	records := []*ndriver.Record{}
	r := NewResponse(records, []string{"n"})
	maps, err := r.ReadNodeMaps()
	if err != nil {
		t.Fatalf("ReadNodeMaps() error = %v (should succeed for empty results)", err)
	}
	if maps == nil {
		t.Error("ReadNodeMaps() returned nil, want empty slice")
	}
	if len(maps) != 0 {
		t.Errorf("ReadNodeMaps() returned %d maps, want 0", len(maps))
	}
}

// TestEdgeCase_QueryNoResults_ReadInt verifies that a count query returning
// zero is handled correctly (0, not error).
// Expected: ReadInt returns (0, nil) for count(n) = 0.
func TestEdgeCase_QueryNoResults_ReadInt(t *testing.T) {
	records := []*ndriver.Record{
		{Keys: []string{"count(n)"}, Values: []any{int64(0)}},
	}
	r := NewResponse(records, []string{"count(n)"})
	count, err := r.ReadInt()
	if err != nil {
		t.Fatalf("ReadInt() error = %v", err)
	}
	if count != 0 {
		t.Errorf("ReadInt() = %d, want 0", count)
	}
}

// TestEdgeCase_EdgeTraversalNoConnected verifies that traversing an edge
// from a node with no connected nodes returns an empty slice.
// Expected: empty record set -> ReadNodeMaps returns empty slice.
func TestEdgeCase_EdgeTraversalNoConnected(t *testing.T) {
	// Simulate: MATCH (n:Business {id: $p0})-[:BUSINESS_HAS_DOCUMENT]->(m:Document)
	// where no documents are connected. Result: empty records.
	records := []*ndriver.Record{}
	r := NewResponse(records, []string{"m"})
	maps, err := r.ReadNodeMaps()
	if err != nil {
		t.Fatalf("ReadNodeMaps() error = %v", err)
	}
	if len(maps) != 0 {
		t.Errorf("edge traversal with no connected nodes: got %d results, want 0", len(maps))
	}
}

// TestEdgeCase_EdgeTraversalCypherPattern verifies the Cypher pattern for
// traversing an edge from a node with no connections is well-formed.
// Expected: MATCH (n:Business {id: $p0})-[:BUSINESS_HAS_DOCUMENT]->(m:Document) RETURN m {.*}
func TestEdgeCase_EdgeTraversalCypherPattern(t *testing.T) {
	b := cypher.New()
	idP := b.AddParam("ksuid-orphan")
	pattern := fmt.Sprintf("(n:Business {id: %s})-[:BUSINESS_HAS_DOCUMENT]->(m:Document)", idP)
	b.Match(pattern)
	b.Return("m {.*}")

	query, _ := b.Query()
	wantQuery := "MATCH (n:Business {id: $p0})-[:BUSINESS_HAS_DOCUMENT]->(m:Document) RETURN m {.*}"
	if query != wantQuery {
		t.Errorf("query = %q\nwant  = %q", query, wantQuery)
	}
}

// TestEdgeCase_DeleteNonexistentEntity verifies that DETACH DELETE on a
// non-existent entity produces a count of 0 (not an error).
// This tests the response handling for a delete that matches nothing.
// Expected: ReadInt returns 0.
func TestEdgeCase_DeleteNonexistentEntity(t *testing.T) {
	// Simulate: MATCH (n:Business) WHERE n.id = $p0 DETACH DELETE n RETURN count(n)
	// where no Business with that ID exists. Neo4j returns count(n) = 0.
	records := []*ndriver.Record{
		{Keys: []string{"count(n)"}, Values: []any{int64(0)}},
	}
	r := NewResponse(records, []string{"count(n)"})
	count, err := r.ReadInt()
	if err != nil {
		t.Fatalf("ReadInt() error = %v", err)
	}
	if count != 0 {
		t.Errorf("delete non-existent entity: ReadInt() = %d, want 0", count)
	}
}

// TestEdgeCase_ReadSingle_NoResults verifies ReadSingle returns error when
// no records match (e.g., query-by-ID for a deleted entity).
// Expected: error "no records in response".
func TestEdgeCase_ReadSingle_NoResults(t *testing.T) {
	r := NewResponse([]*ndriver.Record{}, []string{"n"})
	_, err := r.ReadSingle()
	if err == nil {
		t.Error("ReadSingle() on empty results should return error")
	}
}

// TestEdgeCase_BulkDeleteCount verifies that a bulk delete returns the
// correct count of deleted nodes.
// Expected: DETACH DELETE matching multiple nodes returns accurate count.
func TestEdgeCase_BulkDeleteCount(t *testing.T) {
	// Simulate: MATCH (n:Business) WHERE n.name = $p0 DETACH DELETE n RETURN count(n)
	// where 5 businesses matched.
	records := []*ndriver.Record{
		{Keys: []string{"count(n)"}, Values: []any{int64(5)}},
	}
	r := NewResponse(records, []string{"count(n)"})
	count, err := r.ReadInt()
	if err != nil {
		t.Fatalf("ReadInt() error = %v", err)
	}
	if count != 5 {
		t.Errorf("bulk delete count = %d, want 5", count)
	}
}

// TestEdgeCase_EmptyNodeMap verifies that a node with no properties (just id)
// is handled correctly by ReadNodeMaps.
// Expected: map with only "id" key.
func TestEdgeCase_EmptyNodeMap(t *testing.T) {
	records := []*ndriver.Record{
		{
			Keys:   []string{"n"},
			Values: []any{map[string]any{"id": "ksuid-empty-props"}},
		},
	}
	r := NewResponse(records, []string{"n"})
	maps, err := r.ReadNodeMaps()
	if err != nil {
		t.Fatalf("ReadNodeMaps() error = %v", err)
	}
	if len(maps) != 1 {
		t.Fatalf("ReadNodeMaps() returned %d maps, want 1", len(maps))
	}
	m := maps[0]
	if m["id"] != "ksuid-empty-props" {
		t.Errorf("id = %v, want ksuid-empty-props", m["id"])
	}
	// Should have only the id key.
	if len(m) != 1 {
		t.Errorf("map has %d keys, want 1 (id only)", len(m))
	}
}

// TestEdgeCase_ReadNodeMaps_EmptyValues verifies that ReadNodeMaps returns an
// error when a record has keys but no values (malformed response).
// Expected: error "record has no values".
func TestEdgeCase_ReadNodeMaps_EmptyValues(t *testing.T) {
	records := []*ndriver.Record{
		{Keys: []string{"n"}, Values: []any{}},
	}
	r := NewResponse(records, []string{"n"})
	_, err := r.ReadNodeMaps()
	if err == nil {
		t.Error("ReadNodeMaps() on record with empty Values should return error")
	}
}

// TestEdgeCase_ReadSingle_EmptyValues verifies that ReadSingle returns an
// error when the first record has keys but no values (malformed response).
// Expected: error "record has no values".
func TestEdgeCase_ReadSingle_EmptyValues(t *testing.T) {
	records := []*ndriver.Record{
		{Keys: []string{"n"}, Values: []any{}},
	}
	r := NewResponse(records, []string{"n"})
	_, err := r.ReadSingle()
	if err == nil {
		t.Error("ReadSingle() on record with empty Values should return error")
	}
}
