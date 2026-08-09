package infra

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"
)

// TestMigrateUp_AcceptsLibpqKeyValueDSN guards the DSN syntax gap that broke a
// staging deploy: every other test feeds MigrateUp the testcontainer's URL-form
// connection string, but the rendered config.toml historically used the libpq
// key-value form, which migrate's URL-based constructor rejects outright with
// "failed to parse scheme from database URL: no scheme". GORM accepts both, so
// the app started fine and only `markpost migrate up` failed.
func TestMigrateUp_AcceptsLibpqKeyValueDSN(t *testing.T) {
	_ = SetupTestDB(t)
	ctx := context.Background()

	urlDSN, err := testPGContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("dsn: %v", err)
	}
	u, err := url.Parse(urlDSN)
	if err != nil {
		t.Fatalf("parse dsn: %v", err)
	}
	password, _ := u.User.Password()
	keyValueDSN := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		u.Hostname(), u.Port(), u.User.Username(), password,
		strings.TrimPrefix(u.Path, "/"),
	)

	if err := MigrateUp(keyValueDSN); err != nil {
		t.Fatalf("MigrateUp with libpq key-value DSN: %v", err)
	}

	version, dirty, err := MigrateVersion(keyValueDSN)
	if err != nil {
		t.Fatalf("MigrateVersion with libpq key-value DSN: %v", err)
	}
	if dirty {
		t.Fatal("migrations left the schema dirty")
	}
	if version == 0 {
		t.Fatal("expected a non-zero migration version")
	}
}
