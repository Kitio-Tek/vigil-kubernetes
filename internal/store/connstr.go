/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package store provides utilities for constructing and parsing PostgreSQL
// connection strings (libpq keyword/value format and URI format) used
// throughout the Athos operator.
package store

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

const (
	defaultPort     = 5432
	defaultSSLMode  = "require"
	defaultUser     = "postgres"
	defaultDatabase = "postgres"
)

// ConnParams holds the individual components of a PostgreSQL connection.
type ConnParams struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string

	// Extra holds additional libpq parameters (e.g. connect_timeout, sslrootcert).
	Extra map[string]string
}

// DefaultConnParams returns a ConnParams with sensible operator defaults.
func DefaultConnParams(host string) ConnParams {
	return ConnParams{
		Host:     host,
		Port:     defaultPort,
		User:     defaultUser,
		Database: defaultDatabase,
		SSLMode:  defaultSSLMode,
	}
}

// DSN returns the libpq keyword/value connection string.
// Empty fields are omitted; password is included only when non-empty.
func (c ConnParams) DSN() string {
	kv := c.baseKV()
	return kvString(kv)
}

// URI returns the connection string in postgres:// URI format.
func (c ConnParams) URI() string {
	u := &url.URL{
		Scheme: "postgresql",
		Host:   fmt.Sprintf("%s:%d", c.Host, c.port()),
		Path:   "/" + url.PathEscape(c.database()),
	}

	userInfo := c.user()
	if c.Password != "" {
		u.User = url.UserPassword(userInfo, c.Password)
	} else {
		u.User = url.User(userInfo)
	}

	q := url.Values{}
	q.Set("sslmode", c.sslMode())
	for k, v := range c.Extra {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	return u.String()
}

// WithDatabase returns a copy of ConnParams with the database field replaced.
func (c ConnParams) WithDatabase(db string) ConnParams {
	cp := c
	cp.Database = db
	return cp
}

// WithUser returns a copy of ConnParams with user and password replaced.
func (c ConnParams) WithUser(user, password string) ConnParams {
	cp := c
	cp.User = user
	cp.Password = password
	return cp
}

// WithSSLMode returns a copy of ConnParams with the sslmode replaced.
func (c ConnParams) WithSSLMode(mode string) ConnParams {
	cp := c
	cp.SSLMode = mode
	return cp
}

// WithExtra returns a copy of ConnParams with an additional key/value pair.
func (c ConnParams) WithExtra(key, value string) ConnParams {
	cp := c
	if cp.Extra == nil {
		cp.Extra = make(map[string]string)
	} else {
		newExtra := make(map[string]string, len(cp.Extra))
		for k, v := range cp.Extra {
			newExtra[k] = v
		}
		cp.Extra = newExtra
	}
	cp.Extra[key] = value
	return cp
}

func (c ConnParams) port() int {
	if c.Port == 0 {
		return defaultPort
	}
	return c.Port
}

func (c ConnParams) user() string {
	if c.User == "" {
		return defaultUser
	}
	return c.User
}

func (c ConnParams) database() string {
	if c.Database == "" {
		return defaultDatabase
	}
	return c.Database
}

func (c ConnParams) sslMode() string {
	if c.SSLMode == "" {
		return defaultSSLMode
	}
	return c.SSLMode
}

func (c ConnParams) baseKV() map[string]string {
	kv := map[string]string{
		"host":    c.Host,
		"port":    fmt.Sprintf("%d", c.port()),
		"user":    c.user(),
		"dbname":  c.database(),
		"sslmode": c.sslMode(),
	}
	if c.Password != "" {
		kv["password"] = c.Password
	}
	for k, v := range c.Extra {
		kv[k] = v
	}
	return kv
}

func kvString(kv map[string]string) string {
	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v := kv[k]
		if strings.ContainsAny(v, " '\\") {
			v = "'" + strings.ReplaceAll(v, "'", "\\'") + "'"
		}
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, " ")
}

// ReplicationDSN returns a connection string targeting the replication
// pseudo-database, suitable for pg_basebackup or streaming replication.
func ReplicationDSN(host string, port int, user, password string) string {
	p := ConnParams{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Database: "replication",
		SSLMode:  defaultSSLMode,
	}
	return p.DSN()
}

// SuperuserDSN returns a DSN for the operator's superuser connection,
// used for administrative queries against the cluster.
func SuperuserDSN(host string, port int, password string) string {
	p := DefaultConnParams(host)
	p.Port = port
	p.Password = password
	return p.DSN()
}

// ParseDSN parses a libpq keyword/value DSN into a ConnParams.
// Only standard fields (host, port, user, password, dbname, sslmode) are
// extracted; unrecognised keys are placed in Extra.
func ParseDSN(dsn string) (ConnParams, error) {
	p := ConnParams{
		Extra: make(map[string]string),
	}
	tokens, err := tokenizeDSN(dsn)
	if err != nil {
		return p, err
	}
	for k, v := range tokens {
		switch k {
		case "host":
			p.Host = v
		case "port":
			_, err := fmt.Sscanf(v, "%d", &p.Port)
			if err != nil {
				return p, fmt.Errorf("invalid port %q: %w", v, err)
			}
		case "user":
			p.User = v
		case "password":
			p.Password = v
		case "dbname":
			p.Database = v
		case "sslmode":
			p.SSLMode = v
		default:
			p.Extra[k] = v
		}
	}
	return p, nil
}

// tokenizeDSN splits a libpq keyword/value DSN into a map.
func tokenizeDSN(dsn string) (map[string]string, error) {
	result := make(map[string]string)
	s := strings.TrimSpace(dsn)
	for s != "" {
		s = strings.TrimLeft(s, " \t\n\r")
		if s == "" {
			break
		}
		eq := strings.IndexByte(s, '=')
		if eq < 0 {
			return nil, fmt.Errorf("malformed DSN near %q: missing '='", s)
		}
		key := strings.TrimSpace(s[:eq])
		s = s[eq+1:]

		var value string
		if strings.HasPrefix(s, "'") {
			// Quoted value.
			end := strings.Index(s[1:], "'")
			if end < 0 {
				return nil, fmt.Errorf("unterminated quoted value for key %q", key)
			}
			value = s[1 : end+1]
			s = s[end+2:]
		} else {
			// Unquoted value ends at next whitespace.
			end := strings.IndexAny(s, " \t\n\r")
			if end < 0 {
				value = s
				s = ""
			} else {
				value = s[:end]
				s = s[end:]
			}
		}
		result[key] = value
	}
	return result, nil
}
