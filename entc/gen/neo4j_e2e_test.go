// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package gen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"entgo.io/ent/dialect/neo4j/cypher"
	"entgo.io/ent/entc/load"
	"entgo.io/ent/schema/field"
)

// megatronSchemas returns the 6 entity schemas matching the megatron-api model.
// This is the contract for the Neo4j end-to-end codegen test.
//
// Entity graph:
//
//	Business (name) --[documents]--> Document (name)
//	Business (name) --[categories]--> Category (name)
//	Business (name) --[years]--> Year (value)
//	Document (name) --[categories]--> Category
//	Document (name) --[years]--> Year
//	Document (name) --[tables]--> Table (name)
//	Category (name) --[table_cells]--> TableCell (value, vector, categories)
//	Table (name) --[table_cells]--> TableCell
//	Year (value) --[table_cells]--> TableCell
func megatronSchemas() []*load.Schema {
	return []*load.Schema{
		{
			Name: "Business",
			Fields: []*load.Field{
				{Name: "name", Info: &field.TypeInfo{Type: field.TypeString}},
			},
			Edges: []*load.Edge{
				{Name: "documents", Type: "Document"},
				{Name: "categories", Type: "Category"},
				{Name: "years", Type: "Year"},
			},
		},
		{
			Name: "Document",
			Fields: []*load.Field{
				{Name: "name", Info: &field.TypeInfo{Type: field.TypeString}},
			},
			Edges: []*load.Edge{
				{Name: "business", Type: "Business", Inverse: true, Unique: true, RefName: "documents"},
				{Name: "categories", Type: "Category"},
				{Name: "years", Type: "Year"},
				{Name: "tables", Type: "Table"},
			},
		},
		{
			Name: "Category",
			Fields: []*load.Field{
				{Name: "name", Info: &field.TypeInfo{Type: field.TypeString}},
			},
			Edges: []*load.Edge{
				{Name: "business", Type: "Business", Inverse: true, Unique: true, RefName: "categories"},
				{Name: "document", Type: "Document", Inverse: true, Unique: true, RefName: "categories"},
				{Name: "table_cells", Type: "TableCell"},
			},
		},
		{
			Name: "Year",
			Fields: []*load.Field{
				{Name: "value", Info: &field.TypeInfo{Type: field.TypeInt}},
			},
			Edges: []*load.Edge{
				{Name: "business", Type: "Business", Inverse: true, Unique: true, RefName: "years"},
				{Name: "document", Type: "Document", Inverse: true, Unique: true, RefName: "years"},
				{Name: "table_cells", Type: "TableCell"},
			},
		},
		{
			Name: "Table",
			Fields: []*load.Field{
				{Name: "name", Info: &field.TypeInfo{Type: field.TypeString}},
			},
			Edges: []*load.Edge{
				{Name: "document", Type: "Document", Inverse: true, Unique: true, RefName: "tables"},
				{Name: "table_cells", Type: "TableCell"},
			},
		},
		{
			Name: "TableCell",
			Fields: []*load.Field{
				{Name: "value", Info: &field.TypeInfo{Type: field.TypeFloat64}},
				{Name: "vector", Info: &field.TypeInfo{Type: field.TypeJSON, Ident: "[]float64", Nillable: true}},
				{Name: "categories", Info: &field.TypeInfo{Type: field.TypeJSON, Ident: "[]string", Nillable: true}},
			},
			Edges: []*load.Edge{
				{Name: "category", Type: "Category", Inverse: true, Unique: true, RefName: "table_cells"},
				{Name: "table", Type: "Table", Inverse: true, Unique: true, RefName: "table_cells"},
				{Name: "year", Type: "Year", Inverse: true, Unique: true, RefName: "table_cells"},
			},
		},
	}
}

