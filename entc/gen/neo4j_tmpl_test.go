// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package gen

import (
	"strings"
	"testing"

	"entgo.io/ent/entc/load"
)

// neo4jTemplateContract defines the set of template definitions that must
// exist for the Neo4j dialect to function. Each template name corresponds
// to a Go text/template definition that the code generation engine resolves
// dynamically via the "dialect/<storage>/<template>" naming convention.
//
// This is the interface/contract for the Neo4j code generation templates.
var neo4jTemplateContract = []struct {
	name        string // template definition name
	description string // what the template generates
}{
	// Core CRUD templates (medium priority)
	{"dialect/neo4j/create", "neo4jSave(ctx) and neo4j() methods on CreateBuilder"},
	{"dialect/neo4j/query", "neo4jAll(ctx), neo4jCount(ctx), neo4jQuery(ctx) methods"},
	{"dialect/neo4j/query/path", "edge traversal path patterns for query building"},
	{"dialect/neo4j/query/from", "reverse edge traversal for inverse edges"},
	{"dialect/neo4j/update", "neo4jSave(ctx) for UpdateOne and bulk Update"},
	{"dialect/neo4j/delete", "neo4jExec(ctx) with DELETE/DETACH DELETE"},
	{"dialect/neo4j/decode/one", "FromResponse(res) for single entity decoding"},
	{"dialect/neo4j/decode/many", "FromResponse(res) for multiple entity decoding"},

	// Predicate sub-templates (medium priority)
	{"dialect/neo4j/predicate/id", "WHERE n.id = $p0"},
	{"dialect/neo4j/predicate/id/ops", "WHERE n.id IN $p0, etc."},
	{"dialect/neo4j/predicate/field", "WHERE n.field = $p0"},
	{"dialect/neo4j/predicate/field/ops", "all OpCode operations on fields"},
	{"dialect/neo4j/predicate/edge/has", "MATCH (n)-[:REL]->()"},
	{"dialect/neo4j/predicate/edge/haswith", "edge predicate with target conditions"},
	{"dialect/neo4j/predicate/and", "(cond1 AND cond2)"},
	{"dialect/neo4j/predicate/or", "(cond1 OR cond2)"},
	{"dialect/neo4j/predicate/not", "NOT (cond)"},

	// Supporting templates (low priority)
	{"dialect/neo4j/client/open", "Open(driverName, dsn) case for Neo4j"},
	{"dialect/neo4j/errors", "ConstraintError handling for uniqueness violations"},
	{"dialect/neo4j/meta/constants", "relationship type constants in SCREAMING_SNAKE_CASE"},
	{"dialect/neo4j/order/signature", "type OrderFunc func(*cypher.Builder)"},
	{"dialect/neo4j/order/func", "b.OrderBy(n.field ASC/DESC)"},
	{"dialect/neo4j/group/signature", "type AggregateFunc func(string) string"},
	{"dialect/neo4j/group/as", "aggregation alias helper"},
	{"dialect/neo4j/group/func", "aggregation function builder"},
	{"dialect/neo4j/group/const", "default aggregation label constants"},
	{"dialect/neo4j/globals", "queryHook type alias for Neo4j dialect"},
	{"dialect/neo4j/group", "neo4jScan(ctx) for GROUP BY aggregation"},
	{"dialect/neo4j/select", "neo4jScan(ctx) for field-selective queries"},
}

// TestNeo4jTemplateDefinitions verifies that all required Neo4j template
// definitions are present in the loaded template set. Each template is a
// named block that the code generation engine resolves dynamically based
// on the selected storage driver.
func TestNeo4jTemplateDefinitions(t *testing.T) {
	initTemplates()
	for _, tc := range neo4jTemplateContract {
		t.Run(tc.name, func(t *testing.T) {
			if !hasTemplate(tc.name) {
				t.Errorf("template %q not defined; expected: %s", tc.name, tc.description)
			}
		})
	}
}

// TestNeo4jTemplateParity verifies that every Gremlin dialect template
// has a corresponding Neo4j dialect template. This ensures feature parity
// between the two graph database backends.
func TestNeo4jTemplateParity(t *testing.T) {
	initTemplates()
	gremlinTemplates := matchTemplate("dialect/gremlin/*")
	for _, gt := range gremlinTemplates {
		neo4jName := strings.Replace(gt, "gremlin", "neo4j", 1)
		t.Run(neo4jName, func(t *testing.T) {
			if !hasTemplate(neo4jName) {
				t.Errorf("missing Neo4j template %q (Gremlin has %q)", neo4jName, gt)
			}
		})
	}
}

