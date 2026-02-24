// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package neo4j

import (
	"encoding/json"
	"errors"
	"fmt"

	ndriver "github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

// Response wraps collected Neo4j records with typed read methods.
type Response struct {
	records []*ndriver.Record
	columns []string
}

// NewResponse creates a Response from a set of records and column names.
func NewResponse(records []*ndriver.Record, columns []string) *Response {
	return &Response{records: records, columns: columns}
}

// ReadInt extracts an integer result, typically from RETURN count(n) queries.
func (r *Response) ReadInt() (int, error) {
	if len(r.records) == 0 {
		return 0, errors.New("neo4j: no records in response")
	}
	if len(r.records[0].Values) == 0 {
		return 0, errors.New("neo4j: record has no values")
	}
	val := r.records[0].Values[0]
	n, ok := val.(int64)
	if !ok {
		return 0, fmt.Errorf("neo4j: expected int64, got %T", val)
	}
	return int(n), nil
}

// ReadBool extracts a boolean result, typically from RETURN exists(...) queries.
func (r *Response) ReadBool() (bool, error) {
	if len(r.records) == 0 {
		return false, errors.New("neo4j: no records in response")
	}
	if len(r.records[0].Values) == 0 {
		return false, errors.New("neo4j: record has no values")
	}
	val := r.records[0].Values[0]
	b, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("neo4j: expected bool, got %T", val)
	}
	return b, nil
}

// ReadNodeMaps extracts node property maps from all records.
func (r *Response) ReadNodeMaps() ([]map[string]any, error) {
	if r.records == nil {
		return nil, errors.New("neo4j: nil records in response")
	}
	maps := make([]map[string]any, 0, len(r.records))
	for _, rec := range r.records {
		if len(rec.Values) == 0 {
			return nil, errors.New("neo4j: record has no values")
		}
		m, ok := rec.Values[0].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("neo4j: expected map[string]any, got %T", rec.Values[0])
		}
		maps = append(maps, m)
	}
	return maps, nil
}

// ReadSingle extracts a single node's property map from the first record.
func (r *Response) ReadSingle() (map[string]any, error) {
	if len(r.records) == 0 {
		return nil, errors.New("neo4j: no records in response")
	}
	if len(r.records[0].Values) == 0 {
		return nil, errors.New("neo4j: record has no values")
	}
	m, ok := r.records[0].Values[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("neo4j: expected map[string]any, got %T", r.records[0].Values[0])
	}
	return m, nil
}

// Scan reads all records into v by building maps from each record's
// Keys/Values pairs and decoding via JSON round-trip. v should be a
// pointer to a slice (e.g. *[]struct{...}) or any JSON-decodable type.
func (r *Response) Scan(v any) error {
	if r.records == nil {
		return errors.New("neo4j: nil records in response")
	}
	rows := make([]map[string]any, 0, len(r.records))
	for _, rec := range r.records {
		row := make(map[string]any, len(rec.Keys))
		for i, key := range rec.Keys {
			if i < len(rec.Values) {
				row[key] = rec.Values[i]
			}
		}
		rows = append(rows, row)
	}
	data, err := json.Marshal(rows)
	if err != nil {
		return fmt.Errorf("neo4j: marshal response: %w", err)
	}
	return json.Unmarshal(data, v)
}

// DecodeJSONField decodes a raw value from Neo4j (typically []interface{})
// into the target type via JSON round-trip. Used by generated FromResponse
// methods for slice/JSON fields where the Neo4j driver returns []interface{}
// instead of typed Go slices.
//
// Example usage in generated code:
//
//	if v, ok := m["vector"]; ok {
//	    if err := neo4j.DecodeJSONField(v, &tc.Vector); err != nil {
//	        return err
//	    }
//	}
func DecodeJSONField(raw any, target any) error {
	if raw == nil {
		return nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("neo4j: marshal field: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("neo4j: unmarshal field: %w", err)
	}
	return nil
}
