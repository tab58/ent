// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package cypher

import (
	"testing"
)

func TestNew(t *testing.T) {
	b := New()
	if b == nil {
		t.Fatal("New() returned nil")
	}
	if b.params == nil {
		t.Fatal("New() did not initialize params map")
	}
	q, params := b.Query()
	if q != "" {
		t.Errorf("empty builder query = %q, want empty string", q)
	}
	if len(params) != 0 {
		t.Errorf("empty builder params = %v, want empty map", params)
	}
}

func TestBuilder_Match(t *testing.T) {
	tests := []struct {
		name      string
		patterns  []string
		wantQuery string
	}{
		{
			name:      "single match",
			patterns:  []string{"(n:User)"},
			wantQuery: "MATCH (n:User)",
		},
		{
			name:      "multiple matches",
			patterns:  []string{"(n:User)", "(n)-[:OWNS]->(m:Pet)"},
			wantQuery: "MATCH (n:User) MATCH (n)-[:OWNS]->(m:Pet)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New()
			for _, p := range tt.patterns {
				b.Match(p)
			}
			got, _ := b.Query()
			if got != tt.wantQuery {
				t.Errorf("Query() = %q, want %q", got, tt.wantQuery)
			}
		})
	}
}

func TestBuilder_Where(t *testing.T) {
	tests := []struct {
		name      string
		conds     []string
		wantQuery string
	}{
		{
			name:      "single where",
			conds:     []string{"n.name = $p0"},
			wantQuery: "WHERE n.name = $p0",
		},
		{
			name:      "multiple where conditions joined with AND",
			conds:     []string{"n.name = $p0", "n.age > $p1"},
			wantQuery: "WHERE n.name = $p0 AND n.age > $p1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New()
			for _, c := range tt.conds {
				b.Where(c)
			}
			got, _ := b.Query()
			if got != tt.wantQuery {
				t.Errorf("Query() = %q, want %q", got, tt.wantQuery)
			}
		})
	}
}

func TestBuilder_Create(t *testing.T) {
	b := New().Create("(n:User {name: $p0, id: $p1})")
	got, _ := b.Query()
	want := "CREATE (n:User {name: $p0, id: $p1})"
	if got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
}

func TestBuilder_Merge(t *testing.T) {
	b := New().Merge("(n:User {email: $p0})")
	got, _ := b.Query()
	want := "MERGE (n:User {email: $p0})"
	if got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
}

func TestBuilder_Set(t *testing.T) {
	tests := []struct {
		name      string
		exprs     []string
		wantQuery string
	}{
		{
			name:      "single set",
			exprs:     []string{"n.name = $p0"},
			wantQuery: "SET n.name = $p0",
		},
		{
			name:      "multiple set expressions comma-joined",
			exprs:     []string{"n.name = $p0", "n.age = $p1"},
			wantQuery: "SET n.name = $p0, n.age = $p1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New()
			for _, e := range tt.exprs {
				b.Set(e)
			}
			got, _ := b.Query()
			if got != tt.wantQuery {
				t.Errorf("Query() = %q, want %q", got, tt.wantQuery)
			}
		})
	}
}

func TestBuilder_Remove(t *testing.T) {
	b := New().Remove("n.email")
	got, _ := b.Query()
	want := "REMOVE n.email"
	if got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
}

func TestBuilder_Delete(t *testing.T) {
	b := New().Match("(n:User)").Where("n.id = $p0").Delete("n").Return("count(n)")
	got, _ := b.Query()
	want := "MATCH (n:User) WHERE n.id = $p0 DELETE n RETURN count(n)"
	if got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
}

func TestBuilder_DetachDelete(t *testing.T) {
	b := New().Match("(n:User)").Where("n.id = $p0").DetachDelete("n").Return("count(n)")
	got, _ := b.Query()
	want := "MATCH (n:User) WHERE n.id = $p0 DETACH DELETE n RETURN count(n)"
	if got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
}

func TestBuilder_Return(t *testing.T) {
	b := New().Match("(n:User)").Return("n {.*}")
	got, _ := b.Query()
	want := "MATCH (n:User) RETURN n {.*}"
	if got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
}