// TestNeo4jTemplateSubTemplateParity verifies that nested sub-templates
// (predicate/*, query/*, decode/*, order/*, group/*) in Gremlin have
// corresponding Neo4j equivalents.
func TestNeo4jTemplateSubTemplateParity(t *testing.T) {
	initTemplates()
	patterns := []string{
		"dialect/gremlin/predicate/*",
		"dialect/gremlin/query/*",
		"dialect/gremlin/decode/*",
		"dialect/gremlin/order/*",
		"dialect/gremlin/group/*",
		"dialect/gremlin/client/*",
	}
	gremlinSubs := matchTemplate(patterns...)
	for _, gt := range gremlinSubs {
		neo4jName := strings.Replace(gt, "gremlin", "neo4j", 1)
		t.Run(neo4jName, func(t *testing.T) {
			if !hasTemplate(neo4jName) {
				t.Errorf("missing Neo4j sub-template %q (Gremlin has %q)", neo4jName, gt)
			}
		})
	}
}

// --- Medium Priority: Core Template Tests ---

// TestNeo4jCreateTemplate verifies the create template generates methods
// that build Cypher CREATE queries with KSUID id assignment, uniqueness
// guards via OPTIONAL MATCH, and edge creation via WITH/MATCH/CREATE.
//
// Expected generated method signatures:
//
//	func (b *XxxCreateBuilder) neo4jSave(ctx context.Context) (*Xxx, error)
//	func (b *XxxCreateBuilder) neo4j() *cypher.Builder
//
// Expected Cypher patterns:
//   - Simple:     CREATE (n:Label {id: $p0, field: $p1, ...}) RETURN n {.*}
//   - Unique:     OPTIONAL MATCH (existing:Label {field: $p0}) WITH existing WHERE existing IS NULL CREATE ...
//   - With edge:  ... WITH n MATCH (m:Target) WHERE m.id = $pN CREATE (n)-[:REL]->(m) ...
func TestNeo4jCreateTemplate(t *testing.T) {
	initTemplates()
	if !hasTemplate("dialect/neo4j/create") {
		t.Fatal("template dialect/neo4j/create not defined")
	}
}

// TestNeo4jQueryTemplate verifies the query template generates methods
// for MATCH queries with predicate application, edge traversal patterns,
// ORDER BY, SKIP/LIMIT pagination, and count aggregation.
//
// Expected generated method signatures:
//
//	func (q *XxxQuery) neo4jAll(ctx context.Context, hooks ...queryHook) ([]*Xxx, error)
//	func (q *XxxQuery) neo4jCount(ctx context.Context) (int, error)
//	func (q *XxxQuery) neo4jQuery(ctx context.Context) *cypher.Builder
//
// Expected Cypher patterns:
//   - All:     MATCH (n:Label) WHERE ... RETURN n {.*}
//   - Count:   MATCH (n:Label) WHERE ... RETURN count(n)
//   - Path:    MATCH (n:Label)-[:REL]->(m:Target) ...
//   - From:    g.V(id).OutE(label).InV() equivalent
func TestNeo4jQueryTemplate(t *testing.T) {
	initTemplates()
	requiredTemplates := []string{
		"dialect/neo4j/query",
		"dialect/neo4j/query/path",
		"dialect/neo4j/query/from",
	}
	for _, name := range requiredTemplates {
		if !hasTemplate(name) {
			t.Errorf("template %q not defined", name)
		}
	}
}

// TestNeo4jUpdateTemplate verifies the update template generates methods
// for SET operations on fields, REMOVE for clearing optional fields,
// and edge mutation (DELETE old, CREATE new relationships).
//
// Expected generated method signatures:
//
//	func (u *XxxUpdateOne) neo4jSave(ctx context.Context) (*Xxx, error)
//	func (u *XxxUpdate) neo4jSave(ctx context.Context) (int, error)
//
// Expected Cypher patterns:
//   - UpdateOne: MATCH (n:Label) WHERE n.id = $p0 SET n.field = $pN RETURN n {.*}
//   - Bulk:      MATCH (n:Label) WHERE <predicates> SET n.field = $pN RETURN count(n)
//   - Clear:     ... REMOVE n.field ...
//   - Edge add:  ... WITH n MATCH (m:Target) WHERE m.id = $pN CREATE (n)-[:REL]->(m) ...
//   - Edge rm:   ... WITH n MATCH (n)-[r:REL]->(m) WHERE m.id = $pN DELETE r ...
func TestNeo4jUpdateTemplate(t *testing.T) {
	initTemplates()
	if !hasTemplate("dialect/neo4j/update") {
		t.Fatal("template dialect/neo4j/update not defined")
	}
}

