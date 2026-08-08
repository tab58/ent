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
	beginTx(ctx context.Context, database string) (txRunner, error)
	close(ctx context.Context) error
}

// txRunner abstracts one explicit Neo4j transaction. All statements —
// reads and writes — run inside the same transaction; commit/rollback
// release the underlying session.
type txRunner interface {
	run(ctx context.Context, query string, params map[string]any) ([]*ndriver.Record, error)
	commit(ctx context.Context) error
	rollback(ctx context.Context) error
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

// beginTx opens a session pinned to one explicit transaction. The
// session lives exactly as long as the transaction: commit/rollback
// close it.
func (r *neo4jRunner) beginTx(ctx context.Context, database string) (txRunner, error) {
	session := r.db.NewSession(ctx, ndriver.SessionConfig{DatabaseName: database})
	tx, err := session.BeginTransaction(ctx)
	if err != nil {
		session.Close(ctx)
		return nil, err
	}
	return &neo4jTxRunner{session: session, tx: tx}, nil
}

// neo4jTxRunner is the production txRunner: one session, one explicit
// transaction.
type neo4jTxRunner struct {
	session ndriver.Session
	tx      ndriver.ExplicitTransaction
}

func (r *neo4jTxRunner) run(ctx context.Context, query string, params map[string]any) ([]*ndriver.Record, error) {
	res, err := r.tx.Run(ctx, query, params)
	if err != nil {
		return nil, err
	}
	return res.Collect(ctx)
}

func (r *neo4jTxRunner) commit(ctx context.Context) error {
	defer r.session.Close(ctx)
	return r.tx.Commit(ctx)
}

func (r *neo4jTxRunner) rollback(ctx context.Context) error {
	defer r.session.Close(ctx)
	return r.tx.Rollback(ctx)
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

// Tx begins a real explicit transaction: every Exec/Query on the
// returned Tx runs inside it, and Commit/Rollback apply or discard
// them atomically. (Replaces the earlier NopTx placeholder, under
// which every statement auto-committed individually.)
func (d *Driver) Tx(ctx context.Context) (dialect.Tx, error) {
	runner, err := d.runner.beginTx(ctx, d.database)
	if err != nil {
		return nil, fmt.Errorf("neo4j: begin transaction: %w", err)
	}
	return &Tx{runner: runner}, nil
}

// ErrTxDone is returned by operations on a transaction that has
// already been committed or rolled back (mirrors database/sql).
var ErrTxDone = errors.New("neo4j: transaction has already been committed or rolled back")

// Tx is a dialect.Tx over one Neo4j explicit transaction.
type Tx struct {
	runner txRunner
	// done marks a finished transaction. committed additionally
	// tolerates the Rollback-after-Commit pattern Ent's generated
	// code and defer blocks rely on.
	done      bool
	committed bool
}

// Exec runs a write statement inside the transaction.
func (t *Tx) Exec(ctx context.Context, query string, args, v any) error {
	return t.run(ctx, query, args, v)
}

// Query runs a read statement inside the transaction. Neo4j explicit
// transactions do not route reads and writes separately — both run on
// the same transaction.
func (t *Tx) Query(ctx context.Context, query string, args, v any) error {
	return t.run(ctx, query, args, v)
}

func (t *Tx) run(ctx context.Context, query string, args, v any) error {
	if t.done {
		return ErrTxDone
	}
	params, res, err := validateArgs(args, v)
	if err != nil {
		return err
	}
	records, err := t.runner.run(ctx, query, params)
	if err != nil {
		return fmt.Errorf("neo4j: tx run: %w", err)
	}
	res.records = records
	return nil
}

// Commit commits the transaction. The dialect.Tx contract (database/sql
// driver.Tx) has no context — the driver uses a background context to
// release the session.
func (t *Tx) Commit() error {
	if t.done {
		return ErrTxDone
	}
	t.done = true
	ctx := context.Background()
	if err := t.runner.commit(ctx); err != nil {
		return fmt.Errorf("neo4j: tx commit: %w", err)
	}
	t.committed = true
	return nil
}

// Rollback discards the transaction. After Commit — successful or
// failed — it is a nil no-op, so `defer tx.Rollback()` and Ent's
// rollback-on-error paths behave like database/sql.
func (t *Tx) Rollback() error {
	if t.done {
		return nil
	}
	t.done = true
	if err := t.runner.rollback(context.Background()); err != nil {
		return fmt.Errorf("neo4j: tx rollback: %w", err)
	}
	return nil
}

// compile-time check that Tx implements dialect.Tx.
var _ dialect.Tx = (*Tx)(nil)

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