func TestBuilder_Return_Multiple(t *testing.T) {
	b := New().Match("(n:User)").Return("n.name", "n.age")
	got, _ := b.Query()
	want := "MATCH (n:User) RETURN n.name, n.age"
	if got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
}

func TestBuilder_OrderBy(t *testing.T) {
	b := New().Match("(n:User)").Return("n {.*}").OrderBy("n.name ASC")
	got, _ := b.Query()
	want := "MATCH (n:User) RETURN n {.*} ORDER BY n.name ASC"
	if got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
}

func TestBuilder_Skip(t *testing.T) {
	b := New().Match("(n:User)").Return("n {.*}").Skip(10)
	got, _ := b.Query()
	want := "MATCH (n:User) RETURN n {.*} SKIP 10"
	if got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
}

func TestBuilder_Limit(t *testing.T) {
	b := New().Match("(n:User)").Return("n {.*}").Limit(25)
	got, _ := b.Query()
	want := "MATCH (n:User) RETURN n {.*} LIMIT 25"
	if got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
}

func TestBuilder_Pagination(t *testing.T) {
	b := New().Match("(n:User)").Return("n {.*}").OrderBy("n.name ASC").Skip(10).Limit(25)
	got, _ := b.Query()
	want := "MATCH (n:User) RETURN n {.*} ORDER BY n.name ASC SKIP 10 LIMIT 25"
	if got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
}

func TestBuilder_AddParam(t *testing.T) {
	b := New()
	p0 := b.AddParam("alice")
	p1 := b.AddParam(30)
	p2 := b.AddParam(true)

	if p0 != "$p0" {
		t.Errorf("first param = %q, want $p0", p0)
	}
	if p1 != "$p1" {
		t.Errorf("second param = %q, want $p1", p1)
	}
	if p2 != "$p2" {
		t.Errorf("third param = %q, want $p2", p2)
	}

	_, params := b.Query()
	if params["p0"] != "alice" {
		t.Errorf("params[p0] = %v, want alice", params["p0"])
	}
	if params["p1"] != 30 {
		t.Errorf("params[p1] = %v, want 30", params["p1"])
	}
	if params["p2"] != true {
		t.Errorf("params[p2] = %v, want true", params["p2"])
	}
}

func TestBuilder_SetParam(t *testing.T) {
	b := New()
	b.SetParam("userId", "abc-123")
	_, params := b.Query()
	if params["userId"] != "abc-123" {
		t.Errorf("params[userId] = %v, want abc-123", params["userId"])
	}
}

func TestBuilder_Chaining(t *testing.T) {
	b := New()
	p0 := b.AddParam("alice")
	p1 := b.AddParam("ksuid-123")

	result := b.
		Match("(n:User)").
		Where("n.name = " + p0).
		Where("n.id = " + p1).
		Return("n {.*}")

	if result != b {
		t.Error("chaining should return the same builder")
	}

	got, _ := b.Query()
	want := "MATCH (n:User) WHERE n.name = $p0 AND n.id = $p1 RETURN n {.*}"
	if got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
}

func TestBuilder_FullCreateQuery(t *testing.T) {
	b := New()
	id := b.AddParam("ksuid-abc")
	name := b.AddParam("alice")
	email := b.AddParam("alice@example.com")

	b.Create("(n:User {id: " + id + ", name: " + name + ", email: " + email + "})").
		Return("n {.*}")

	got, params := b.Query()
	want := "CREATE (n:User {id: $p0, name: $p1, email: $p2}) RETURN n {.*}"
	if got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
	if len(params) != 3 {
		t.Errorf("params count = %d, want 3", len(params))
	}
}

func TestBuilder_FullUpdateQuery(t *testing.T) {
	b := New()
	id := b.AddParam("ksuid-abc")
	name := b.AddParam("bob")

	b.Match("(n:User)").
		Where("n.id = " + id).
		Set("n.name = " + name).
		Remove("n.email").
		Return("n {.*}")

	got, _ := b.Query()
	want := "MATCH (n:User) WHERE n.id = $p0 SET n.name = $p1 REMOVE n.email RETURN n {.*}"
	if got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
}