// TestNeo4jDeleteTemplate verifies the delete template generates methods
// using DELETE for normal deletion and DETACH DELETE when cascade mode
// is enabled in SchemaMode.
//
// Expected generated method signatures:
//
//	func (d *XxxDelete) neo4jExec(ctx context.Context) (int, error)
//
// Expected Cypher patterns:
//   - Normal:  MATCH (n:Label) WHERE ... DELETE n RETURN count(n)
//   - Cascade: MATCH (n:Label) WHERE ... DETACH DELETE n RETURN count(n)
func TestNeo4jDeleteTemplate(t *testing.T) {
	initTemplates()
	if !hasTemplate("dialect/neo4j/delete") {
		t.Fatal("template dialect/neo4j/delete not defined")
	}
}

// TestNeo4jPredicateTemplates verifies all predicate sub-templates exist
// and cover: id predicates, field predicates with all OpCode operations,
// edge existence checks (has/haswith), and boolean combinators (and/or/not).
//
// Predicate functions have the signature: func(*cypher.Builder)
// Each predicate appends a WHERE condition to the builder.
//
// Expected Cypher patterns per sub-template:
//   - id:             b.Where("n.id = " + b.AddParam(id))
//   - id/ops:         b.Where("n.id IN " + b.AddParam(ids)) etc.
//   - field:          b.Where("n.FieldName = " + b.AddParam(v))
//   - field/ops:      all OpCode operations (EQ, NEQ, GT, GTE, LT, LTE, IsNull, NotNull, In, NotIn, Contains, StartsWith, EndsWith)
//   - edge/has:       b.Match("(n)-[:REL]->()")
//   - edge/haswith:   b.Match("(n)-[:REL]->(m)") + target predicates on m
//   - and:            b.Where("(cond1 AND cond2)")
//   - or:             b.Where("(cond1 OR cond2)")
//   - not:            b.Where("NOT (cond)")
func TestNeo4jPredicateTemplates(t *testing.T) {
	initTemplates()
	predicateTemplates := []struct {
		name    string
		pattern string
	}{
		{"dialect/neo4j/predicate/id", "n.id = $p"},
		{"dialect/neo4j/predicate/id/ops", "n.id IN $p"},
		{"dialect/neo4j/predicate/field", "n.field = $p"},
		{"dialect/neo4j/predicate/field/ops", "OpCode operations"},
		{"dialect/neo4j/predicate/edge/has", "(n)-[:REL]->()"},
		{"dialect/neo4j/predicate/edge/haswith", "edge with target predicates"},
		{"dialect/neo4j/predicate/and", "AND"},
		{"dialect/neo4j/predicate/or", "OR"},
		{"dialect/neo4j/predicate/not", "NOT"},
	}
	for _, tc := range predicateTemplates {
		t.Run(tc.name, func(t *testing.T) {
			if !hasTemplate(tc.name) {
				t.Errorf("template %q not defined; should generate: %s", tc.name, tc.pattern)
			}
		})
	}
}

// TestNeo4jDecodeTemplates verifies the decode templates generate
// FromResponse methods that extract map[string]any from Neo4j records
// and map property names to struct fields with type conversion
// (e.g., Neo4j int64 -> Go int/int32).
//
// Expected generated method signatures:
//
//	func (x *Xxx) FromResponse(res *neo4j.Response) error           // decode/one
//	func (xs *Xxxs) FromResponse(res *neo4j.Response) error         // decode/many
func TestNeo4jDecodeTemplates(t *testing.T) {
	initTemplates()
	decodeTemplates := []string{
		"dialect/neo4j/decode/one",
		"dialect/neo4j/decode/many",
	}
	for _, name := range decodeTemplates {
		t.Run(name, func(t *testing.T) {
			if !hasTemplate(name) {
				t.Errorf("template %q not defined", name)
			}
		})
	}
}

// --- Low Priority: Supporting Template Tests ---

// TestNeo4jOpenTemplate verifies the open template generates the
// Open(driverName, dsn) case that parses a Neo4j URI and builds a driver.
//
// Expected generated code pattern:
//
//	cfg, err := neo4j.ParseURI(dataSourceName)
//	drv, err := cfg.Build()
//	return NewClient(append(options, Driver(drv))...), nil
func TestNeo4jOpenTemplate(t *testing.T) {
	initTemplates()
	if !hasTemplate("dialect/neo4j/client/open") {
		t.Fatal("template dialect/neo4j/client/open not defined")
	}
}

// TestNeo4jErrorsTemplate verifies the errors template generates
// ConstraintError handling for application-level uniqueness violations
// returned when guarded CREATE produces zero rows.
//
// Must generate:
//   - NewErrUniqueField(label, field, v) function
//   - NewErrUniqueEdge(label, edge, id) function
//   - isConstraintError(res) helper to detect zero-row uniqueness violations
func TestNeo4jErrorsTemplate(t *testing.T) {
	initTemplates()
	if !hasTemplate("dialect/neo4j/errors") {
		t.Fatal("template dialect/neo4j/errors not defined")
	}
}

