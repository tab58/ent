// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cypher

import (
	"strings"
	"testing"
)

// --- M1: CRUD Cypher Pattern Tests ---
//
// These tests verify the Cypher query patterns that the generated Neo4j
// client code should produce. Each test constructs a Builder using the
// same pattern the templates would generate, and asserts the resulting
// Cypher string matches the expected pattern.
//
// Interface: cypher.Builder produces correct Cypher for each CRUD operation.

// TestCRUD_CreateSimple verifies the Cypher pattern for creating a simple
// entity with KSUID ID and a string field.
// Expected: CREATE (n:Business {id: $p0, name: $p1}) RETURN n {.*}
func TestCRUD_CreateSimple(t *testing.T) {
	b := New()
	idP := b.AddParam("ksuid-abc")
	nameP := b.AddParam("Acme Corp")
	b.Create("(n:Business {id: " + idP + ", name: " + nameP + "})")
	b.Return("n {.*}")

	query, params := b.Query()
	wantQuery := "CREATE (n:Business {id: $p0, name: $p1}) RETURN n {.*}"
	if query != wantQuery {
		t.Errorf("query = %q\nwant  = %q", query, wantQuery)
	}
	if params["p0"] != "ksuid-abc" {
		t.Errorf("params[p0] = %v, want ksuid-abc", params["p0"])
	}
	if params["p1"] != "Acme Corp" {
		t.Errorf("params[p1] = %v, want Acme Corp", params["p1"])
	}
}

// TestCRUD_CreateWithUniquenessGuard verifies the OPTIONAL MATCH pattern
// used when an entity has unique fields. The uniqueness guard prevents
// duplicate creation at the application level.
// Expected: OPTIONAL MATCH (existing:Business {name: $p0}) ...
//
//	WHERE existing IS NULL
//	CREATE (n:Business {id: $p1, name: $p0}) RETURN n {.*}
func TestCRUD_CreateWithUniquenessGuard(t *testing.T) {
	b := New()
	nameP := b.AddParam("Acme Corp")
	// Uniqueness guard: OPTIONAL MATCH + IS NULL check.
	b.Match("OPTIONAL MATCH (existing:Business {name: " + nameP + "})")
	b.Where("existing IS NULL")

	idP := b.AddParam("ksuid-abc")
	b.Create("(n:Business {id: " + idP + ", name: " + nameP + "})")
	b.Return("n {.*}")

	query, _ := b.Query()
	// Verify key patterns exist in the query.
	if !strings.Contains(query, "OPTIONAL MATCH") {
		t.Error("query should contain OPTIONAL MATCH for uniqueness guard")
	}
	if !strings.Contains(query, "existing IS NULL") {
		t.Error("query should contain 'existing IS NULL' check")
	}
	if !strings.Contains(query, "CREATE (n:Business") {
		t.Error("query should contain CREATE with Business label")
	}
	if !strings.Contains(query, "RETURN n {.*}") {
		t.Error("query should contain RETURN n {.*}")
	}
}

// TestCRUD_CreateTableCellWithSlices verifies that slice fields ([]float64,
// []string) are passed as native parameters to Cypher CREATE.
// Expected: CREATE (n:TableCell {id: $p0, value: $p1, vector: $p2, categories: $p3})
//
//	RETURN n {.*}
func TestCRUD_CreateTableCellWithSlices(t *testing.T) {
	b := New()
	idP := b.AddParam("ksuid-tc1")
	valueP := b.AddParam(3.14)
	vectorP := b.AddParam([]float64{1.0, 2.0, 3.0})
	catP := b.AddParam([]string{"a", "b"})
	b.Create("(n:TableCell {id: " + idP + ", value: " + valueP + ", vector: " + vectorP + ", categories: " + catP + "})")
	b.Return("n {.*}")

	query, params := b.Query()
	wantQuery := "CREATE (n:TableCell {id: $p0, value: $p1, vector: $p2, categories: $p3}) RETURN n {.*}"
	if query != wantQuery {
		t.Errorf("query = %q\nwant  = %q", query, wantQuery)
	}
	// Verify slice params are preserved as-is (Bolt protocol handles native slices).
	vec, ok := params["p2"].([]float64)
	if !ok {
		t.Fatalf("params[p2] type = %T, want []float64", params["p2"])
	}
	if len(vec) != 3 || vec[0] != 1.0 || vec[1] != 2.0 || vec[2] != 3.0 {
		t.Errorf("params[p2] = %v, want [1 2 3]", vec)
	}
	cats, ok := params["p3"].([]string)
	if !ok {
		t.Fatalf("params[p3] type = %T, want []string", params["p3"])
	}
	if len(cats) != 2 || cats[0] != "a" || cats[1] != "b" {
		t.Errorf("params[p3] = %v, want [a b]", cats)
	}
}

