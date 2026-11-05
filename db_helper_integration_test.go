//go:build dbintegration

package forum

import (
	"testing"

	"github.com/kosatka-cms/kosatka/modulekit"
	"github.com/kosatka-cms/kosatka/modulekit/testkit"
)

func seedRow(t *testing.T, db testkit.DB, insert string) {
	t.Helper()
	if _, err := db.DB.Exec(insert); err != nil {
		t.Fatal(err)
	}
}

func seedUser(t *testing.T, db testkit.DB) {
	seedRow(t, db, `INSERT INTO users (id, username, slug, password_hash) VALUES (1, 'alice', 'alice', '')`)
}

// newForumRepository - the forum's repository on the test database, with
// the core's nodes, users and taxonomy (d - testkit.Deps of it).
func newForumRepository(d modulekit.Deps) *EntForumRepository {
	return NewForumRepository(d.DB, d.NodeRepo, d.Users, d.Taxonomy)
}