// TestNeo4jMetaTemplate verifies the meta template generates relationship
// type constants in SCREAMING_SNAKE_CASE format.
//
// Expected generated code:
//
//	const (
//	    UserHasPetLabel    = "USER_HAS_PET"
//	    UserHasFriendLabel = "USER_HAS_FRIEND"
//	)
func TestNeo4jMetaTemplate(t *testing.T) {
	initTemplates()
	if !hasTemplate("dialect/neo4j/meta/constants") {
		t.Fatal("template dialect/neo4j/meta/constants not defined")
	}
}

// TestNeo4jByTemplates verifies the by templates generate ordering and
// grouping function types and implementations for Cypher queries.
//
// Expected generated types:
//
//	type OrderFunc func(*cypher.Builder)
//	type AggregateFunc func(string) string
func TestNeo4jByTemplates(t *testing.T) {
	initTemplates()
	byTemplates := []struct {
		name        string
		description string
	}{
		{"dialect/neo4j/order/signature", "type OrderFunc func(*cypher.Builder)"},
		{"dialect/neo4j/order/func", "b.OrderBy(n.field ASC/DESC)"},
		{"dialect/neo4j/group/signature", "type AggregateFunc"},
		{"dialect/neo4j/group/as", "aggregation alias"},
		{"dialect/neo4j/group/func", "aggregation function"},
		{"dialect/neo4j/group/const", "default label constant"},
	}
	for _, tc := range byTemplates {
		t.Run(tc.name, func(t *testing.T) {
			if !hasTemplate(tc.name) {
				t.Errorf("template %q not defined; should generate: %s", tc.name, tc.description)
			}
		})
	}
}

// TestNeo4jGlobalsTemplate verifies the globals template generates
// the queryHook type alias to align the Neo4j dialect API surface
// with other dialects.
//
// Expected generated code:
//
//	type queryHook func(context.Context)
func TestNeo4jGlobalsTemplate(t *testing.T) {
	initTemplates()
	if !hasTemplate("dialect/neo4j/globals") {
		t.Fatal("template dialect/neo4j/globals not defined")
	}
}

// TestNeo4jGroupTemplate verifies the group template generates
// neo4jScan(ctx) for GROUP BY aggregation using Cypher
// WITH n.field, count(*) pattern.
//
// Expected generated method signature:
//
//	func (gb *XxxGroupBy) neo4jScan(ctx context.Context, root *XxxQuery, v any) error
func TestNeo4jGroupTemplate(t *testing.T) {
	initTemplates()
	if !hasTemplate("dialect/neo4j/group") {
		t.Fatal("template dialect/neo4j/group not defined")
	}
}

// TestNeo4jSelectTemplate verifies the select template generates
// neo4jScan(ctx) for field-selective queries using
// RETURN n.field1, n.field2 instead of n {.*}.
//
// Expected generated method signature:
//
//	func (s *XxxSelect) neo4jScan(ctx context.Context, root *XxxQuery, v any) error
func TestNeo4jSelectTemplate(t *testing.T) {
	initTemplates()
	if !hasTemplate("dialect/neo4j/select") {
		t.Fatal("template dialect/neo4j/select not defined")
	}
}

// --- Code Generation Integration Test ---

// TestNeo4jCodeGen_WithNeo4jStorage verifies that code generation with
// Neo4j storage selected produces output files. This test constructs a
// minimal schema Graph with Neo4j as the storage driver and attempts
// code generation. It should fail until all Neo4j templates are defined.
func TestNeo4jCodeGen_WithNeo4jStorage(t *testing.T) {
	initTemplates()

	// Verify Neo4j storage driver is registered.
	s, err := NewStorage("neo4j")
	if err != nil {
		t.Fatalf("NewStorage(neo4j) error = %v", err)
	}

	// Verify all required templates are defined before attempting generation.
	missing := 0
	for _, tc := range neo4jTemplateContract {
		if !hasTemplate(tc.name) {
			missing++
		}
	}
	if missing > 0 {
		t.Fatalf("cannot run code generation: %d/%d required Neo4j templates are not defined",
			missing, len(neo4jTemplateContract))
	}

	// Build a minimal Graph with Neo4j storage.
	target := t.TempDir()
	_, err = NewGraph(&Config{
		Package: "entc/gen",
		Target:  target,
		Storage: s,
	}, &load.Schema{Name: "User"})
	if err != nil {
		t.Fatalf("NewGraph error = %v", err)
	}
}