func TestBuilder_FullDeleteWithCascade(t *testing.T) {
	b := New()
	id := b.AddParam("ksuid-abc")

	b.Match("(n:User)").
		Where("n.id = " + id).
		DetachDelete("n").
		Return("count(n)")

	got, _ := b.Query()
	want := "MATCH (n:User) WHERE n.id = $p0 DETACH DELETE n RETURN count(n)"
	if got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
}

func TestBuilder_EdgeCreation(t *testing.T) {
	b := New()
	id := b.AddParam("ksuid-user")
	petID := b.AddParam("ksuid-pet")

	b.Match("(n:User)").
		Where("n.id = " + id).
		Match("(m:Pet)").
		Where("m.id = " + petID).
		Create("(n)-[:USER_HAS_PET]->(m)").
		Return("n {.*}")

	got, _ := b.Query()
	// Multiple MATCH and WHERE clauses, WHERE conditions are ANDed per block
	want := "MATCH (n:User) MATCH (m:Pet) WHERE n.id = $p0 AND m.id = $p1 CREATE (n)-[:USER_HAS_PET]->(m) RETURN n {.*}"
	if got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
}

func TestBuilder_Clone(t *testing.T) {
	b := New()
	b.AddParam("alice")
	b.Match("(n:User)").Where("n.name = $p0").Return("n {.*}").Skip(5).Limit(10)

	c := b.Clone()

	// Modify the clone — should not affect original.
	c.AddParam("bob")
	c.Where("n.age > $p1")

	origQ, origP := b.Query()
	cloneQ, cloneP := c.Query()

	if origQ == cloneQ {
		t.Error("modifying clone should not affect original query")
	}
	if len(origP) == len(cloneP) {
		t.Error("modifying clone params should not affect original params")
	}
	// Original should still only have 1 param
	if len(origP) != 1 {
		t.Errorf("original params count = %d, want 1", len(origP))
	}
}

func TestBuilder_Clone_Nil(t *testing.T) {
	var b *Builder
	c := b.Clone()
	if c != nil {
		t.Error("Clone of nil should return nil")
	}
}

func TestBuilder_Clone_PreservesSkipLimit(t *testing.T) {
	b := New().Match("(n:User)").Return("n {.*}").Skip(5).Limit(10)
	c := b.Clone()

	origQ, _ := b.Query()
	cloneQ, _ := c.Query()
	if origQ != cloneQ {
		t.Errorf("clone query = %q, want %q", cloneQ, origQ)
	}
}

func TestBuilder_EmptyClauses(t *testing.T) {
	// A builder with no clauses should produce an empty query.
	b := New()
	q, params := b.Query()
	if q != "" {
		t.Errorf("empty Query() = %q, want empty string", q)
	}
	if len(params) != 0 {
		t.Errorf("empty params = %v, want empty map", params)
	}
}

func TestBuilder_UniqueFieldGuard(t *testing.T) {
	// Tests the OPTIONAL MATCH pattern for uniqueness enforcement.
	b := New()
	email := b.AddParam("alice@example.com")
	id := b.AddParam("ksuid-abc")
	name := b.AddParam("alice")

	b.Match("(existing:User {email: " + email + "})").
		Where("existing IS NULL").
		Create("(n:User {id: " + id + ", email: " + email + ", name: " + name + "})").
		Return("n {.*}")

	got, params := b.Query()
	want := "MATCH (existing:User {email: $p0}) WHERE existing IS NULL CREATE (n:User {id: $p1, email: $p0, name: $p2}) RETURN n {.*}"
	if got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
	if len(params) != 3 {
		t.Errorf("params count = %d, want 3", len(params))
	}
}

func TestBuilder_CountQuery(t *testing.T) {
	b := New().Match("(n:User)").Return("count(n)")
	got, _ := b.Query()
	want := "MATCH (n:User) RETURN count(n)"
	if got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
}

func TestBuilder_ParamSequencing(t *testing.T) {
	b := New()
	// Ensure params are sequentially numbered.
	for i := 0; i < 5; i++ {
		p := b.AddParam(i)
		want := "$p" + string(rune('0'+i))
		if p != want {
			t.Errorf("param %d = %q, want %q", i, p, want)
		}
	}
	_, params := b.Query()
	if len(params) != 5 {
		t.Errorf("params count = %d, want 5", len(params))
	}
}

