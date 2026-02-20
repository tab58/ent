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

// Builder assembles Cypher query clauses (MATCH, WHERE, CREATE, etc.)
// and manages parameterized values. It is the Neo4j equivalent of
// dsl.Traversal for Gremlin and sql.Selector for SQL.
type Builder struct {
	match   []string
	where   []string
	create  []string
	merge   []string
	set     []string
	remove  []string
	del     []string
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
	b.match = append(b.match, pattern)
	return b
}

// Where appends a WHERE condition.
func (b *Builder) Where(cond string) *Builder {
	b.where = append(b.where, cond)
	return b
}

// Create appends a CREATE pattern clause.
func (b *Builder) Create(pattern string) *Builder {
	b.create = append(b.create, pattern)
	return b
}

// Merge appends a MERGE pattern clause.
func (b *Builder) Merge(pattern string) *Builder {
	b.merge = append(b.merge, pattern)
	return b
}

// Set appends a SET expression.
func (b *Builder) Set(expr string) *Builder {
	b.set = append(b.set, expr)
	return b
}

// Remove appends a REMOVE expression.
func (b *Builder) Remove(expr string) *Builder {
	b.remove = append(b.remove, expr)
	return b
}

// Delete appends a DELETE expression.
func (b *Builder) Delete(expr string) *Builder {
	b.del = append(b.del, expr)
	return b
}

// DetachDelete appends a DETACH DELETE expression.
func (b *Builder) DetachDelete(expr string) *Builder {
	b.del = append(b.del, "DETACH "+expr)
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
// WHERE keyword. Used by predicate combinators (AND/OR/NOT) to extract
// conditions from sub-builders without generating nested WHERE keywords.
func (b *Builder) WhereClauses() []string {
	return b.where
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
	before := len(b.where)
	fn(b)
	added := b.where[before:]
	result := make([]string, len(added))
	copy(result, added)
	b.where = b.where[:before]
	return result
}

// Query returns the assembled Cypher query string and its parameters map.
func (b *Builder) Query() (string, map[string]any) {
	var parts []string

	if len(b.match) > 0 {
		for _, m := range b.match {
			parts = append(parts, "MATCH "+m)
		}
	}
	if len(b.where) > 0 {
		parts = append(parts, "WHERE "+strings.Join(b.where, " AND "))
	}
	if len(b.create) > 0 {
		for _, c := range b.create {
			parts = append(parts, "CREATE "+c)
		}
	}
	if len(b.merge) > 0 {
		for _, m := range b.merge {
			parts = append(parts, "MERGE "+m)
		}
	}
	if len(b.set) > 0 {
		parts = append(parts, "SET "+strings.Join(b.set, ", "))
	}
	if len(b.remove) > 0 {
		for _, r := range b.remove {
			parts = append(parts, "REMOVE "+r)
		}
	}
	if len(b.del) > 0 {
		for _, d := range b.del {
			// DetachDelete already prepends "DETACH ", regular delete does not.
			if after, ok := strings.CutPrefix(d, "DETACH "); ok {
				parts = append(parts, "DETACH DELETE "+after)
			} else {
				parts = append(parts, "DELETE "+d)
			}
		}
	}
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

// Clone returns a deep copy of the Builder.
func (b *Builder) Clone() *Builder {
	if b == nil {
		return nil
	}
	c := &Builder{
		match:   sliceCopy(b.match),
		where:   sliceCopy(b.where),
		create:  sliceCopy(b.create),
		merge:   sliceCopy(b.merge),
		set:     sliceCopy(b.set),
		remove:  sliceCopy(b.remove),
		del:     sliceCopy(b.del),
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

func sliceCopy(s []string) []string {
	if s == nil {
		return nil
	}
	c := make([]string, len(s))
	copy(c, s)
	return c
}
