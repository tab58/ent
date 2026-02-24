// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package neo4j

import (
	"context"
	"fmt"
	"os"
	"slices"
	"testing"

	"entgo.io/ent/dialect/neo4j/cypher"
	ndriver "github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

// --- L1: Integration test against live Neo4j ---
//
// These tests require a running Neo4j instance. Set the NEO4J_TEST_URI
// environment variable to run them (e.g., NEO4J_TEST_URI=neo4j://localhost:7687).
// Optional: NEO4J_TEST_USER, NEO4J_TEST_PASS (default: neo4j/neo4j).
//
// Interface: Driver.Exec, Driver.Query with live Neo4j using cypher.Builder.

// skipIfNoNeo4j skips the test if NEO4J_TEST_URI is not set.
func skipIfNoNeo4j(t *testing.T) {
	t.Helper()
	if os.Getenv("NEO4J_TEST_URI") == "" {
		t.Skip("NEO4J_TEST_URI not set, skipping live integration test")
	}
}

// liveDriver creates a Driver connected to a live Neo4j instance.
// Cleans all nodes before returning to ensure a fresh state.
func liveDriver(t *testing.T) *Driver {
	t.Helper()
	skipIfNoNeo4j(t)

	uri := os.Getenv("NEO4J_TEST_URI")
	user := os.Getenv("NEO4J_TEST_USER")
	if user == "" {
		user = "neo4j"
	}
	pass := os.Getenv("NEO4J_TEST_PASS")
	if pass == "" {
		pass = "neo4j"
	}

	db, err := ndriver.NewDriver(uri, ndriver.BasicAuth(user, pass, ""))
	if err != nil {
		t.Fatalf("NewDriverWithContext(%q) error = %v", uri, err)
	}
	drv := NewDriver(db, "neo4j")

	// Clean all nodes for test isolation.
	ctx := context.Background()
	res := &Response{}
	if err := drv.Exec(ctx, "MATCH (n) DETACH DELETE n", map[string]any{}, res); err != nil {
		t.Fatalf("cleanup DETACH DELETE: %v", err)
	}

	t.Cleanup(func() {
		// Clean up after test.
		res := &Response{}
		_ = drv.Exec(context.Background(), "MATCH (n) DETACH DELETE n", map[string]any{}, res)
		drv.Close()
	})

	return drv
}

// TestLiveNeo4j_CreateAndQueryByID tests the full CREATE + MATCH-by-ID cycle
// against a live Neo4j instance.
// Expected: CREATE a Business node, then MATCH it back by its KSUID id.
func TestLiveNeo4j_CreateAndQueryByID(t *testing.T) {
	drv := liveDriver(t)
	ctx := context.Background()

	// CREATE a Business node.
	b := cypher.New()
	idP := b.AddParam("ksuid-live-biz-1")
	nameP := b.AddParam("Live Corp")
	b.Create(fmt.Sprintf("(n:Business {id: %s, name: %s})", idP, nameP))
	b.Return("n {.*}")
	query, params := b.Query()

	createRes := &Response{}
	if err := drv.Exec(ctx, query, params, createRes); err != nil {
		t.Fatalf("Exec CREATE: %v", err)
	}

	// MATCH back by ID.
	b2 := cypher.New()
	idP2 := b2.AddParam("ksuid-live-biz-1")
	b2.Match("(n:Business)")
	b2.Where("n.id = " + idP2)
	b2.Return("n {.*}")
	query2, params2 := b2.Query()

	queryRes := &Response{}
	if err := drv.Query(ctx, query2, params2, queryRes); err != nil {
		t.Fatalf("Query MATCH: %v", err)
	}
	m, err := queryRes.ReadSingle()
	if err != nil {
		t.Fatalf("ReadSingle: %v", err)
	}
	if m["id"] != "ksuid-live-biz-1" {
		t.Errorf("id = %v, want ksuid-live-biz-1", m["id"])
	}
	if m["name"] != "Live Corp" {
		t.Errorf("name = %v, want Live Corp", m["name"])
	}
}

// TestLiveNeo4j_UpdateField tests SET on a live node.
// Expected: CREATE a node, SET its name, verify the updated value.
func TestLiveNeo4j_UpdateField(t *testing.T) {
	drv := liveDriver(t)
	ctx := context.Background()

	// CREATE.
	b := cypher.New()
	idP := b.AddParam("ksuid-live-biz-2")
	nameP := b.AddParam("Original")
	b.Create(fmt.Sprintf("(n:Business {id: %s, name: %s})", idP, nameP))
	b.Return("n {.*}")
	q, p := b.Query()
	if err := drv.Exec(ctx, q, p, &Response{}); err != nil {
		t.Fatalf("CREATE: %v", err)
	}

	// UPDATE.
	b2 := cypher.New()
	idP2 := b2.AddParam("ksuid-live-biz-2")
	newNameP := b2.AddParam("Updated")
	b2.Match("(n:Business)")
	b2.Where("n.id = " + idP2)
	b2.Set("n.name = " + newNameP)
	b2.Return("n {.*}")
	q2, p2 := b2.Query()
	updateRes := &Response{}
	if err := drv.Exec(ctx, q2, p2, updateRes); err != nil {
		t.Fatalf("UPDATE: %v", err)
	}
	m, err := updateRes.ReadSingle()
	if err != nil {
		t.Fatalf("ReadSingle after update: %v", err)
	}
	if m["name"] != "Updated" {
		t.Errorf("name after update = %v, want Updated", m["name"])
	}
}

// TestLiveNeo4j_DetachDelete tests DETACH DELETE on a live node.
// Expected: CREATE a node, DETACH DELETE it, verify count is 0.
func TestLiveNeo4j_DetachDelete(t *testing.T) {
	drv := liveDriver(t)
	ctx := context.Background()

	// CREATE.
	b := cypher.New()
	idP := b.AddParam("ksuid-live-del-1")
	nameP := b.AddParam("ToDelete")
	b.Create(fmt.Sprintf("(n:Business {id: %s, name: %s})", idP, nameP))
	b.Return("n {.*}")
	q, p := b.Query()
	if err := drv.Exec(ctx, q, p, &Response{}); err != nil {
		t.Fatalf("CREATE: %v", err)
	}

	// DETACH DELETE.
	b2 := cypher.New()
	idP2 := b2.AddParam("ksuid-live-del-1")
	b2.Match("(n:Business)")
	b2.Where("n.id = " + idP2)
	b2.DetachDelete("n")
	b2.Return("count(n)")
	q2, p2 := b2.Query()
	if err := drv.Exec(ctx, q2, p2, &Response{}); err != nil {
		t.Fatalf("DETACH DELETE: %v", err)
	}

	// Verify gone.
	b3 := cypher.New()
	b3.Match("(n:Business)")
	b3.Return("count(n)")
	q3, p3 := b3.Query()
	countRes := &Response{}
	if err := drv.Query(ctx, q3, p3, countRes); err != nil {
		t.Fatalf("COUNT query: %v", err)
	}
	count, err := countRes.ReadInt()
	if err != nil {
		t.Fatalf("ReadInt: %v", err)
	}
	if count != 0 {
		t.Errorf("count after delete = %d, want 0", count)
	}
}

// TestLiveNeo4j_EdgeCreationAndTraversal tests creating two nodes and an edge,
// then traversing the edge to find the connected node.
// Expected: CREATE Business and Document, link via BUSINESS_HAS_DOCUMENT,
// traverse forward and verify.
func TestLiveNeo4j_EdgeCreationAndTraversal(t *testing.T) {
	drv := liveDriver(t)
	ctx := context.Background()

	// CREATE Business.
	b := cypher.New()
	bizID := b.AddParam("ksuid-live-biz-edge")
	bizName := b.AddParam("EdgeCorp")
	b.Create(fmt.Sprintf("(n:Business {id: %s, name: %s})", bizID, bizName))
	b.Return("n {.*}")
	q, p := b.Query()
	if err := drv.Exec(ctx, q, p, &Response{}); err != nil {
		t.Fatalf("CREATE Business: %v", err)
	}

	// CREATE Document.
	b2 := cypher.New()
	docID := b2.AddParam("ksuid-live-doc-edge")
	docName := b2.AddParam("EdgeDoc")
	b2.Create(fmt.Sprintf("(n:Document {id: %s, name: %s})", docID, docName))
	b2.Return("n {.*}")
	q2, p2 := b2.Query()
	if err := drv.Exec(ctx, q2, p2, &Response{}); err != nil {
		t.Fatalf("CREATE Document: %v", err)
	}

	// CREATE edge: Business -[:BUSINESS_HAS_DOCUMENT]-> Document.
	b3 := cypher.New()
	srcP := b3.AddParam("ksuid-live-biz-edge")
	tgtP := b3.AddParam("ksuid-live-doc-edge")
	b3.Match("(n:Business)")
	b3.Where("n.id = " + srcP)
	b3.Match(fmt.Sprintf("WITH n MATCH (m:Document) WHERE m.id = %s", tgtP))
	b3.Create("(n)-[:BUSINESS_HAS_DOCUMENT]->(m)")
	b3.Return("n {.*}")
	q3, p3 := b3.Query()
	if err := drv.Exec(ctx, q3, p3, &Response{}); err != nil {
		t.Fatalf("CREATE edge: %v", err)
	}

	// TRAVERSE: Business -> Document via edge.
	b4 := cypher.New()
	travP := b4.AddParam("ksuid-live-biz-edge")
	b4.Match(fmt.Sprintf("(n:Business {id: %s})-[:BUSINESS_HAS_DOCUMENT]->(m:Document)", travP))
	b4.Return("m {.*}")
	q4, p4 := b4.Query()
	travRes := &Response{}
	if err := drv.Query(ctx, q4, p4, travRes); err != nil {
		t.Fatalf("TRAVERSE query: %v", err)
	}
	m, err := travRes.ReadSingle()
	if err != nil {
		t.Fatalf("ReadSingle from traversal: %v", err)
	}
	if m["name"] != "EdgeDoc" {
		t.Errorf("traversed document name = %v, want EdgeDoc", m["name"])
	}
}

// TestLiveNeo4j_SliceFieldRoundTrip tests creating a TableCell with slice fields
// (vector []float64, categories []string) and reading them back.
// Expected: slices are preserved through the CREATE + MATCH + DecodeJSONField path.
func TestLiveNeo4j_SliceFieldRoundTrip(t *testing.T) {
	drv := liveDriver(t)
	ctx := context.Background()

	// CREATE TableCell with slice fields.
	b := cypher.New()
	idP := b.AddParam("ksuid-live-tc-1")
	valueP := b.AddParam(3.14)
	vectorP := b.AddParam([]float64{1.0, 2.0, 3.0})
	catP := b.AddParam([]string{"alpha", "beta"})
	b.Create(fmt.Sprintf("(n:TableCell {id: %s, value: %s, vector: %s, categories: %s})",
		idP, valueP, vectorP, catP))
	b.Return("n {.*}")
	q, p := b.Query()
	if err := drv.Exec(ctx, q, p, &Response{}); err != nil {
		t.Fatalf("CREATE TableCell: %v", err)
	}

	// MATCH back.
	b2 := cypher.New()
	idP2 := b2.AddParam("ksuid-live-tc-1")
	b2.Match("(n:TableCell)")
	b2.Where("n.id = " + idP2)
	b2.Return("n {.*}")
	q2, p2 := b2.Query()
	queryRes := &Response{}
	if err := drv.Query(ctx, q2, p2, queryRes); err != nil {
		t.Fatalf("MATCH TableCell: %v", err)
	}
	m, err := queryRes.ReadSingle()
	if err != nil {
		t.Fatalf("ReadSingle: %v", err)
	}

	// Decode vector via DecodeJSONField.
	var vector []float64
	if err := DecodeJSONField(m["vector"], &vector); err != nil {
		t.Fatalf("DecodeJSONField(vector): %v", err)
	}
	expectedVec := []float64{1.0, 2.0, 3.0}
	if !slices.Equal(vector, expectedVec) {
		t.Errorf("vector = %v, want %v", vector, expectedVec)
	}

	// Decode categories via DecodeJSONField.
	var categories []string
	if err := DecodeJSONField(m["categories"], &categories); err != nil {
		t.Fatalf("DecodeJSONField(categories): %v", err)
	}
	if len(categories) != 2 || categories[0] != "alpha" || categories[1] != "beta" {
		t.Errorf("categories = %v, want [alpha beta]", categories)
	}
}

// TestLiveNeo4j_AllEntityTypes tests creating all 6 entity types and verifying
// they can be queried back.
// Expected: Business, Document, Category, Year, Table, TableCell nodes all exist.
func TestLiveNeo4j_AllEntityTypes(t *testing.T) {
	drv := liveDriver(t)
	ctx := context.Background()

	entities := []struct {
		label string
		props string
		id    string
	}{
		{"Business", "id: %s, name: %s", "ksuid-all-biz"},
		{"Document", "id: %s, name: %s", "ksuid-all-doc"},
		{"Category", "id: %s, name: %s", "ksuid-all-cat"},
		{"Year", "id: %s, value: %s", "ksuid-all-yr"},
		{"Table", "id: %s, name: %s", "ksuid-all-tbl"},
		{"TableCell", "id: %s, value: %s", "ksuid-all-tc"},
	}

	for _, e := range entities {
		b := cypher.New()
		idP := b.AddParam(e.id)
		valP := b.AddParam("test-value")
		b.Create(fmt.Sprintf("(n:%s {"+e.props+"})", e.label, idP, valP))
		b.Return("n {.*}")
		q, p := b.Query()
		if err := drv.Exec(ctx, q, p, &Response{}); err != nil {
			t.Fatalf("CREATE %s: %v", e.label, err)
		}
	}

	// Verify all 6 types exist.
	for _, e := range entities {
		b := cypher.New()
		idP := b.AddParam(e.id)
		b.Match(fmt.Sprintf("(n:%s)", e.label))
		b.Where("n.id = " + idP)
		b.Return("n {.*}")
		q, p := b.Query()
		res := &Response{}
		if err := drv.Query(ctx, q, p, res); err != nil {
			t.Fatalf("MATCH %s: %v", e.label, err)
		}
		m, err := res.ReadSingle()
		if err != nil {
			t.Fatalf("ReadSingle %s: %v", e.label, err)
		}
		if m["id"] != e.id {
			t.Errorf("%s id = %v, want %s", e.label, m["id"], e.id)
		}
	}
}
