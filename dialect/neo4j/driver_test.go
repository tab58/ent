// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package neo4j

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"entgo.io/ent/dialect"
	ndriver "github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

// mockNeo4jDB is a minimal mock of ndriver.Driver for testing neo4jRunner.
type mockNeo4jDB struct {
	ndriver.Driver // embed to satisfy interface; only override methods we test
	closeErr       error
	closeCalled    bool
}

func (m *mockNeo4jDB) Close(_ context.Context) error {
	m.closeCalled = true
	return m.closeErr
}

// mockRunner is a test double for the queryRunner interface.
type mockRunner struct {
	readRecords  []*ndriver.Record
	writeRecords []*ndriver.Record
	readErr      error
	writeErr     error
	closeErr     error
	closed       bool
	lastQuery    string
	lastParams   map[string]any
	lastDatabase string
}

func (m *mockRunner) executeRead(_ context.Context, database, query string, params map[string]any) ([]*ndriver.Record, error) {
	m.lastDatabase = database
	m.lastQuery = query
	m.lastParams = params
	if m.readErr != nil {
		return nil, m.readErr
	}
	return m.readRecords, nil
}

func (m *mockRunner) executeWrite(_ context.Context, database, query string, params map[string]any) ([]*ndriver.Record, error) {
	m.lastDatabase = database
	m.lastQuery = query
	m.lastParams = params
	if m.writeErr != nil {
		return nil, m.writeErr
	}
	return m.writeRecords, nil
}

func (m *mockRunner) close(_ context.Context) error {
	m.closed = true
	return m.closeErr
}

func TestDriver_ImplementsDialectDriver(t *testing.T) {
	var _ dialect.Driver = (*Driver)(nil)
}

func TestDriver_Dialect(t *testing.T) {
	d := &Driver{}
	got := d.Dialect()
	if got != dialect.Neo4j {
		t.Errorf("Dialect() = %q, want %q", got, dialect.Neo4j)
	}
}

func TestDriver_Tx_ReturnsNopTx(t *testing.T) {
	d := &Driver{}
	tx, err := d.Tx(context.Background())
	if err != nil {
		t.Fatalf("Tx() error = %v", err)
	}
	if tx == nil {
		t.Fatal("Tx() returned nil, want NopTx")
	}
	if err := tx.Commit(); err != nil {
		t.Errorf("NopTx.Commit() error = %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Errorf("NopTx.Rollback() error = %v", err)
	}
}

func TestDriver_Exec_InvalidArgs(t *testing.T) {
	d := &Driver{}
	var res Response
	err := d.Exec(context.Background(), "MATCH (n) RETURN n", "bad-args", &res)
	if err == nil {
		t.Error("Exec() with invalid args should return an error")
	}
}

func TestDriver_Exec_InvalidResult(t *testing.T) {
	d := &Driver{}
	var badResult int
	err := d.Exec(context.Background(), "MATCH (n) RETURN n", map[string]any{}, &badResult)
	if err == nil {
		t.Error("Exec() with invalid result type should return an error")
	}
}

func TestDriver_Query_InvalidArgs(t *testing.T) {
	d := &Driver{}
	var res Response
	err := d.Query(context.Background(), "MATCH (n) RETURN n", "bad-args", &res)
	if err == nil {
		t.Error("Query() with invalid args should return an error")
	}
}

func TestDriver_Query_InvalidResult(t *testing.T) {
	d := &Driver{}
	var badResult string
	err := d.Query(context.Background(), "MATCH (n) RETURN n", map[string]any{}, &badResult)
	if err == nil {
		t.Error("Query() with invalid result type should return an error")
	}
}

func TestDriver_Close_WithNilRunner(t *testing.T) {
	d := &Driver{}
	err := d.Close()
	if err == nil {
		t.Error("Close() with nil runner should return an error")
	}
}

func TestDriver_Exec_TypeAssertions(t *testing.T) {
	tests := []struct {
		name    string
		args    any
		v       any
		wantErr string
	}{
		{
			name:    "args is nil",
			args:    nil,
			v:       &Response{},
			wantErr: "invalid type",
		},
		{
			name:    "args is slice instead of map",
			args:    []any{"a", "b"},
			v:       &Response{},
			wantErr: "invalid type",
		},
		{
			name:    "v is nil",
			args:    map[string]any{},
			v:       nil,
			wantErr: "invalid type",
		},
		{
			name:    "v is wrong pointer type",
			args:    map[string]any{},
			v:       new(int),
			wantErr: "invalid type",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Driver{}
			err := d.Exec(context.Background(), "RETURN 1", tt.args, tt.v)
			if err == nil {
				t.Errorf("Exec(%v, %v) should return error containing %q", tt.args, tt.v, tt.wantErr)
			}
		})
	}
}

func TestNewDriver(t *testing.T) {
	d := NewDriver(nil, "testdb")
	if d == nil {
		t.Fatal("NewDriver returned nil")
	}
	if d.database != "testdb" {
		t.Errorf("database = %q, want %q", d.database, "testdb")
	}
}

