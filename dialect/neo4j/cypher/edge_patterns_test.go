// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cypher

import (
	"fmt"
	"strings"
	"testing"
)

// --- M2: Edge Traversal Cypher Pattern Tests ---
//
// These tests verify the Cypher patterns for all 9 O2M edge traversals
// in the megatron-api entity model. Each edge produces a MATCH pattern with:
// - Source node label
// - Relationship type in SCREAMING_SNAKE_CASE
// - Forward direction (->)
// - Target node label
//
// Interface: cypher.Builder.Match() with edge traversal patterns.

// edgeTraversal defines the expected Cypher pattern for a single edge.
type edgeTraversal struct {
	sourceLabel string // e.g., "Business"
	relType     string // e.g., "BUSINESS_HAS_DOCUMENT"
	targetLabel string // e.g., "Document"
}

// megatronEdges returns all 9 O2M edges in the megatron-api entity model.
func megatronEdges() []edgeTraversal {
	return []edgeTraversal{
		{"Business", "BUSINESS_HAS_DOCUMENT", "Document"},
		{"Business", "BUSINESS_HAS_CATEGORY", "Category"},
		{"Business", "BUSINESS_HAS_YEAR", "Year"},
		{"Document", "DOCUMENT_HAS_CATEGORY", "Category"},
		{"Document", "DOCUMENT_HAS_YEAR", "Year"},
		{"Document", "DOCUMENT_HAS_TABLE", "Table"},
		{"Category", "CATEGORY_HAS_TABLE_CELL", "TableCell"},
		{"Table", "TABLE_HAS_TABLE_CELL", "TableCell"},
		{"Year", "YEAR_HAS_TABLE_CELL", "TableCell"},
	}
}

// TestEdgeTraversal_ForwardDirection verifies that each of the 9 O2M edges
// produces a forward MATCH pattern: (n:Source)-[:REL]->(m:Target).
// Expected: MATCH (n:Source {id: $p0})-[:REL]->(m:Target) RETURN m {.*}
func TestEdgeTraversal_ForwardDirection(t *testing.T) {
	for _, edge := range megatronEdges() {
		name := fmt.Sprintf("%s->%s", edge.sourceLabel, edge.targetLabel)
		t.Run(name, func(t *testing.T) {
			b := New()
			idP := b.AddParam("ksuid-source")
			pattern := fmt.Sprintf("(n:%s {id: %s})-[:%s]->(m:%s)",
				edge.sourceLabel, idP, edge.relType, edge.targetLabel)
			b.Match(pattern)
			b.Return("m {.*}")

			query, _ := b.Query()
			wantQuery := fmt.Sprintf("MATCH (n:%s {id: $p0})-[:%s]->(m:%s) RETURN m {.*}",
				edge.sourceLabel, edge.relType, edge.targetLabel)
			if query != wantQuery {
				t.Errorf("query = %q\nwant  = %q", query, wantQuery)
			}
		})
	}
}

// TestEdgeTraversal_InverseDirection verifies that inverse edges (M2O) use
// the reverse arrow pattern: (n:Target)<-[:REL]-(m:Source).
// Expected: MATCH (n:Document {id: $p0})<-[:BUSINESS_HAS_DOCUMENT]-(m:Business)
//
//	RETURN m {.*}
func TestEdgeTraversal_InverseDirection(t *testing.T) {
	// Test a subset of inverse traversals (M2O direction).
	inverseEdges := []edgeTraversal{
		// Document -> its parent Business via inverse of BUSINESS_HAS_DOCUMENT.
		{"Document", "BUSINESS_HAS_DOCUMENT", "Business"},
		// Category -> its parent Business via inverse of BUSINESS_HAS_CATEGORY.
		{"Category", "BUSINESS_HAS_CATEGORY", "Business"},
		// TableCell -> its parent Category via inverse of CATEGORY_HAS_TABLE_CELL.
		{"TableCell", "CATEGORY_HAS_TABLE_CELL", "Category"},
	}
	for _, edge := range inverseEdges {
		name := fmt.Sprintf("%s<-%s", edge.sourceLabel, edge.targetLabel)
		t.Run(name, func(t *testing.T) {
			b := New()
			idP := b.AddParam("ksuid-child")
			// Inverse pattern uses <-[:REL]- direction.
			pattern := fmt.Sprintf("(n:%s {id: %s})<-[:%s]-(m:%s)",
				edge.sourceLabel, idP, edge.relType, edge.targetLabel)
			b.Match(pattern)
			b.Return("m {.*}")

			query, _ := b.Query()
			wantQuery := fmt.Sprintf("MATCH (n:%s {id: $p0})<-[:%s]-(m:%s) RETURN m {.*}",
				edge.sourceLabel, edge.relType, edge.targetLabel)
			if query != wantQuery {
				t.Errorf("query = %q\nwant  = %q", query, wantQuery)
			}
		})
	}
}

