//go:build dbintegration

package manifest_test

import (
	"context"
	"encoding/json"
	"testing"

	// The module registers itself (init): its set is installed with the
	// others.
	_ "github.com/kosatka-cms/kosatka-module-forum"
	"github.com/kosatka-cms/kosatka-module-forum/manifest"
	"github.com/kosatka-cms/kosatka/modulekit/manifestkit"
	"github.com/kosatka-cms/kosatka/modulekit/testkit"
)

// TestGrantCreateTopics - the role of every signed-in user gets "create
// forum content", its other grants kept; run again, nothing changes.
func TestGrantCreateTopics(t *testing.T) {
	testkit.SchemaDB(t, func(t *testing.T, db testkit.DB) {
		ctx := context.Background()
		if _, err := db.DB.ExecContext(ctx, db.Dialect.Rebind("DELETE FROM roles WHERE name = ?"), manifestkit.RoleUser); err != nil {
			t.Fatal(err)
		}
		env := manifestkit.InstallEnv{DB: db.DB, Dialect: db.Dialect}
		if err := manifest.GrantCreateTopics(ctx, env); err != nil { // no such role: nothing to do
			t.Fatal(err)
		}
		if _, err := db.DB.ExecContext(ctx, db.Dialect.Rebind(`INSERT INTO roles (name, description, permissions, created_at, updated_at)
			VALUES (?, '', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`), manifestkit.RoleUser, `{"edit own forum content": true}`); err != nil {
			t.Fatal(err)
		}
		for range 2 {
			if err := manifest.GrantCreateTopics(ctx, env); err != nil {
				t.Fatal(err)
			}
		}
		var raw string
		if err := db.DB.QueryRowContext(ctx, db.Dialect.Rebind("SELECT permissions FROM roles WHERE name = ?"), manifestkit.RoleUser).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		var perms map[string]bool
		if err := json.Unmarshal([]byte(raw), &perms); err != nil {
			t.Fatal(err)
		}
		if !perms["create forum content"] || !perms["edit own forum content"] || len(perms) != 2 {
			t.Errorf("permissions: %v", perms)
		}
	})
}

// TestInstallGrantsCreateTopics - installing the forum (a new site, the
// Modules page) lets signed-in users start topics: the data step that
// grants it is only recorded on install, so the install step does it.
func TestInstallGrantsCreateTopics(t *testing.T) {
	// SchemaDB installs every set, the forum's with its install step.
	testkit.SchemaDB(t, func(t *testing.T, db testkit.DB) {
		ctx := context.Background()
		var raw string
		if err := db.DB.QueryRowContext(ctx, db.Dialect.Rebind("SELECT permissions FROM roles WHERE name = ?"), manifestkit.RoleUser).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		var perms map[string]bool
		if err := json.Unmarshal([]byte(raw), &perms); err != nil {
			t.Fatal(err)
		}
		if !perms["create forum content"] {
			t.Errorf("the %s role after install: %v", manifestkit.RoleUser, perms)
		}
	})
}