// TestCRUD_MatchByID verifies the Cypher pattern for querying a single
// entity by its KSUID ID.
// Expected: MATCH (n:Business) WHERE n.id = $p0 RETURN n {.*}
func TestCRUD_MatchByID(t *testing.T) {
	b := New()
	idP := b.AddParam("ksuid-abc")
	b.Match("(n:Business)")
	b.Where("n.id = " + idP)
	b.Return("n {.*}")

	query, params := b.Query()
	wantQuery := "MATCH (n:Business) WHERE n.id = $p0 RETURN n {.*}"
	if query != wantQuery {
		t.Errorf("query = %q\nwant  = %q", query, wantQuery)
	}
	if params["p0"] != "ksuid-abc" {
		t.Errorf("params[p0] = %v, want ksuid-abc", params["p0"])
	}
}

// TestCRUD_MatchAll verifies the Cypher pattern for querying all entities
// of a type.
// Expected: MATCH (n:Business) RETURN n {.*}
func TestCRUD_MatchAll(t *testing.T) {
	b := New()
	b.Match("(n:Business)")
	b.Return("n {.*}")

	query, _ := b.Query()
	wantQuery := "MATCH (n:Business) RETURN n {.*}"
	if query != wantQuery {
		t.Errorf("query = %q\nwant  = %q", query, wantQuery)
	}
}

// TestCRUD_MatchWithFieldPredicate verifies the Cypher pattern for querying
// entities with a WHERE condition on a field.
// Expected: MATCH (n:Business) WHERE n.name = $p0 RETURN n {.*}
func TestCRUD_MatchWithFieldPredicate(t *testing.T) {
	b := New()
	nameP := b.AddParam("Acme Corp")
	b.Match("(n:Business)")
	b.Where("n.name = " + nameP)
	b.Return("n {.*}")

	query, _ := b.Query()
	wantQuery := "MATCH (n:Business) WHERE n.name = $p0 RETURN n {.*}"
	if query != wantQuery {
		t.Errorf("query = %q\nwant  = %q", query, wantQuery)
	}
}

// TestCRUD_MatchCount verifies the Cypher pattern for counting entities.
// Expected: MATCH (n:Business) RETURN count(n)
func TestCRUD_MatchCount(t *testing.T) {
	b := New()
	b.Match("(n:Business)")
	b.Return("count(n)")

	query, _ := b.Query()
	wantQuery := "MATCH (n:Business) RETURN count(n)"
	if query != wantQuery {
		t.Errorf("query = %q\nwant  = %q", query, wantQuery)
	}
}

// TestCRUD_MatchWithPagination verifies SKIP/LIMIT are appended correctly.
// Expected: MATCH (n:Business) RETURN n {.*} ORDER BY n.name ASC SKIP 10 LIMIT 25
func TestCRUD_MatchWithPagination(t *testing.T) {
	b := New()
	b.Match("(n:Business)")
	b.Return("n {.*}")
	b.OrderBy("n.name ASC")
	b.Skip(10)
	b.Limit(25)

	query, _ := b.Query()
	wantQuery := "MATCH (n:Business) RETURN n {.*} ORDER BY n.name ASC SKIP 10 LIMIT 25"
	if query != wantQuery {
		t.Errorf("query = %q\nwant  = %q", query, wantQuery)
	}
}

// TestCRUD_UpdateSingleField verifies the Cypher pattern for updating a
// single field on an entity found by ID.
// Expected: MATCH (n:Business) WHERE n.id = $p0 SET n.name = $p1 RETURN n {.*}
func TestCRUD_UpdateSingleField(t *testing.T) {
	b := New()
	idP := b.AddParam("ksuid-abc")
	nameP := b.AddParam("New Name")
	b.Match("(n:Business)")
	b.Where("n.id = " + idP)
	b.Set("n.name = " + nameP)
	b.Return("n {.*}")

	query, params := b.Query()
	wantQuery := "MATCH (n:Business) WHERE n.id = $p0 SET n.name = $p1 RETURN n {.*}"
	if query != wantQuery {
		t.Errorf("query = %q\nwant  = %q", query, wantQuery)
	}
	if params["p1"] != "New Name" {
		t.Errorf("params[p1] = %v, want New Name", params["p1"])
	}
}

// TestCRUD_UpdateMultipleFields verifies SET with multiple field assignments.
// Expected: MATCH (n:TableCell) WHERE n.id = $p0
//
//	SET n.value = $p1, n.vector = $p2 RETURN n {.*}
func TestCRUD_UpdateMultipleFields(t *testing.T) {
	b := New()
	idP := b.AddParam("ksuid-tc1")
	valueP := b.AddParam(2.71)
	vectorP := b.AddParam([]float64{4.0, 5.0})
	b.Match("(n:TableCell)")
	b.Where("n.id = " + idP)
	b.Set("n.value = " + valueP)
	b.Set("n.vector = " + vectorP)
	b.Return("n {.*}")

	query, _ := b.Query()
	wantQuery := "MATCH (n:TableCell) WHERE n.id = $p0 SET n.value = $p1, n.vector = $p2 RETURN n {.*}"
	if query != wantQuery {
		t.Errorf("query = %q\nwant  = %q", query, wantQuery)
	}
}

