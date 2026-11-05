package forum

import (
	"os"
	"testing"

	"github.com/kosatka-cms/kosatka/modulekit/testkit"
)

// TestMain registers the modules, as main does (a topic's path takes the
// forum's path prefix from the registry).
func TestMain(m *testing.M) {
	testkit.RegisterModules()
	os.Exit(m.Run())
}
