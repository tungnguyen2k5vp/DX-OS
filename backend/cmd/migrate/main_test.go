package main

import "testing"

func TestMigrationChecksumDetectsContentChanges(t *testing.T) {
	original := migrationChecksum([]byte("CREATE TABLE example (id uuid);"))
	repeated := migrationChecksum([]byte("CREATE TABLE example (id uuid);"))
	modified := migrationChecksum([]byte("CREATE TABLE example (id text);"))

	if len(original) != 64 || original != repeated {
		t.Fatalf("expected a stable SHA-256 checksum, got %q and %q", original, repeated)
	}
	if original == modified {
		t.Fatal("expected modified migration content to produce a different checksum")
	}
}