// TestCRUD_UpdateClearOptionalField verifies REMOVE for clearing an
// optional field value.
// Expected: MATCH (n:Business) WHERE n.id = $p0 REMOVE n.description RETURN n {.*}
func TestCRUD_UpdateClearOptionalField(t *testing.T) {
	b := New()
	idP := b.AddParam("ksuid-abc")
	b.Match("(n:Business)")
	b.Where("n.id = " + idP)
	b.Remove("n.description")
	b.Return("n {.*}")

	query, _ := b.Query()
	wantQuery := "MATCH (n:Business) WHERE n.id = $p0 REMOVE n.description RETURN n {.*}"
	if query != wantQuery {
		t.Errorf("query = %q\nwant  = %q", query, wantQuery)
	}
}

// TestCRUD_UpdateBulk verifies the bulk update pattern that returns a count.
// Expected: MATCH (n:Business) WHERE n.name = $p0 SET n.name = $p1 RETURN count(n)
func TestCRUD_UpdateBulk(t *testing.T) {
	b := New()
	oldNameP := b.AddParam("Old Name")
	newNameP := b.AddParam("New Name")
	b.Match("(n:Business)")
	b.Where("n.name = " + oldNameP)
	b.Set("n.name = " + newNameP)
	b.Return("count(n)")

	query, _ := b.Query()
	wantQuery := "MATCH (n:Business) WHERE n.name = $p0 SET n.name = $p1 RETURN count(n)"
	if query != wantQuery {
		t.Errorf("query = %q\nwant  = %q", query, wantQuery)
	}
}

// TestCRUD_DetachDeleteByID verifies DETACH DELETE for removing an entity
// and all its relationships by ID.
// Expected: MATCH (n:Business) WHERE n.id = $p0 DETACH DELETE n RETURN count(n)
func TestCRUD_DetachDeleteByID(t *testing.T) {
	b := New()
	idP := b.AddParam("ksuid-abc")
	b.Match("(n:Business)")
	b.Where("n.id = " + idP)
	b.DetachDelete("n")
	b.Return("count(n)")

	query, params := b.Query()
	wantQuery := "MATCH (n:Business) WHERE n.id = $p0 DETACH DELETE n RETURN count(n)"
	if query != wantQuery {
		t.Errorf("query = %q\nwant  = %q", query, wantQuery)
	}
	if params["p0"] != "ksuid-abc" {
		t.Errorf("params[p0] = %v, want ksuid-abc", params["p0"])
	}
}

// TestCRUD_DeleteByPredicate verifies DETACH DELETE with a field predicate.
// Expected: MATCH (n:Business) WHERE n.name = $p0 DETACH DELETE n RETURN count(n)
func TestCRUD_DeleteByPredicate(t *testing.T) {
	b := New()
	nameP := b.AddParam("Acme Corp")
	b.Match("(n:Business)")
	b.Where("n.name = " + nameP)
	b.DetachDelete("n")
	b.Return("count(n)")

	query, _ := b.Query()
	wantQuery := "MATCH (n:Business) WHERE n.name = $p0 DETACH DELETE n RETURN count(n)"
	if query != wantQuery {
		t.Errorf("query = %q\nwant  = %q", query, wantQuery)
	}
}

// TestCRUD_CreateWithEdge verifies the Cypher pattern for creating an entity
// and linking it to an existing entity via a relationship edge.
// Expected: CREATE (n:Business {id: $p0, name: $p1})
//
//	MATCH (m:Document) WHERE m.id = $p2
//	CREATE (n)-[:BUSINESS_HAS_DOCUMENT]->(m) RETURN n {.*}
func TestCRUD_CreateWithEdge(t *testing.T) {
	b := New()
	idP := b.AddParam("ksuid-biz")
	nameP := b.AddParam("Acme")
	b.Create("(n:Business {id: " + idP + ", name: " + nameP + "})")

	// WITH n MATCH + CREATE for the edge.
	docP := b.AddParam("ksuid-doc")
	b.Match("WITH n MATCH (m:Document) WHERE m.id = " + docP)
	b.Create("(n)-[:BUSINESS_HAS_DOCUMENT]->(m)")
	b.Return("n {.*}")

	query, params := b.Query()
	if !strings.Contains(query, "CREATE (n:Business") {
		t.Error("query should contain CREATE (n:Business ...)")
	}
	if !strings.Contains(query, "BUSINESS_HAS_DOCUMENT") {
		t.Error("query should contain BUSINESS_HAS_DOCUMENT relationship type")
	}
	if !strings.Contains(query, "RETURN n {.*}") {
		t.Error("query should end with RETURN n {.*}")
	}
	if params["p2"] != "ksuid-doc" {
		t.Errorf("params[p2] = %v, want ksuid-doc", params["p2"])
	}
}
