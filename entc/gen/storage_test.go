// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package gen

import (
	"reflect"
	"testing"

	"entgo.io/ent/dialect/neo4j/cypher"
)

func TestNewStorage_Neo4j(t *testing.T) {
	s, err := NewStorage("neo4j")
	if err != nil {
		t.Fatalf("NewStorage(neo4j) error = %v", err)
	}
	if s.Name != "neo4j" {
		t.Errorf("Name = %q, want %q", s.Name, "neo4j")
	}
	if s.IdentName != "Neo4j" {
		t.Errorf("IdentName = %q, want %q", s.IdentName, "Neo4j")
	}
}

func TestNewStorage_Neo4j_Builder(t *testing.T) {
	s, err := NewStorage("neo4j")
	if err != nil {
		t.Fatalf("NewStorage(neo4j) error = %v", err)
	}
	want := reflect.TypeOf(&cypher.Builder{})
	if s.Builder != want {
		t.Errorf("Builder = %v, want %v", s.Builder, want)
	}
}

func TestNewStorage_Neo4j_Dialects(t *testing.T) {
	s, err := NewStorage("neo4j")
	if err != nil {
		t.Fatalf("NewStorage(neo4j) error = %v", err)
	}
	if len(s.Dialects) != 1 || s.Dialects[0] != "dialect.Neo4j" {
		t.Errorf("Dialects = %v, want [dialect.Neo4j]", s.Dialects)
	}
}

func TestNewStorage_Neo4j_Imports(t *testing.T) {
	s, err := NewStorage("neo4j")
	if err != nil {
		t.Fatalf("NewStorage(neo4j) error = %v", err)
	}
	wantImports := map[string]bool{
		"entgo.io/ent/dialect/neo4j":        true,
		"entgo.io/ent/dialect/neo4j/cypher": true,
	}
	for _, imp := range s.Imports {
		if !wantImports[imp] {
			t.Errorf("unexpected import %q", imp)
		}
		delete(wantImports, imp)
	}
	for imp := range wantImports {
		t.Errorf("missing import %q", imp)
	}
}

func TestNewStorage_Neo4j_SchemaMode(t *testing.T) {
	s, err := NewStorage("neo4j")
	if err != nil {
		t.Fatalf("NewStorage(neo4j) error = %v", err)
	}
	if !s.SchemaMode.Support(Unique) {
		t.Error("SchemaMode should support Unique")
	}
	if !s.SchemaMode.Support(Indexes) {
		t.Error("SchemaMode should support Indexes")
	}
	if !s.SchemaMode.Support(Cascade) {
		t.Error("SchemaMode should support Cascade")
	}
	if s.SchemaMode.Support(Migrate) {
		t.Error("SchemaMode should NOT support Migrate")
	}
}

func TestNewStorage_Neo4j_OpCode(t *testing.T) {
	s, err := NewStorage("neo4j")
	if err != nil {
		t.Fatalf("NewStorage(neo4j) error = %v", err)
	}
	tests := []struct {
		op   Op
		want string
	}{
		{EQ, "EQ"},
		{NEQ, "NEQ"},
		{GT, "GT"},
		{GTE, "GTE"},
		{LT, "LT"},
		{LTE, "LTE"},
		{IsNil, "IsNull"},
		{NotNil, "NotNull"},
		{In, "In"},
		{NotIn, "NotIn"},
		{Contains, "Contains"},
		{HasPrefix, "StartsWith"},
		{HasSuffix, "EndsWith"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := s.OpCode(tt.op)
			if got != tt.want {
				t.Errorf("OpCode(%v) = %q, want %q", tt.op, got, tt.want)
			}
		})
	}
}

func TestNewStorage_Neo4j_Init(t *testing.T) {
	s, err := NewStorage("neo4j")
	if err != nil {
		t.Fatalf("NewStorage(neo4j) error = %v", err)
	}
	if s.Init == nil {
		t.Fatal("Init should not be nil")
	}
	// Init should be a no-op that returns nil.
	if err := s.Init(nil); err != nil {
		t.Errorf("Init() = %v, want nil", err)
	}
}

func TestNewStorage_InvalidDriver(t *testing.T) {
	_, err := NewStorage("invalid")
	if err == nil {
		t.Error("NewStorage(invalid) should return an error")
	}
}

func TestNewStorage_AllDrivers(t *testing.T) {
	// Verify all expected storage drivers are registered.
	names := []string{"sql", "gremlin", "neo4j"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			s, err := NewStorage(name)
			if err != nil {
				t.Fatalf("NewStorage(%q) error = %v", name, err)
			}
			if s.Name != name {
				t.Errorf("Name = %q, want %q", s.Name, name)
			}
		})
	}
}
