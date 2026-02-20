// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package neo4j

import (
	"testing"
)

func TestParseURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		wantCfg  Config
		wantErr  bool
	}{
		{
			name: "bolt URI with credentials and database",
			uri:  "bolt://neo4j:password@localhost:7687/mydb",
			wantCfg: Config{
				URI:      "bolt://localhost:7687",
				Username: "neo4j",
				Password: "password",
				Database: "mydb",
			},
		},
		{
			name: "neo4j URI with credentials",
			uri:  "neo4j://admin:secret@db.example.com:7687/production",
			wantCfg: Config{
				URI:      "neo4j://db.example.com:7687",
				Username: "admin",
				Password: "secret",
				Database: "production",
			},
		},
		{
			name: "bolt URI without database defaults to neo4j",
			uri:  "bolt://neo4j:password@localhost:7687",
			wantCfg: Config{
				URI:      "bolt://localhost:7687",
				Username: "neo4j",
				Password: "password",
				Database: "neo4j",
			},
		},
		{
			name:    "invalid URI",
			uri:     "://invalid",
			wantErr: true,
		},
		{
			name:    "empty URI",
			uri:     "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseURI(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseURI(%q) error = %v, wantErr %v", tt.uri, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if cfg.URI != tt.wantCfg.URI {
				t.Errorf("URI = %q, want %q", cfg.URI, tt.wantCfg.URI)
			}
			if cfg.Username != tt.wantCfg.Username {
				t.Errorf("Username = %q, want %q", cfg.Username, tt.wantCfg.Username)
			}
			if cfg.Password != tt.wantCfg.Password {
				t.Errorf("Password = %q, want %q", cfg.Password, tt.wantCfg.Password)
			}
			if cfg.Database != tt.wantCfg.Database {
				t.Errorf("Database = %q, want %q", cfg.Database, tt.wantCfg.Database)
			}
		})
	}
}

func TestConfig_Build_DefaultDatabase(t *testing.T) {
	// Config with empty Database should default to "neo4j".
	cfg := Config{
		URI:      "bolt://localhost:7687",
		Username: "neo4j",
		Password: "test",
	}
	drv, err := cfg.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if drv == nil {
		t.Fatal("Build() returned nil driver")
	}
	if drv.database != "neo4j" {
		t.Errorf("database = %q, want %q", drv.database, "neo4j")
	}
}

func TestConfig_Build_CustomDatabase(t *testing.T) {
	cfg := Config{
		URI:      "bolt://localhost:7687",
		Username: "neo4j",
		Password: "test",
		Database: "mydb",
	}
	drv, err := cfg.Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if drv == nil {
		t.Fatal("Build() returned nil driver")
	}
	if drv.database != "mydb" {
		t.Errorf("database = %q, want %q", drv.database, "mydb")
	}
}

func TestConfig_Build_InvalidURI(t *testing.T) {
	cfg := Config{
		URI:      "not-a-valid-uri",
		Username: "neo4j",
		Password: "test",
	}
	_, err := cfg.Build()
	if err == nil {
		t.Error("Build() with invalid URI should return an error")
	}
}
