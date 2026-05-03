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

// Package secretkey enumerates the keys the operator stores inside
// generated Kubernetes Secrets and offers helpers to read them safely.
//
// The data layout is intentionally compatible with libpq's environment
// variables so that consumers can mount a Secret straight into a pod and
// connect with no extra configuration.
package secretkey

import "fmt"

// Standard libpq-aligned keys.
const (
	// Username is the PostgreSQL role to authenticate as.
	Username = "username"
	// Password is the password matching Username.
	Password = "password"
	// Host is the cluster's read-write Service FQDN.
	Host = "host"
	// Port is the TCP port the host listens on.
	Port = "port"
	// Database is the default database for the role.
	Database = "database"
	// URI is a libpq-style connection URI containing all of the above.
	URI = "uri"
	// JDBCURL is a JDBC-style connection URL for Java consumers.
	JDBCURL = "jdbc-url"

	// CACertificate is the PEM-encoded cluster CA certificate.
	CACertificate = "ca.crt"
	// ClientCertificate is the PEM-encoded client certificate.
	ClientCertificate = "tls.crt"
	// ClientKey is the PEM-encoded client private key.
	ClientKey = "tls.key"
)

// Required keys for a fully populated credential secret.
var Required = []string{Username, Password, Host, Port, Database, URI}

// Optional keys present only on TLS-enabled clusters.
var Optional = []string{JDBCURL, CACertificate, ClientCertificate, ClientKey}

// Has reports whether data has all of the required keys.
func Has(data map[string][]byte, required ...string) bool {
	keys := required
	if len(keys) == 0 {
		keys = Required
	}
	for _, k := range keys {
		v, ok := data[k]
		if !ok || len(v) == 0 {
			return false
		}
	}
	return true
}

// MissingKeys returns the keys from required that are missing or empty in data.
func MissingKeys(data map[string][]byte, required ...string) []string {
	keys := required
	if len(keys) == 0 {
		keys = Required
	}
	var out []string
	for _, k := range keys {
		if v, ok := data[k]; !ok || len(v) == 0 {
			out = append(out, k)
		}
	}
	return out
}

// String returns the value at key as a string. The bool return is false
// when the key is absent.
func String(data map[string][]byte, key string) (string, bool) {
	v, ok := data[key]
	if !ok {
		return "", false
	}
	return string(v), true
}

// MustString panics if the key is absent. Intended for unit tests; do not
// use it on the controller's hot path.
func MustString(data map[string][]byte, key string) string {
	v, ok := String(data, key)
	if !ok {
		panic(fmt.Sprintf("secretkey: required key %q is missing", key))
	}
	return v
}

// JDBCFromLibpq builds a JDBC URL from libpq-style fields.
func JDBCFromLibpq(host, port, database, user, password string) string {
	url := fmt.Sprintf("jdbc:postgresql://%s:%s/%s?user=%s", host, port, database, user)
	if password != "" {
		url += "&password=" + password
	}
	return url
}

// LibpqURI builds a libpq-style connection URI from the same components.
func LibpqURI(host, port, database, user, password string) string {
	if password == "" {
		return fmt.Sprintf("postgresql://%s@%s:%s/%s", user, host, port, database)
	}
	return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", user, password, host, port, database)
}
