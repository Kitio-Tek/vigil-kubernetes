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

package walarchive_test

import (
	"strings"
	"testing"

	"github.com/Kitio-Tek/vigil-kubernetes/internal/walarchive"
)

func TestEndpoint_URL(t *testing.T) {
	cases := []struct {
		ep   walarchive.Endpoint
		want string
	}{
		{walarchive.Endpoint{Provider: walarchive.ProviderS3, Bucket: "b", Prefix: "p"}, "s3://b/p"},
		{walarchive.Endpoint{Provider: walarchive.ProviderS3, Bucket: "b"}, "s3://b"},
		{walarchive.Endpoint{Provider: walarchive.ProviderGCS, Bucket: "b", Prefix: "/p/"}, "gs://b/p/"},
		{walarchive.Endpoint{Provider: walarchive.ProviderAzure, Bucket: "b", Prefix: ""}, "azure://b"},
		{walarchive.Endpoint{Provider: walarchive.ProviderFile, Path: "/var/wal/"}, "file:///var/wal"},
	}
	for _, tc := range cases {
		if got := tc.ep.URL(); got != tc.want {
			t.Errorf("URL %+v = %q, want %q", tc.ep, got, tc.want)
		}
	}
}

func TestEndpoint_Validate(t *testing.T) {
	if err := (walarchive.Endpoint{Provider: walarchive.ProviderS3}).Validate(); err == nil {
		t.Error("S3 without bucket should fail validation")
	}
	if err := (walarchive.Endpoint{Provider: walarchive.ProviderFile}).Validate(); err == nil {
		t.Error("file without path should fail validation")
	}
	if err := (walarchive.Endpoint{Provider: "unknown"}).Validate(); err == nil {
		t.Error("unknown provider should fail validation")
	}
	if err := (walarchive.Endpoint{Provider: walarchive.ProviderS3, Bucket: "x"}).Validate(); err != nil {
		t.Errorf("valid endpoint failed: %v", err)
	}
}

func TestArchiveCommand_WALG(t *testing.T) {
	cmd, err := walarchive.ArchiveCommand(walarchive.ToolWALG,
		walarchive.Endpoint{Provider: walarchive.ProviderS3, Bucket: "b", Prefix: "p"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(cmd, "wal-g wal-push") || !strings.Contains(cmd, "s3://b/p") {
		t.Errorf("command = %q", cmd)
	}
}

func TestArchiveCommand_Barman(t *testing.T) {
	cmd, err := walarchive.ArchiveCommand(walarchive.ToolBarman,
		walarchive.Endpoint{Provider: walarchive.ProviderS3, Bucket: "b"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(cmd, "barman-cloud-wal-archive") {
		t.Errorf("command = %q", cmd)
	}
}

func TestRestoreCommand_WALG(t *testing.T) {
	cmd, err := walarchive.RestoreCommand(walarchive.ToolWALG,
		walarchive.Endpoint{Provider: walarchive.ProviderS3, Bucket: "b"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(cmd, "wal-fetch") {
		t.Errorf("command = %q", cmd)
	}
}

func TestArchiveCommand_UnknownTool(t *testing.T) {
	if _, err := walarchive.ArchiveCommand("nope",
		walarchive.Endpoint{Provider: walarchive.ProviderS3, Bucket: "b"}); err == nil {
		t.Error("expected error for unknown tool")
	}
}

func TestSegmentFileName(t *testing.T) {
	got := walarchive.SegmentFileName(1, 0, 0)
	want := "000000010000000000000000"
	if got != want {
		t.Errorf("SegmentFileName = %q, want %q", got, want)
	}
}

func TestParseSegmentName_RoundTrip(t *testing.T) {
	want := "00000005000000010000000A"
	tl, l, s, err := walarchive.ParseSegmentName(want)
	if err != nil {
		t.Fatalf("ParseSegmentName: %v", err)
	}
	got := walarchive.SegmentFileName(tl, l, s)
	if got != want {
		t.Errorf("round-trip mismatch: got %q want %q", got, want)
	}
}

func TestParseSegmentName_Invalid(t *testing.T) {
	if _, _, _, err := walarchive.ParseSegmentName("short"); err == nil {
		t.Error("expected error for short name")
	}
	if _, _, _, err := walarchive.ParseSegmentName(strings.Repeat("Z", 24)); err == nil {
		t.Error("expected error for non-hex name")
	}
}

func TestProviderForScheme(t *testing.T) {
	cases := map[string]walarchive.Provider{
		"s3":    walarchive.ProviderS3,
		"gs":    walarchive.ProviderGCS,
		"azure": walarchive.ProviderAzure,
		"file":  walarchive.ProviderFile,
	}
	for scheme, want := range cases {
		got, ok := walarchive.ProviderForScheme(scheme)
		if !ok || got != want {
			t.Errorf("ProviderForScheme(%q) = %v,%v want %v,true", scheme, got, ok, want)
		}
	}
	if _, ok := walarchive.ProviderForScheme("ftp"); ok {
		t.Error("ProviderForScheme should return false for unknown schemes")
	}
}