// neo4jGraph creates a gen.Graph from the megatron schemas with Neo4j storage.
// Returns the graph, temp output directory, and any error.
func neo4jGraph(t *testing.T) (*Graph, string) {
	t.Helper()
	initTemplates()
	s, err := NewStorage("neo4j")
	if err != nil {
		t.Fatalf("NewStorage(neo4j) error = %v", err)
	}
	target := t.TempDir()
	g, err := NewGraph(&Config{
		Package: "entgo.io/ent/entc/gen/internal/megatron/ent",
		Target:  target,
		Storage: s,
	}, megatronSchemas()...)
	if err != nil {
		t.Fatalf("NewGraph error = %v", err)
	}
	return g, target
}

// --- H1: Schema structure tests ---

// TestMegatronSchema_EntityCount verifies that all 6 entity types are defined.
// Expected: Business, Document, Category, Year, Table, TableCell
func TestMegatronSchema_EntityCount(t *testing.T) {
	schemas := megatronSchemas()
	if got := len(schemas); got != 6 {
		t.Fatalf("schema count = %d, want 6", got)
	}
	expectedNames := []string{"Business", "Document", "Category", "Year", "Table", "TableCell"}
	for i, want := range expectedNames {
		if schemas[i].Name != want {
			t.Errorf("schema[%d].Name = %q, want %q", i, schemas[i].Name, want)
		}
	}
}

// TestMegatronSchema_BusinessFields verifies Business has 1 field: name (string).
// Expected: Business.Fields = [{name, TypeString}]
func TestMegatronSchema_BusinessFields(t *testing.T) {
	schemas := megatronSchemas()
	biz := schemas[0]
	if len(biz.Fields) != 1 {
		t.Fatalf("Business field count = %d, want 1", len(biz.Fields))
	}
	f := biz.Fields[0]
	if f.Name != "name" {
		t.Errorf("Business.Fields[0].Name = %q, want %q", f.Name, "name")
	}
	if f.Info.Type != field.TypeString {
		t.Errorf("Business.Fields[0].Type = %v, want TypeString", f.Info.Type)
	}
}

// TestMegatronSchema_BusinessEdges verifies Business has 3 forward edges:
// documents->Document, categories->Category, years->Year.
func TestMegatronSchema_BusinessEdges(t *testing.T) {
	schemas := megatronSchemas()
	biz := schemas[0]
	// Business has 3 forward (non-inverse) edges.
	forwardEdges := 0
	for _, e := range biz.Edges {
		if !e.Inverse {
			forwardEdges++
		}
	}
	if forwardEdges != 3 {
		t.Fatalf("Business forward edge count = %d, want 3", forwardEdges)
	}
	expectedEdges := []struct {
		name   string
		target string
	}{
		{"documents", "Document"},
		{"categories", "Category"},
		{"years", "Year"},
	}
	for i, want := range expectedEdges {
		if biz.Edges[i].Name != want.name {
			t.Errorf("Business.Edges[%d].Name = %q, want %q", i, biz.Edges[i].Name, want.name)
		}
		if biz.Edges[i].Type != want.target {
			t.Errorf("Business.Edges[%d].Type = %q, want %q", i, biz.Edges[i].Type, want.target)
		}
	}
}

// TestMegatronSchema_TableCellSliceFields verifies TableCell has slice fields:
// vector ([]float64) and categories ([]string) represented as TypeJSON.
func TestMegatronSchema_TableCellSliceFields(t *testing.T) {
	schemas := megatronSchemas()
	tc := schemas[5] // TableCell is the last schema
	if tc.Name != "TableCell" {
		t.Fatalf("expected TableCell, got %q", tc.Name)
	}
	if len(tc.Fields) != 3 {
		t.Fatalf("TableCell field count = %d, want 3", len(tc.Fields))
	}
	tests := []struct {
		name      string
		fieldType field.Type
		ident     string
	}{
		{"value", field.TypeFloat64, ""},
		{"vector", field.TypeJSON, "[]float64"},
		{"categories", field.TypeJSON, "[]string"},
	}
	for i, tt := range tests {
		f := tc.Fields[i]
		if f.Name != tt.name {
			t.Errorf("TableCell.Fields[%d].Name = %q, want %q", i, f.Name, tt.name)
		}
		if f.Info.Type != tt.fieldType {
			t.Errorf("TableCell.Fields[%d].Type = %v, want %v", i, f.Info.Type, tt.fieldType)
		}
		if tt.ident != "" && f.Info.Ident != tt.ident {
			t.Errorf("TableCell.Fields[%d].Ident = %q, want %q", i, f.Info.Ident, tt.ident)
		}
	}
}