func TestDriver_ExecQueryErrorMessages(t *testing.T) {
	d := &Driver{}
	var res Response
	err := d.Exec(context.Background(), "RETURN 1", 42, &res)
	if err == nil {
		t.Fatal("expected error for non-map args")
	}
	msg := fmt.Sprint(err)
	if msg == "" {
		t.Error("error message should not be empty")
	}
}

func TestDriver_Exec_Success(t *testing.T) {
	mock := &mockRunner{
		writeRecords: []*ndriver.Record{
			{Keys: []string{"n"}, Values: []any{map[string]any{"id": "ksuid-1", "name": "alice"}}},
		},
	}
	d := &Driver{runner: mock, database: "testdb"}
	var res Response
	params := map[string]any{"name": "alice"}
	err := d.Exec(context.Background(), "CREATE (n:User {name: $name}) RETURN n", params, &res)
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	if mock.lastQuery != "CREATE (n:User {name: $name}) RETURN n" {
		t.Errorf("query = %q, want CREATE query", mock.lastQuery)
	}
	if mock.lastDatabase != "testdb" {
		t.Errorf("database = %q, want %q", mock.lastDatabase, "testdb")
	}
	if mock.lastParams["name"] != "alice" {
		t.Errorf("params[name] = %v, want alice", mock.lastParams["name"])
	}
	if len(res.records) != 1 {
		t.Fatalf("records = %d, want 1", len(res.records))
	}
}

func TestDriver_Exec_RunnerError(t *testing.T) {
	mock := &mockRunner{writeErr: errors.New("connection refused")}
	d := &Driver{runner: mock, database: "testdb"}
	var res Response
	err := d.Exec(context.Background(), "CREATE (n:User) RETURN n", map[string]any{}, &res)
	if err == nil {
		t.Fatal("Exec() should return error when runner fails")
	}
}

func TestDriver_Query_Success(t *testing.T) {
	mock := &mockRunner{
		readRecords: []*ndriver.Record{
			{Keys: []string{"n"}, Values: []any{map[string]any{"id": "ksuid-1", "name": "alice"}}},
			{Keys: []string{"n"}, Values: []any{map[string]any{"id": "ksuid-2", "name": "bob"}}},
		},
	}
	d := &Driver{runner: mock, database: "mydb"}
	var res Response
	params := map[string]any{"label": "User"}
	err := d.Query(context.Background(), "MATCH (n:User) RETURN n", params, &res)
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if mock.lastQuery != "MATCH (n:User) RETURN n" {
		t.Errorf("query = %q, want MATCH query", mock.lastQuery)
	}
	if mock.lastDatabase != "mydb" {
		t.Errorf("database = %q, want %q", mock.lastDatabase, "mydb")
	}
	if len(res.records) != 2 {
		t.Fatalf("records = %d, want 2", len(res.records))
	}
}

func TestDriver_Query_RunnerError(t *testing.T) {
	mock := &mockRunner{readErr: errors.New("timeout")}
	d := &Driver{runner: mock, database: "testdb"}
	var res Response
	err := d.Query(context.Background(), "MATCH (n) RETURN n", map[string]any{}, &res)
	if err == nil {
		t.Fatal("Query() should return error when runner fails")
	}
}

func TestDriver_Close_Success(t *testing.T) {
	mock := &mockRunner{}
	d := &Driver{runner: mock, database: "testdb"}
	err := d.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !mock.closed {
		t.Error("Close() should call runner.close()")
	}
}

func TestDriver_Close_Error(t *testing.T) {
	mock := &mockRunner{closeErr: errors.New("close failed")}
	d := &Driver{runner: mock, database: "testdb"}
	err := d.Close()
	if err == nil {
		t.Fatal("Close() should propagate runner error")
	}
}

func TestDriver_Exec_EmptyParams(t *testing.T) {
	mock := &mockRunner{writeRecords: []*ndriver.Record{}}
	d := &Driver{runner: mock, database: "testdb"}
	var res Response
	err := d.Exec(context.Background(), "CREATE (n:Label) RETURN n", map[string]any{}, &res)
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	if len(res.records) != 0 {
		t.Errorf("records = %d, want 0", len(res.records))
	}
}

func TestDriver_Query_EmptyResult(t *testing.T) {
	mock := &mockRunner{readRecords: []*ndriver.Record{}}
	d := &Driver{runner: mock, database: "testdb"}
	var res Response
	err := d.Query(context.Background(), "MATCH (n:NoExist) RETURN n", map[string]any{}, &res)
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if len(res.records) != 0 {
		t.Errorf("records = %d, want 0", len(res.records))
	}
}

func TestNeo4jRunner_Close(t *testing.T) {
	mock := &mockNeo4jDB{}
	runner := &neo4jRunner{db: mock}
	err := runner.close(context.Background())
	if err != nil {
		t.Fatalf("close() error = %v", err)
	}
	if !mock.closeCalled {
		t.Error("close() should call db.Close()")
	}
}

func TestNeo4jRunner_Close_Error(t *testing.T) {
	mock := &mockNeo4jDB{closeErr: errors.New("close failed")}
	runner := &neo4jRunner{db: mock}
	err := runner.close(context.Background())
	if err == nil {
		t.Fatal("close() should propagate error")
	}
}