// TestEdgeTraversal_ExistenceCheck verifies the HasEdge predicate pattern
// that checks if a node has any connected nodes via a given relationship.
// Expected: MATCH (n:Business)-[:BUSINESS_HAS_DOCUMENT]->()
func TestEdgeTraversal_ExistenceCheck(t *testing.T) {
	for _, edge := range megatronEdges() {
		name := fmt.Sprintf("Has_%s_%s", edge.sourceLabel, edge.relType)
		t.Run(name, func(t *testing.T) {
			b := New()
			b.Match("(n:" + edge.sourceLabel + ")")
			// HasEdge predicate appends a MATCH with anonymous target.
			b.Match(fmt.Sprintf("(n)-[:%s]->()", edge.relType))
			b.Return("n {.*}")

			query, _ := b.Query()
			if query == "" {
				t.Error("query should not be empty")
			}
			// Verify the edge existence pattern is present.
			wantPattern := fmt.Sprintf("[:%s]->()", edge.relType)
			if !strings.Contains(query, wantPattern) {
				t.Errorf("query = %q\nmissing pattern %q", query, wantPattern)
			}
		})
	}
}

// TestEdgeTraversal_EdgeCreation verifies the CREATE pattern for establishing
// a new relationship between two existing nodes.
// Expected: MATCH (n:Business) WHERE n.id = $p0
//
//	WITH n MATCH (m:Document) WHERE m.id = $p1
//	CREATE (n)-[:BUSINESS_HAS_DOCUMENT]->(m) RETURN n {.*}
func TestEdgeTraversal_EdgeCreation(t *testing.T) {
	edges := megatronEdges()
	for _, edge := range edges {
		name := fmt.Sprintf("Create_%s", edge.relType)
		t.Run(name, func(t *testing.T) {
			b := New()
			srcP := b.AddParam("ksuid-source")
			tgtP := b.AddParam("ksuid-target")
			b.Match(fmt.Sprintf("(n:%s)", edge.sourceLabel))
			b.Where("n.id = " + srcP)
			b.Match(fmt.Sprintf("WITH n MATCH (m:%s) WHERE m.id = %s", edge.targetLabel, tgtP))
			b.Create(fmt.Sprintf("(n)-[:%s]->(m)", edge.relType))
			b.Return("n {.*}")

			query, _ := b.Query()
			if !strings.Contains(query, edge.relType) {
				t.Errorf("query missing relationship type %q", edge.relType)
			}
			if !strings.Contains(query, "CREATE (n)-[:"+edge.relType+"]->(m)") {
				t.Errorf("query missing CREATE pattern for %q", edge.relType)
			}
		})
	}
}

// TestEdgeTraversal_EdgeDeletion verifies the DELETE pattern for removing
// a specific relationship between two nodes.
// Expected: MATCH (n:Business)-[r:BUSINESS_HAS_DOCUMENT]->(m)
//
//	WHERE m.id = $p1 DELETE r
func TestEdgeTraversal_EdgeDeletion(t *testing.T) {
	edges := megatronEdges()
	for _, edge := range edges[:3] { // Test first 3 edges.
		name := fmt.Sprintf("Delete_%s", edge.relType)
		t.Run(name, func(t *testing.T) {
			b := New()
			srcP := b.AddParam("ksuid-source")
			tgtP := b.AddParam("ksuid-target")
			b.Match(fmt.Sprintf("(n:%s)", edge.sourceLabel))
			b.Where("n.id = " + srcP)
			b.Match(fmt.Sprintf("WITH n MATCH (n)-[r:%s]->(m) WHERE m.id = %s", edge.relType, tgtP))
			b.Delete("r")
			b.Return("count(r)")

			query, _ := b.Query()
			if !strings.Contains(query, "DELETE r") {
				t.Errorf("query missing DELETE r")
			}
			if !strings.Contains(query, edge.relType) {
				t.Errorf("query missing relationship type %q", edge.relType)
			}
		})
	}
}

// TestEdgeTraversal_RelationshipNameFormat verifies all relationship type
// names follow SCREAMING_SNAKE_CASE convention.
// Expected: all names match pattern [A-Z][A-Z0-9_]*
func TestEdgeTraversal_RelationshipNameFormat(t *testing.T) {
	for _, edge := range megatronEdges() {
		t.Run(edge.relType, func(t *testing.T) {
			for _, c := range edge.relType {
				if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
					t.Errorf("relationship type %q contains invalid char %q (expected SCREAMING_SNAKE_CASE)", edge.relType, string(c))
					break
				}
			}
		})
	}
}

