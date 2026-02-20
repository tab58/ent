// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package neo4j

import (
	"testing"

	ndriver "github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

func TestNewResponse(t *testing.T) {
	r := NewResponse(nil, nil)
	if r == nil {
		t.Fatal("NewResponse returned nil")
	}
}

func TestResponse_ReadInt(t *testing.T) {
	tests := []struct {
		name    string
		records []*ndriver.Record
		want    int
		wantErr bool
	}{
		{
			name:    "nil records returns error",
			records: nil,
			wantErr: true,
		},
		{
			name:    "empty records returns error",
			records: []*ndriver.Record{},
			wantErr: true,
		},
		{
			name: "single record with int64 value",
			records: []*ndriver.Record{
				{
					Keys:   []string{"count(n)"},
					Values: []any{int64(42)},
				},
			},
			want: 42,
		},
		{
			name: "single record with zero",
			records: []*ndriver.Record{
				{
					Keys:   []string{"count(n)"},
					Values: []any{int64(0)},
				},
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewResponse(tt.records, []string{"count(n)"})
			got, err := r.ReadInt()
			if (err != nil) != tt.wantErr {
				t.Fatalf("ReadInt() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ReadInt() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestResponse_ReadBool(t *testing.T) {
	tests := []struct {
		name    string
		records []*ndriver.Record
		want    bool
		wantErr bool
	}{
		{
			name:    "nil records returns error",
			records: nil,
			wantErr: true,
		},
		{
			name: "single record with true",
			records: []*ndriver.Record{
				{
					Keys:   []string{"exists"},
					Values: []any{true},
				},
			},
			want: true,
		},
		{
			name: "single record with false",
			records: []*ndriver.Record{
				{
					Keys:   []string{"exists"},
					Values: []any{false},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewResponse(tt.records, []string{"exists"})
			got, err := r.ReadBool()
			if (err != nil) != tt.wantErr {
				t.Fatalf("ReadBool() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ReadBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResponse_ReadNodeMaps(t *testing.T) {
	tests := []struct {
		name    string
		records []*ndriver.Record
		want    int // number of maps expected
		wantErr bool
	}{
		{
			name:    "nil records returns error",
			records: nil,
			wantErr: true,
		},
		{
			name:    "empty records returns empty slice",
			records: []*ndriver.Record{},
			want:    0,
		},
		{
			name: "multiple records with map values",
			records: []*ndriver.Record{
				{
					Keys:   []string{"n"},
					Values: []any{map[string]any{"id": "ksuid-1", "name": "alice"}},
				},
				{
					Keys:   []string{"n"},
					Values: []any{map[string]any{"id": "ksuid-2", "name": "bob"}},
				},
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewResponse(tt.records, []string{"n"})
			got, err := r.ReadNodeMaps()
			if (err != nil) != tt.wantErr {
				t.Fatalf("ReadNodeMaps() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(got) != tt.want {
				t.Errorf("ReadNodeMaps() returned %d maps, want %d", len(got), tt.want)
			}
		})
	}
}

func TestResponse_ReadSingle(t *testing.T) {
	tests := []struct {
		name    string
		records []*ndriver.Record
		wantID  string
		wantErr bool
	}{
		{
			name:    "nil records returns error",
			records: nil,
			wantErr: true,
		},
		{
			name:    "empty records returns error",
			records: []*ndriver.Record{},
			wantErr: true,
		},
		{
			name: "single record returns map",
			records: []*ndriver.Record{
				{
					Keys:   []string{"n"},
					Values: []any{map[string]any{"id": "ksuid-1", "name": "alice"}},
				},
			},
			wantID: "ksuid-1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewResponse(tt.records, []string{"n"})
			got, err := r.ReadSingle()
			if (err != nil) != tt.wantErr {
				t.Fatalf("ReadSingle() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				id, ok := got["id"].(string)
				if !ok || id != tt.wantID {
					t.Errorf("ReadSingle()[id] = %v, want %q", got["id"], tt.wantID)
				}
			}
		})
	}
}

func TestResponse_ReadInt_WrongType(t *testing.T) {
	r := NewResponse([]*ndriver.Record{
		{Keys: []string{"val"}, Values: []any{"not-an-int"}},
	}, []string{"val"})
	_, err := r.ReadInt()
	if err == nil {
		t.Error("ReadInt() with string value should return error")
	}
}

func TestResponse_ReadBool_WrongType(t *testing.T) {
	r := NewResponse([]*ndriver.Record{
		{Keys: []string{"val"}, Values: []any{int64(1)}},
	}, []string{"val"})
	_, err := r.ReadBool()
	if err == nil {
		t.Error("ReadBool() with int64 value should return error")
	}
}

func TestResponse_ReadNodeMaps_WrongType(t *testing.T) {
	r := NewResponse([]*ndriver.Record{
		{Keys: []string{"n"}, Values: []any{"not-a-map"}},
	}, []string{"n"})
	_, err := r.ReadNodeMaps()
	if err == nil {
		t.Error("ReadNodeMaps() with string value should return error")
	}
}

func TestResponse_ReadSingle_WrongType(t *testing.T) {
	r := NewResponse([]*ndriver.Record{
		{Keys: []string{"n"}, Values: []any{42}},
	}, []string{"n"})
	_, err := r.ReadSingle()
	if err == nil {
		t.Error("ReadSingle() with int value should return error")
	}
}

func TestResponse_ReadInt_EmptyValues(t *testing.T) {
	r := NewResponse([]*ndriver.Record{
		{Keys: []string{"val"}, Values: []any{}},
	}, []string{"val"})
	_, err := r.ReadInt()
	if err == nil {
		t.Error("ReadInt() with empty Values should return error")
	}
}

func TestResponse_ReadBool_EmptyValues(t *testing.T) {
	r := NewResponse([]*ndriver.Record{
		{Keys: []string{"val"}, Values: []any{}},
	}, []string{"val"})
	_, err := r.ReadBool()
	if err == nil {
		t.Error("ReadBool() with empty Values should return error")
	}
}

func TestResponse_Scan(t *testing.T) {
	tests := []struct {
		name    string
		records []*ndriver.Record
		wantLen int
		wantErr bool
	}{
		{
			name:    "nil records returns error",
			records: nil,
			wantErr: true,
		},
		{
			name:    "empty records returns empty slice",
			records: []*ndriver.Record{},
			wantLen: 0,
		},
		{
			name: "single record with keyed values",
			records: []*ndriver.Record{
				{Keys: []string{"name", "age"}, Values: []any{"alice", int64(30)}},
			},
			wantLen: 1,
		},
		{
			name: "multiple records",
			records: []*ndriver.Record{
				{Keys: []string{"name"}, Values: []any{"alice"}},
				{Keys: []string{"name"}, Values: []any{"bob"}},
			},
			wantLen: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewResponse(tt.records, nil)
			var result []map[string]any
			err := r.Scan(&result)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Scan() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(result) != tt.wantLen {
				t.Errorf("Scan() returned %d rows, want %d", len(result), tt.wantLen)
			}
		})
	}
}

func TestResponse_Scan_StructDecode(t *testing.T) {
	type row struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	records := []*ndriver.Record{
		{Keys: []string{"name", "age"}, Values: []any{"alice", float64(30)}},
		{Keys: []string{"name", "age"}, Values: []any{"bob", float64(25)}},
	}
	r := NewResponse(records, nil)
	var result []row
	if err := r.Scan(&result); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("Scan() returned %d rows, want 2", len(result))
	}
	if result[0].Name != "alice" {
		t.Errorf("result[0].Name = %q, want alice", result[0].Name)
	}
	if result[1].Age != 25 {
		t.Errorf("result[1].Age = %d, want 25", result[1].Age)
	}
}

func TestResponse_Scan_KeysValuesMismatch(t *testing.T) {
	// More Keys than Values — should not panic.
	records := []*ndriver.Record{
		{Keys: []string{"name", "age", "email"}, Values: []any{"alice", float64(30)}},
	}
	r := NewResponse(records, nil)
	var result []map[string]any
	if err := r.Scan(&result); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("Scan() returned %d rows, want 1", len(result))
	}
	if _, ok := result[0]["email"]; ok {
		t.Error("email key should not be present when Values is shorter than Keys")
	}
}

func TestResponse_ReadNodeMaps_PropertyExtraction(t *testing.T) {
	// When records contain node maps, ReadNodeMaps should extract
	// the property maps correctly.
	records := []*ndriver.Record{
		{
			Keys: []string{"n"},
			Values: []any{map[string]any{
				"id":    "ksuid-1",
				"name":  "alice",
				"age":   int64(30),
				"email": "alice@example.com",
			}},
		},
	}
	r := NewResponse(records, []string{"n"})
	maps, err := r.ReadNodeMaps()
	if err != nil {
		t.Fatalf("ReadNodeMaps() error = %v", err)
	}
	if len(maps) != 1 {
		t.Fatalf("ReadNodeMaps() returned %d maps, want 1", len(maps))
	}
	m := maps[0]
	if m["name"] != "alice" {
		t.Errorf("name = %v, want alice", m["name"])
	}
	if m["age"] != int64(30) {
		t.Errorf("age = %v, want 30", m["age"])
	}
}
