// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cypher

import "testing"

// --- Filtered traversal tests ---
//
// These mirror the builder call sequence the generated code produces for
// A.Query().Where(...).QueryB().Where(...): the parent builder gets the
// parent predicates, the traversal appends MATCH + WITH m AS n, and the
// child predicates are applied after the rebind. WHERE must be emitted
// positionally so each predicate filters the right node.

// traverse appends one traversal segment the way query.tmpl emits it.
func traverse(b *Builder, pattern string) *Builder {
	b.Match(pattern)
	b.With("m AS n")
	return b
}

func TestTraversal_PredicateOnStartOnly(t *testing.T) {
	b := New()
	b.Match("(n:business)")
	p0 := b.AddParam("biz-1")
	b.Where("n.id = " + p0)
	traverse(b, "(n)-[:business_documents]->(m:document)")
	b.Return("n {.*}")

	got, _ := b.Query()
	want := "MATCH (n:business) WHERE n.id = $p0 " +
		"MATCH (n)-[:business_documents]->(m:document) WITH m AS n " +
		"RETURN n {.*}"
	if got != want {
		t.Errorf("Query() = %q\nwant  = %q", got, want)
	}
}

func TestTraversal_PredicateOnTargetOnly(t *testing.T) {
	b := New()
	b.Match("(n:business)")
	traverse(b, "(n)-[:business_documents]->(m:document)")
	p0 := b.AddParam("doc-1")
	b.Where("n.id = " + p0)
	b.Return("n {.*}")

	got, _ := b.Query()
	want := "MATCH (n:business) " +
		"MATCH (n)-[:business_documents]->(m:document) WITH m AS n " +
		"WHERE n.id = $p0 RETURN n {.*}"
	if got != want {
		t.Errorf("Query() = %q\nwant  = %q", got, want)
	}
}

// TestTraversal_PredicateOnBothEnds is the exact repro from the filed bug:
// Business.Query().Where(ID(from)).QueryDocuments().Where(ID(to)).Exist(ctx)
// used to emit both predicates against the start node before the rebind,
// making the query always empty.
func TestTraversal_PredicateOnBothEnds(t *testing.T) {
	b := New()
	b.Match("(n:business)")
	p0 := b.AddParam("biz-1")
	b.Where("n.id = " + p0)
	traverse(b, "(n)-[:business_documents]->(m:document)")
	p1 := b.AddParam("doc-1")
	b.Where("n.id = " + p1)
	b.Return("n {.*}")
	b.Limit(1)

	got, _ := b.Query()
	want := "MATCH (n:business) WHERE n.id = $p0 " +
		"MATCH (n)-[:business_documents]->(m:document) WITH m AS n " +
		"WHERE n.id = $p1 RETURN n {.*} LIMIT 1"
	if got != want {
		t.Errorf("Query() = %q\nwant  = %q", got, want)
	}
}

func TestTraversal_InverseEdge(t *testing.T) {
	b := New()
	b.Match("(n:document)")
	p0 := b.AddParam("doc-1")
	b.Where("n.id = " + p0)
	traverse(b, "(n)<-[:business_documents]-(m:business)")
	p1 := b.AddParam("biz-1")
	b.Where("n.id = " + p1)
	b.Return("count(n)")

	got, _ := b.Query()
	want := "MATCH (n:document) WHERE n.id = $p0 " +
		"MATCH (n)<-[:business_documents]-(m:business) WITH m AS n " +
		"WHERE n.id = $p1 RETURN count(n)"
	if got != want {
		t.Errorf("Query() = %q\nwant  = %q", got, want)
	}
}

func TestTraversal_BidiEdge(t *testing.T) {
	b := New()
	b.Match("(n:user)")
	p0 := b.AddParam("u1")
	b.Where("n.id = " + p0)
	traverse(b, "(n)-[:user_friends]-(m:user)")
	p1 := b.AddParam("u2")
	b.Where("n.id = " + p1)
	b.Return("n {.*}")

	got, _ := b.Query()
	want := "MATCH (n:user) WHERE n.id = $p0 " +
		"MATCH (n)-[:user_friends]-(m:user) WITH m AS n " +
		"WHERE n.id = $p1 RETURN n {.*}"
	if got != want {
		t.Errorf("Query() = %q\nwant  = %q", got, want)
	}
}

func TestTraversal_MultiHopWithPredicates(t *testing.T) {
	b := New()
	b.Match("(n:business)")
	p0 := b.AddParam("biz-1")
	b.Where("n.id = " + p0)
	traverse(b, "(n)-[:business_documents]->(m:document)")
	p1 := b.AddParam("draft")
	b.Where("n.status = " + p1)
	traverse(b, "(n)-[:document_tables]->(m:table)")
	p2 := b.AddParam("tbl-1")
	b.Where("n.id = " + p2)
	b.Return("n {.*}")

	got, _ := b.Query()
	want := "MATCH (n:business) WHERE n.id = $p0 " +
		"MATCH (n)-[:business_documents]->(m:document) WITH m AS n WHERE n.status = $p1 " +
		"MATCH (n)-[:document_tables]->(m:table) WITH m AS n WHERE n.id = $p2 " +
		"RETURN n {.*}"
	if got != want {
		t.Errorf("Query() = %q\nwant  = %q", got, want)
	}
}