// TestMegatronSchema_AllEdgesO2M verifies all forward edges are O2M
// (non-unique, non-inverse). The inverse edges are M2O (unique, inverse).
func TestMegatronSchema_AllEdgesO2M(t *testing.T) {
	schemas := megatronSchemas()
	for _, s := range schemas {
		for _, e := range s.Edges {
			if e.Inverse {
				// Inverse edges should be unique (M2O side).
				if !e.Unique {
					t.Errorf("%s.%s is inverse but not unique (expected M2O)", s.Name, e.Name)
				}
			} else {
				// Forward edges should not be unique (O2M side).
				if e.Unique {
					t.Errorf("%s.%s is forward but unique (expected O2M)", s.Name, e.Name)
				}
			}
		}
	}
}

// TestMegatronSchema_EdgeCount verifies each entity has the expected total edge count
// (forward + inverse edges).
func TestMegatronSchema_EdgeCount(t *testing.T) {
	schemas := megatronSchemas()
	expectedEdgeCounts := map[string]int{
		"Business":  3, // 3 forward (documents, categories, years)
		"Document":  4, // 1 inverse (business) + 3 forward (categories, years, tables)
		"Category":  3, // 2 inverse (business, document) + 1 forward (table_cells)
		"Year":      3, // 2 inverse (business, document) + 1 forward (table_cells)
		"Table":     2, // 1 inverse (document) + 1 forward (table_cells)
		"TableCell": 3, // 3 inverse (category, table, year)
	}
	for _, s := range schemas {
		want, ok := expectedEdgeCounts[s.Name]
		if !ok {
			t.Errorf("unexpected schema %q", s.Name)
			continue
		}
		if got := len(s.Edges); got != want {
			t.Errorf("%s edge count = %d, want %d", s.Name, got, want)
		}
	}
}

// --- H2: Code generation with Neo4j storage ---

// TestMegatronNeo4j_GraphCreation verifies that a gen.Graph can be created
// from the megatron schemas with Neo4j storage. The Graph creation resolves
// edges, foreign keys, and aliases. Expected: no error.
func TestMegatronNeo4j_GraphCreation(t *testing.T) {
	g, _ := neo4jGraph(t)
	if g == nil {
		t.Fatal("neo4jGraph returned nil graph")
	}
	if len(g.Nodes) != 6 {
		t.Errorf("graph node count = %d, want 6", len(g.Nodes))
	}
}

// TestMegatronNeo4j_CodeGeneration verifies that running Gen() on the megatron
// schema graph with Neo4j storage produces generated Go files without error.
// Expected: Gen() completes without error and output directory contains .go files.
func TestMegatronNeo4j_CodeGeneration(t *testing.T) {
	g, target := neo4jGraph(t)
	err := g.Gen()
	if err != nil {
		t.Fatalf("Graph.Gen() error = %v", err)
	}
	// Verify output directory has generated files.
	entries, err := os.ReadDir(target)
	if err != nil {
		t.Fatalf("ReadDir(%q) error = %v", target, err)
	}
	goFiles := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".go" {
			goFiles++
		}
	}
	if goFiles == 0 {
		t.Error("Gen() produced no .go files in output directory")
	}
}

