// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

// Package cypher provides a declarative Cypher query builder for Neo4j.
package cypher

import (
	"fmt"
	"maps"
	"strings"
)

// clauseKind identifies the type of a Cypher clause in the ordered list.
type clauseKind int

const (
	kindMatch clauseKind = iota
	kindWith
	kindWhere
	kindCreate
	kindMerge
	kindSet
	kindOnCreateSet
	kindOnMatchSet
	kindRemove
	kindDelete
)

// clause is a single Cypher clause with its kind and raw content.
type clause struct {
	kind    clauseKind
	content string
}

// Builder assembles Cypher query clauses (MATCH, WHERE, CREATE, etc.)
// and manages parameterized values. It is the Neo4j equivalent of
// dsl.Traversal for Gremlin and sql.Selector for SQL.
//
// All clauses live in a single ordered stream and are emitted in insertion
// order, so a WHERE filters the MATCH/WITH segment it was added after —
// not the head of the query. Consecutive WHERE conditions are merged with
// AND, consecutive SET expressions with commas.
type Builder struct {
	clauses []clause
	ret     []string
	orderBy []string
	skip    *int
	limit   *int
	params  map[string]any
	paramN  int
}

// New returns a new empty Builder.
func New() *Builder {
	return &Builder{
		params: make(map[string]any),
	}
}

// Match appends a MATCH pattern clause.
func (b *Builder) Match(pattern string) *Builder {
	b.clauses = append(b.clauses, clause{kindMatch, pattern})
	return b
}

// With appends a WITH clause that rebinds the working row
// (e.g. With("m AS n") or With("n")).
func (b *Builder) With(expr string) *Builder {
	b.clauses = append(b.clauses, clause{kindWith, expr})
	return b
}

// Where appends a WHERE condition. The condition applies to the clause
// segment it follows; consecutive conditions are merged with AND.
func (b *Builder) Where(cond string) *Builder {
	b.clauses = append(b.clauses, clause{kindWhere, cond})
	return b
}

// Create appends a CREATE pattern clause.
func (b *Builder) Create(pattern string) *Builder {
	b.clauses = append(b.clauses, clause{kindCreate, pattern})
	return b
}

// Merge appends a MERGE pattern clause.
func (b *Builder) Merge(pattern string) *Builder {
	b.clauses = append(b.clauses, clause{kindMerge, pattern})
	return b
}

// Set appends a SET expression. Consecutive expressions are comma-merged.
func (b *Builder) Set(expr string) *Builder {
	b.clauses = append(b.clauses, clause{kindSet, expr})
	return b
}

// OnCreateSet appends an ON CREATE SET expression for a preceding MERGE.
// Consecutive expressions are comma-merged.
func (b *Builder) OnCreateSet(expr string) *Builder {
	b.clauses = append(b.clauses, clause{kindOnCreateSet, expr})
	return b
}

// OnMatchSet appends an ON MATCH SET expression for a preceding MERGE.
// Consecutive expressions are comma-merged.
func (b *Builder) OnMatchSet(expr string) *Builder {
	b.clauses = append(b.clauses, clause{kindOnMatchSet, expr})
	return b
}

// Remove appends a REMOVE expression.
func (b *Builder) Remove(expr string) *Builder {
	b.clauses = append(b.clauses, clause{kindRemove, expr})
	return b
}

// Delete appends a DELETE expression.
func (b *Builder) Delete(expr string) *Builder {
	b.clauses = append(b.clauses, clause{kindDelete, expr})
	return b
}

// DetachDelete appends a DETACH DELETE expression.
func (b *Builder) DetachDelete(expr string) *Builder {
	b.clauses = append(b.clauses, clause{kindDelete, "DETACH " + expr})
	return b
}

// Return sets the RETURN expressions.
func (b *Builder) Return(exprs ...string) *Builder {
	b.ret = append(b.ret, exprs...)
	return b
}

// OrderBy appends an ORDER BY expression.
func (b *Builder) OrderBy(expr string) *Builder {
	b.orderBy = append(b.orderBy, expr)
	return b
}

// Skip sets the SKIP value for pagination.
func (b *Builder) Skip(n int) *Builder {
	b.skip = &n
	return b
}

// Limit sets the LIMIT value for pagination.
func (b *Builder) Limit(n int) *Builder {
	b.limit = &n
	return b
}

// AddParam adds an anonymous parameter and returns its placeholder name ($pN).
func (b *Builder) AddParam(value any) string {
	name := fmt.Sprintf("p%d", b.paramN)
	b.paramN++
	b.params[name] = value
	return "$" + name
}

// SetParam sets a named parameter.
func (b *Builder) SetParam(name string, value any) {
	b.params[name] = value
}

// WhereClauses returns the raw WHERE condition strings without the
// WHERE keyword, in insertion order. Used by predicate combinators
// (AND/OR/NOT) to extract conditions from sub-builders without
// generating nested WHERE keywords.
func (b *Builder) WhereClauses() []string {
	var conds []string
	for _, c := range b.clauses {
		if c.kind == kindWhere {
			conds = append(conds, c.content)
		}
	}
	return conds
}