func TestBuilder_WhereClauses(t *testing.T) {
	b := New()
	b.Where("n.name = $p0")
	b.Where("n.age > $p1")
	clauses := b.WhereClauses()
	if len(clauses) != 2 {
		t.Fatalf("WhereClauses() returned %d clauses, want 2", len(clauses))
	}
	if clauses[0] != "n.name = $p0" {
		t.Errorf("clauses[0] = %q, want %q", clauses[0], "n.name = $p0")
	}
	if clauses[1] != "n.age > $p1" {
		t.Errorf("clauses[1] = %q, want %q", clauses[1], "n.age > $p1")
	}
}

func TestBuilder_WhereClauses_Empty(t *testing.T) {
	b := New()
	clauses := b.WhereClauses()
	if len(clauses) != 0 {
		t.Errorf("WhereClauses() on empty builder = %v, want empty", clauses)
	}
}

func TestBuilder_Params(t *testing.T) {
	b := New()
	b.AddParam("alice")
	b.SetParam("custom", 42)
	params := b.Params()
	if len(params) != 2 {
		t.Fatalf("Params() returned %d params, want 2", len(params))
	}
	if params["p0"] != "alice" {
		t.Errorf("Params()[p0] = %v, want alice", params["p0"])
	}
	if params["custom"] != 42 {
		t.Errorf("Params()[custom] = %v, want 42", params["custom"])
	}
}

func TestBuilder_Params_Empty(t *testing.T) {
	b := New()
	params := b.Params()
	if len(params) != 0 {
		t.Errorf("Params() on empty builder = %v, want empty", params)
	}
}

func TestBuilder_CollectWhere(t *testing.T) {
	// Simulate AND predicate combination: CollectWhere captures WHERE
	// conditions added by each predicate without nested WHERE keywords,
	// while params remain correctly sequenced on the parent builder.
	pred1 := func(b *Builder) {
		p := b.AddParam("alice")
		b.Where("n.name = " + p)
	}
	pred2 := func(b *Builder) {
		p := b.AddParam(30)
		b.Where("n.age > " + p)
	}

	parent := New()
	parent.Match("(n:User)")

	var conds []string
	for _, pred := range []func(*Builder){pred1, pred2} {
		conds = append(conds, parent.CollectWhere(pred)...)
	}
	parent.Where("(" + conds[0] + " AND " + conds[1] + ")")
	parent.Return("n {.*}")

	got, params := parent.Query()
	want := "MATCH (n:User) WHERE (n.name = $p0 AND n.age > $p1) RETURN n {.*}"
	if got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
	if len(params) != 2 {
		t.Errorf("params count = %d, want 2", len(params))
	}
}

func TestBuilder_CollectWhere_Empty(t *testing.T) {
	b := New()
	conds := b.CollectWhere(func(b *Builder) {
		// no-op predicate
	})
	if len(conds) != 0 {
		t.Errorf("CollectWhere with no-op = %v, want empty", conds)
	}
}

func TestBuilder_CollectWhere_PreservesExisting(t *testing.T) {
	b := New()
	b.Where("n.active = true")

	conds := b.CollectWhere(func(b *Builder) {
		b.Where("n.name = $p0")
	})

	if len(conds) != 1 || conds[0] != "n.name = $p0" {
		t.Errorf("CollectWhere = %v, want [n.name = $p0]", conds)
	}

	// Original WHERE should still be there.
	existing := b.WhereClauses()
	if len(existing) != 1 || existing[0] != "n.active = true" {
		t.Errorf("existing WHERE = %v, want [n.active = true]", existing)
	}
}

func TestBuilder_MultipleOrderBy(t *testing.T) {
	b := New().Match("(n:User)").Return("n {.*}").OrderBy("n.name ASC").OrderBy("n.age DESC")
	got, _ := b.Query()
	want := "MATCH (n:User) RETURN n {.*} ORDER BY n.name ASC, n.age DESC"
	if got != want {
		t.Errorf("Query() = %q, want %q", got, want)
	}
}