// Terminals only change the tail of the query; verify each with a filtered
// traversal prefix.
func TestTraversal_Terminals(t *testing.T) {
	build := func() *Builder {
		b := New()
		b.Match("(n:business)")
		p0 := b.AddParam("biz-1")
		b.Where("n.id = " + p0)
		traverse(b, "(n)-[:business_documents]->(m:document)")
		p1 := b.AddParam("doc-1")
		b.Where("n.id = " + p1)
		return b
	}
	prefix := "MATCH (n:business) WHERE n.id = $p0 " +
		"MATCH (n)-[:business_documents]->(m:document) WITH m AS n " +
		"WHERE n.id = $p1 "
	tests := []struct {
		name     string
		terminal func(*Builder)
		want     string
	}{
		{"Exist", func(b *Builder) { b.Return("count(n)").Limit(1) }, prefix + "RETURN count(n) LIMIT 1"},
		{"Count", func(b *Builder) { b.Return("count(n)") }, prefix + "RETURN count(n)"},
		{"All", func(b *Builder) { b.Return("n {.*}") }, prefix + "RETURN n {.*}"},
		{"FirstID", func(b *Builder) { b.Return("n.id").Limit(1) }, prefix + "RETURN n.id LIMIT 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := build()
			tt.terminal(b)
			got, _ := b.Query()
			if got != tt.want {
				t.Errorf("Query() = %q\nwant  = %q", got, tt.want)
			}
		})
	}
}

// --- With clause ---

func TestBuilder_With(t *testing.T) {
	b := New()
	b.Match("(n:User)")
	b.With("n")
	b.Match("(m:Pet)")
	b.Where("m.id = $p0")
	b.Merge("(n)-[:USER_HAS_PET]->(m)")
	b.Return("n {.*}")

	got, _ := b.Query()
	want := "MATCH (n:User) WITH n MATCH (m:Pet) WHERE m.id = $p0 MERGE (n)-[:USER_HAS_PET]->(m) RETURN n {.*}"
	if got != want {
		t.Errorf("Query() = %q\nwant  = %q", got, want)
	}
}

func TestBuilder_With_Rebind(t *testing.T) {
	b := New().Match("(n)-[:E]->(m:Target)").With("m AS n").Return("n {.*}")
	got, _ := b.Query()
	want := "MATCH (n)-[:E]->(m:Target) WITH m AS n RETURN n {.*}"
	if got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
}

// --- Upsert (MERGE + ON CREATE SET / ON MATCH SET) ---

func TestBuilder_OnCreateSetOnMatchSet(t *testing.T) {
	b := New()
	id := b.AddParam("ksuid-1")
	name := b.AddParam("Acme")
	created := b.AddParam("2026-01-01")
	b.Merge("(n:business {id: " + id + "})")
	b.OnCreateSet("n.name = " + name)
	b.OnCreateSet("n.created_at = " + created)
	b.OnMatchSet("n.name = " + name)
	b.Return("n {.*}")

	got, _ := b.Query()
	want := "MERGE (n:business {id: $p0}) " +
		"ON CREATE SET n.name = $p1, n.created_at = $p2 " +
		"ON MATCH SET n.name = $p1 " +
		"RETURN n {.*}"
	if got != want {
		t.Errorf("Query() = %q\nwant  = %q", got, want)
	}
}

func TestBuilder_UpsertWithEdges(t *testing.T) {
	// Full upsert shape the create template emits with OnConflict + edges:
	// MERGE node, conditional sets, then MERGE relationships.
	b := New()
	id := b.AddParam("biz-1")
	name := b.AddParam("Acme")
	doc := b.AddParam("doc-1")
	b.Merge("(n:business {id: " + id + "})")
	b.OnCreateSet("n.name = " + name)
	b.OnMatchSet("n.name = " + name)
	b.With("n")
	b.Match("(m:document)")
	b.Where("m.id = " + doc)
	b.Merge("(n)-[:business_documents]->(m)")
	b.Return("n {.*}")

	got, _ := b.Query()
	want := "MERGE (n:business {id: $p0}) " +
		"ON CREATE SET n.name = $p1 ON MATCH SET n.name = $p1 " +
		"WITH n MATCH (m:document) WHERE m.id = $p2 " +
		"MERGE (n)-[:business_documents]->(m) RETURN n {.*}"
	if got != want {
		t.Errorf("Query() = %q\nwant  = %q", got, want)
	}
}
