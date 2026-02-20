// Copyright 2019-present Facebook Inc. All rights reserved.
// This source code is licensed under the Apache 2.0 license found
// in the LICENSE file in the root directory of this source tree.

package neo4j

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	ndriver "github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

// Config holds connection parameters for a Neo4j database.
type Config struct {
	URI      string // bolt://host:7687, neo4j://host:7687, etc.
	Username string
	Password string
	Database string // default: "neo4j"
}

// Build creates a new Driver from the Config.
func (cfg Config) Build() (*Driver, error) {
	database := cfg.Database
	if database == "" {
		database = "neo4j"
	}
	db, err := ndriver.NewDriver(cfg.URI, ndriver.BasicAuth(cfg.Username, cfg.Password, ""))
	if err != nil {
		return nil, fmt.Errorf("neo4j: build driver: %w", err)
	}
	return NewDriver(db, database), nil
}

// ParseURI extracts a Config from a connection URI string.
// Format: bolt://user:pass@host:7687/dbname
func ParseURI(uri string) (Config, error) {
	if uri == "" {
		return Config{}, errors.New("neo4j: empty URI")
	}
	u, err := url.Parse(uri)
	if err != nil || u.Host == "" {
		return Config{}, fmt.Errorf("neo4j: invalid URI %q", uri)
	}
	var username, password string
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
	}
	database := strings.TrimPrefix(u.Path, "/")
	if database == "" {
		database = "neo4j"
	}
	cleanURI := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	return Config{
		URI:      cleanURI,
		Username: username,
		Password: password,
		Database: database,
	}, nil
}