// TestMegatronNeo4j_GeneratedEntityFiles verifies that code generation produces
// a Go file for each of the 6 entity types.
// Expected: business.go, document.go, category.go, year.go, table.go, tablecell.go
func TestMegatronNeo4j_GeneratedEntityFiles(t *testing.T) {
	g, target := neo4jGraph(t)
	if err := g.Gen(); err != nil {
		t.Fatalf("Graph.Gen() error = %v", err)
	}
	expectedFiles := []string{
		"business.go",
		"document.go",
		"category.go",
		"year.go",
		"table.go",
		"tablecell.go",
	}
	for _, name := range expectedFiles {
		path := filepath.Join(target, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected generated file %q not found", name)
		}
	}
}

// --- H3: Generated code compilation ---

// TestMegatronNeo4j_GeneratedCodeCompiles verifies that the generated Go code
// compiles without errors. This test runs 'go build' on the generated output
// directory. Expected: exit code 0 from 'go build'.
func TestMegatronNeo4j_GeneratedCodeCompiles(t *testing.T) {
	g, target := neo4jGraph(t)
	if err := g.Gen(); err != nil {
		t.Fatalf("Graph.Gen() error = %v", err)
	}
	// Compute the absolute path to the ent repo root (6 levels up from target).
	// target is a temp dir, but the repo root is relative to the source file location.
	repoRoot, err := filepath.Abs(filepath.Join(".", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	// Write a go.mod in the target so we can build it.
	gomod := "module entgo.io/ent/entc/gen/internal/megatron/ent\n\ngo 1.24\n\nrequire entgo.io/ent v0.0.0\n\nreplace entgo.io/ent => " + repoRoot + "\n"
	if err := os.WriteFile(filepath.Join(target, "go.mod"), []byte(gomod), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	// Run go build to verify the generated code compiles.
	// This is deferred to the green phase — this test should fail in red
	// because templates produce code that doesn't compile.
	// Run go mod tidy to resolve dependencies before building.
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = target
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed:\n%s", out)
	}
	cmd := testBuildCmd(target)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated code does not compile:\n%s", out)
	}
}

// --- H5: SQL-only assumptions in core codegen ---

// TestMegatronNeo4j_NoSQLAssumptions verifies that the generated code for
// Neo4j storage does not contain SQL-specific constructs like table names,
// column references, or migration code.
// Expected: generated files reference Neo4j/Cypher constructs, not SQL.
func TestMegatronNeo4j_NoSQLAssumptions(t *testing.T) {
	g, target := neo4jGraph(t)
	if err := g.Gen(); err != nil {
		t.Fatalf("Graph.Gen() error = %v", err)
	}
	// Check that no generated entity file contains SQL-specific patterns.
	entries, err := os.ReadDir(target)
	if err != nil {
		t.Fatalf("ReadDir error = %v", err)
	}
	sqlPatterns := []string{
		"sql.Selector",
		"sql.CreateSpec",
		"sql.UpdateSpec",
		"sql.DeleteSpec",
		"sqlgraph.",
		"CREATE TABLE",
		"ALTER TABLE",
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(target, entry.Name()))
		if err != nil {
			t.Errorf("ReadFile(%q) error = %v", entry.Name(), err)
			continue
		}
		content := string(data)
		for _, pattern := range sqlPatterns {
			if strings.Contains(content, pattern) {
				t.Errorf("generated file %q contains SQL pattern %q (expected Neo4j-only)", entry.Name(), pattern)
			}
		}
	}
}

// --- M4: KSUID ID generation verification ---

// TestMegatronNeo4j_IDTypeIsString verifies that the Neo4j graph resolves
// the ID type as TypeString (for KSUID), not the default TypeInt.
// Expected: every node in the graph has ID.Type == field.TypeString.
func TestMegatronNeo4j_IDTypeIsString(t *testing.T) {
	g, _ := neo4jGraph(t)
	for _, node := range g.Nodes {
		if !node.HasOneFieldID() {
			t.Errorf("%s: expected HasOneFieldID=true", node.Name)
			continue
		}
		if node.ID.Type.Type != field.TypeString {
			t.Errorf("%s: ID type = %v, want TypeString (KSUID)", node.Name, node.ID.Type.Type)
		}
	}
}

