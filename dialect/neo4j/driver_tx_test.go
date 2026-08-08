// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package neo4j

import (
	"context"
	"errors"
	"testing"

	"entgo.io/ent/dialect"
	ndriver "github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

// mockTxRunner is a test double for the txRunner interface.
type mockTxRunner struct {
	records    []*ndriver.Record
	runErr     error
	commitErr  error
	queries    []string
	lastParams map[string]any
	committed  bool
	rolledBack bool
}

func (m *mockTxRunner) run(_ context.Context, query string, params map[string]any) ([]*ndriver.Record, error) {
	m.queries = append(m.queries, query)
	m.lastParams = params
	if m.runErr != nil {
		return nil, m.runErr
	}
	return m.records, nil
}

func (m *mockTxRunner) commit(_ context.Context) error {
	m.committed = true
	return m.commitErr
}

func (m *mockTxRunner) rollback(_ context.Context) error {
	m.rolledBack = true
	return nil
}

func txDriverPair() (*Driver, *mockRunner, *mockTxRunner) {
	tr := &mockTxRunner{}
	r := &mockRunner{txRunner: tr}
	return &Driver{runner: r, database: "neo4j"}, r, tr
}

func TestTx_ImplementsDialectTx(t *testing.T) {
	var _ dialect.Tx = (*Tx)(nil)
}

func TestTx_ExecAndQueryRouteThroughTransaction(t *testing.T) {
	drv, r, tr := txDriverPair()
	tx, err := drv.Tx(context.Background())
	if err != nil {
		t.Fatalf("Tx() error = %v", err)
	}
	if r.beginDatabase != "neo4j" {
		t.Errorf("beginTx database = %q, want neo4j", r.beginDatabase)
	}

	res := &Response{}
	if err := tx.Exec(context.Background(), "CREATE (n:Test)", map[string]any{"a": 1}, res); err != nil {
		t.Fatalf("Exec error = %v", err)
	}
	if err := tx.Query(context.Background(), "MATCH (n) RETURN n", map[string]any{}, res); err != nil {
		t.Fatalf("Query error = %v", err)
	}
	if len(tr.queries) != 2 {
		t.Fatalf("transaction saw %d statements, want 2 — statements must NOT go through session runners", len(tr.queries))
	}
	if r.lastQuery != "" {
		t.Errorf("session runner saw query %q — must be untouched during a tx", r.lastQuery)
	}
}

func TestTx_CommitOnce(t *testing.T) {
	drv, _, tr := txDriverPair()
	tx, _ := drv.Tx(context.Background())
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit error = %v", err)
	}
	if !tr.committed {
		t.Fatal("underlying transaction not committed")
	}
	// Exec after commit fails loudly.
	if err := tx.Exec(context.Background(), "CREATE (n)", map[string]any{}, &Response{}); err == nil {
		t.Error("Exec after Commit must error")
	}
	// Second Commit is a no-op error-free call is NOT allowed — it must
	// error like database/sql's ErrTxDone.
	if err := tx.Commit(); err == nil {
		t.Error("second Commit must error")
	}
}

func TestTx_RollbackAfterCommitIsNoop(t *testing.T) {
	// Ent's generated code (and defer patterns) call Rollback after a
	// successful Commit — that must be a nil no-op, not an error.
	drv, _, tr := txDriverPair()
	tx, _ := drv.Tx(context.Background())
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit error = %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Errorf("Rollback after Commit = %v, want nil", err)
	}
	if tr.rolledBack {
		t.Error("underlying rollback must not run after commit")
	}
}

func TestTx_Rollback(t *testing.T) {
	drv, _, tr := txDriverPair()
	tx, _ := drv.Tx(context.Background())
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback error = %v", err)
	}
	if !tr.rolledBack {
		t.Fatal("underlying transaction not rolled back")
	}
	if err := tx.Exec(context.Background(), "CREATE (n)", map[string]any{}, &Response{}); err == nil {
		t.Error("Exec after Rollback must error")
	}
}

func TestTx_CommitErrorPropagates(t *testing.T) {
	drv, _, tr := txDriverPair()
	tr.commitErr = errors.New("boom")
	tx, _ := drv.Tx(context.Background())
	if err := tx.Commit(); err == nil {
		t.Fatal("Commit must propagate the error")
	}
	// A failed commit leaves the tx done: Ent rolls back on commit
	// failure, and that must be tolerated.
	if err := tx.Rollback(); err != nil {
		t.Errorf("Rollback after failed Commit = %v, want nil", err)
	}
}

func TestTx_BeginError(t *testing.T) {
	r := &mockRunner{beginErr: errors.New("no session")}
	drv := &Driver{runner: r, database: "neo4j"}
	if _, err := drv.Tx(context.Background()); err == nil {
		t.Fatal("Tx must propagate beginTx errors")
	}
}