// Params returns the parameter map. Used by predicate combinators to
// transfer parameters from sub-builders to the parent builder.
func (b *Builder) Params() map[string]any {
	return b.params
}

// CollectWhere applies fn to this builder, captures the WHERE conditions
// that fn added, removes them from the builder, and returns them.
// Parameters added by fn remain in the builder with correct sequencing.
// This is used by predicate combinators (AND/OR/NOT) to capture
// individual conditions for recombination without param counter collisions.
func (b *Builder) CollectWhere(fn func(*Builder)) []string {
	before := len(b.clauses)
	fn(b)
	var conds []string
	kept := b.clauses[:before]
	for _, c := range b.clauses[before:] {
		if c.kind == kindWhere {
			conds = append(conds, c.content)
		} else {
			kept = append(kept, c)
		}
	}
	b.clauses = kept
	return conds
}

// Query returns the assembled Cypher query string and its parameters map.
//
// Clauses are emitted in insertion order. Runs of consecutive WHERE
// conditions are merged into one WHERE joined with AND; runs of SET,
// ON CREATE SET, and ON MATCH SET expressions are comma-merged.
// Terminal clauses (RETURN, ORDER BY, SKIP, LIMIT) come last.
func (b *Builder) Query() (string, map[string]any) {
	var parts []string

	for i := 0; i < len(b.clauses); i++ {
		c := b.clauses[i]
		// Merge runs of same-kind mergeable clauses.
		if sep, kw, ok := mergeSpec(c.kind); ok {
			exprs := []string{c.content}
			for i+1 < len(b.clauses) && b.clauses[i+1].kind == c.kind {
				i++
				exprs = append(exprs, b.clauses[i].content)
			}
			parts = append(parts, kw+" "+strings.Join(exprs, sep))
			continue
		}
		switch c.kind {
		case kindMatch:
			parts = append(parts, matchPrefix(c.content))
		case kindWith:
			parts = append(parts, "WITH "+c.content)
		case kindCreate:
			parts = append(parts, "CREATE "+c.content)
		case kindMerge:
			parts = append(parts, "MERGE "+c.content)
		case kindRemove:
			parts = append(parts, "REMOVE "+c.content)
		case kindDelete:
			if after, ok := strings.CutPrefix(c.content, "DETACH "); ok {
				parts = append(parts, "DETACH DELETE "+after)
			} else {
				parts = append(parts, "DELETE "+c.content)
			}
		}
	}

	// Terminal clauses.
	if len(b.ret) > 0 {
		parts = append(parts, "RETURN "+strings.Join(b.ret, ", "))
	}
	if len(b.orderBy) > 0 {
		parts = append(parts, "ORDER BY "+strings.Join(b.orderBy, ", "))
	}
	if b.skip != nil {
		parts = append(parts, fmt.Sprintf("SKIP %d", *b.skip))
	}
	if b.limit != nil {
		parts = append(parts, fmt.Sprintf("LIMIT %d", *b.limit))
	}

	return strings.Join(parts, " "), b.params
}

// mergeSpec returns the join separator and keyword for clause kinds whose
// consecutive runs are merged into a single emitted clause.
func mergeSpec(k clauseKind) (sep, keyword string, ok bool) {
	switch k {
	case kindWhere:
		return " AND ", "WHERE", true
	case kindSet:
		return ", ", "SET", true
	case kindOnCreateSet:
		return ", ", "ON CREATE SET", true
	case kindOnMatchSet:
		return ", ", "ON MATCH SET", true
	}
	return "", "", false
}

// matchPrefix prepends "MATCH " to content unless it already starts with
// "OPTIONAL MATCH " or "WITH " (self-contained clause prefixes).
//
// Deprecated: the "WITH " sniffing exists only for backward compatibility
// with callers that smuggle compound clauses through Match. New code should
// use With, Match, and Where as separate calls.
func matchPrefix(content string) string {
	if strings.HasPrefix(content, "OPTIONAL MATCH ") || strings.HasPrefix(content, "WITH ") {
		return content
	}
	return "MATCH " + content
}

// Clone returns a deep copy of the Builder.
func (b *Builder) Clone() *Builder {
	if b == nil {
		return nil
	}
	c := &Builder{
		clauses: clausesCopy(b.clauses),
		ret:     sliceCopy(b.ret),
		orderBy: sliceCopy(b.orderBy),
		paramN:  b.paramN,
		params:  make(map[string]any, len(b.params)),
	}
	if b.skip != nil {
		v := *b.skip
		c.skip = &v
	}
	if b.limit != nil {
		v := *b.limit
		c.limit = &v
	}
	maps.Copy(c.params, b.params)
	return c
}

// clausesCopy returns a shallow copy of the clause slice. The clause struct
// contains only value types so a shallow copy is sufficient.
func clausesCopy(s []clause) []clause {
	if s == nil {
		return nil
	}
	c := make([]clause, len(s))
	copy(c, s)
	return c
}

func sliceCopy(s []string) []string {
	if s == nil {
		return nil
	}
	c := make([]string, len(s))
	copy(c, s)
	return c
}