// TestMegatronNeo4j_IDFieldIsString verifies that the generated entity files
// declare the ID field as a string type, not int.
// Expected: generated code has `ID string` (not `ID int`).
func TestMegatronNeo4j_IDFieldIsString(t *testing.T) {
	g, target := neo4jGraph(t)
	if err := g.Gen(); err != nil {
		t.Fatalf("Graph.Gen() error = %v", err)
	}
	// Check each entity file for string ID field declaration.
	entityFiles := []string{
		"business.go", "document.go", "category.go",
		"year.go", "table.go", "tablecell.go",
	}
	for _, name := range entityFiles {
		data, err := os.ReadFile(filepath.Join(target, name))
		if err != nil {
			t.Errorf("ReadFile(%q) error = %v", name, err)
			continue
		}
		content := string(data)
		// The ID field should be typed as string for KSUID.
		if !strings.Contains(content, "ID string") {
			t.Errorf("%s: missing 'ID string' field declaration (KSUID requires string ID)", name)
		}
		// Should NOT contain int ID.
		if strings.Contains(content, "ID int") {
			t.Errorf("%s: contains 'ID int' — Neo4j should use string KSUID IDs", name)
		}
	}
}

// TestMegatronNeo4j_CreateCypherIncludesIDProperty verifies that the
// generated create method's Cypher includes 'id:' as the first property
// in the CREATE pattern.
// Expected: CREATE (n:Business {id: $p0, ...})
func TestMegatronNeo4j_CreateCypherIncludesIDProperty(t *testing.T) {
	g, target := neo4jGraph(t)
	if err := g.Gen(); err != nil {
		t.Fatalf("Graph.Gen() error = %v", err)
	}
	// Check the business_create.go file for Cypher with id property.
	createFiles := []string{
		"business_create.go", "document_create.go", "category_create.go",
		"year_create.go", "table_create.go", "tablecell_create.go",
	}
	for _, name := range createFiles {
		data, err := os.ReadFile(filepath.Join(target, name))
		if err != nil {
			t.Errorf("ReadFile(%q) error = %v", name, err)
			continue
		}
		content := string(data)
		// The CREATE Cypher builder should set the id property.
		if !strings.Contains(content, "id:") && !strings.Contains(content, `"id"`) {
			t.Errorf("%s: missing id property in CREATE Cypher pattern", name)
		}
	}
}

// TestMegatronNeo4j_QueryByIDUsesStringParam verifies that the Cypher builder
// produces a query-by-ID pattern where the ID parameter is a string placeholder.
// Expected: MATCH (n:Business) WHERE n.id = $p0 (with string "ksuid-..." param)
func TestMegatronNeo4j_QueryByIDUsesStringParam(t *testing.T) {
	// This tests the cypher.Builder contract for ID queries.
	// The Builder should accept string IDs and produce n.id = $p0 patterns.
	tests := []struct {
		label string
		id    string
	}{
		{"Business", "2QZ1KsuidExample01"},
		{"Document", "2QZ1KsuidExample02"},
		{"TableCell", "2QZ1KsuidExample03"},
	}
	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			b := cypher.New()
			idP := b.AddParam(tt.id)
			b.Match("(n:" + tt.label + ")")
			b.Where("n.id = " + idP)
			b.Return("n {.*}")

			query, params := b.Query()
			wantQuery := "MATCH (n:" + tt.label + ") WHERE n.id = $p0 RETURN n {.*}"
			if query != wantQuery {
				t.Errorf("query = %q\nwant  = %q", query, wantQuery)
			}
			// ID param should be a string, not an int.
			idVal, ok := params["p0"].(string)
			if !ok {
				t.Errorf("params[p0] type = %T, want string", params["p0"])
			}
			if idVal != tt.id {
				t.Errorf("params[p0] = %q, want %q", idVal, tt.id)
			}
		})
	}
}

// testBuildCmd returns an *exec.Cmd for running 'go build' in the given dir.
func testBuildCmd(dir string) *exec.Cmd {
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	return cmd
}
