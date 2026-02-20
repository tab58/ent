// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package neo4j

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect"
	ndriver "github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

// queryRunner abstracts Neo4j session management for testability.
type queryRunner interface {
	executeRead(ctx context.Context, database, query string, params map[string]any) ([]*ndriver.Record, error)
	executeWrite(ctx context.Context, database, query string, params map[string]any) ([]*ndriver.Record, error)
	close(ctx context.Context) error
}

// neo4jRunner is the production queryRunner backed by a real neo4j.Driver.
type neo4jRunner struct {
	db ndriver.Driver
}

func (r *neo4jRunner) executeRead(ctx context.Context, database, query string, params map[string]any) ([]*ndriver.Record, error) {
	session := r.db.NewSession(ctx, ndriver.SessionConfig{DatabaseName: database})
	defer session.Close(ctx)
	result, err := session.ExecuteRead(ctx, func(tx ndriver.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		return res.Collect(ctx)
	})
	if err != nil {
		return nil, err
	}
	return result.([]*ndriver.Record), nil
}

func (r *neo4jRunner) executeWrite(ctx context.Context, database, query string, params map[string]any) ([]*ndriver.Record, error) {
	session := r.db.NewSession(ctx, ndriver.SessionConfig{DatabaseName: database})
	defer session.Close(ctx)
	result, err := session.ExecuteWrite(ctx, func(tx ndriver.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		return res.Collect(ctx)
	})
	if err != nil {
		return nil, err
	}
	return result.([]*ndriver.Record), nil
}

func (r *neo4jRunner) close(ctx context.Context) error {
	return r.db.Close(ctx)
}

// Driver is a dialect.Driver implementation for Neo4j graph database.
// It wraps a neo4j.Driver and routes queries through
// ExecuteRead/ExecuteWrite on auto-managed sessions.
type Driver struct {
	runner   queryRunner
	database string
}

// NewDriver returns a new dialect.Driver for Neo4j.
func NewDriver(db ndriver.Driver, database string) *Driver {
	return &Driver{runner: &neo4jRunner{db: db}, database: database}
}

// validateArgs checks that args is map[string]any and v is *Response.
func validateArgs(args, v any) (map[string]any, *Response, error) {
	params, ok := args.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("neo4j: invalid type for args: expected map[string]any, got %T", args)
	}
	res, ok := v.(*Response)
	if !ok {
		return nil, nil, fmt.Errorf("neo4j: invalid type for result: expected *Response, got %T", v)
	}
	return params, res, nil
}

// Exec executes a write Cypher statement. args must be map[string]any,
// v must be *Response.
func (d *Driver) Exec(ctx context.Context, query string, args, v any) error {
	params, res, err := validateArgs(args, v)
	if err != nil {
		return err
	}
	records, err := d.runner.executeWrite(ctx, d.database, query, params)
	if err != nil {
		return fmt.Errorf("neo4j: exec: %w", err)
	}
	res.records = records
	return nil
}

// Query executes a read Cypher statement. args must be map[string]any,
// v must be *Response.
func (d *Driver) Query(ctx context.Context, query string, args, v any) error {
	params, res, err := validateArgs(args, v)
	if err != nil {
		return err
	}
	records, err := d.runner.executeRead(ctx, d.database, query, params)
	if err != nil {
		return fmt.Errorf("neo4j: query: %w", err)
	}
	res.records = records
	return nil
}

// Tx returns a NopTx wrapping the driver. Real transactions deferred.
func (d *Driver) Tx(_ context.Context) (dialect.Tx, error) {
	return dialect.NopTx(d), nil
}

// Close closes the underlying Neo4j driver connection.
func (d *Driver) Close() error {
	if d.runner == nil {
		return errors.New("neo4j: driver connection is nil")
	}
	return d.runner.close(context.Background())
}

// Dialect returns the dialect name.
func (d *Driver) Dialect() string {
	return dialect.Neo4j
}

// compile-time check that Driver implements dialect.Driver.
var _ dialect.Driver = (*Driver)(nil)
